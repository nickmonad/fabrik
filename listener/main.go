package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/opolis/build/parameters"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

const (
	// sha1Prefix is the prefix used by GitHub before the HMAC hexdigest.
	sha1Prefix = "sha1"
	// sha256Prefix and sha512Prefix are provided for future compatibility.
	sha256Prefix = "sha256"
	sha512Prefix = "sha512"
	// HTTP request headers
	signatureHeader = "X-Hub-Signature"
	deliveryHeader  = "X-GitHub-Delivery"
	eventHeader     = "X-GitHub-Event"
)

func main() {
	lambda.Start(Handler)
}

// Handler captures the incoming webhook from GitHub, verifies its integrity with HMAC,
// and pushes the event onto a queue for processing.
func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// AWS session
	sess := session.Must(session.NewSession(
		&aws.Config{Region: aws.String(endpoints.UsWest2RegionID)}))

	// Get HMAC key
	parameter := parameters.NewAWSParameterStore(sess)
	hmacKey, err := parameter.Get("opolis-build-hmac")
	if err != nil {
		fmt.Println("could not read hmac key: ", err.Error())
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, nil
	}

	// Validate the request
	signature := request.Headers[signatureHeader]
	err = Validate(signature, []byte(request.Body), []byte(hmacKey))
	if err != nil {
		fmt.Println(err.Error())
		return events.APIGatewayProxyResponse{StatusCode: http.StatusUnauthorized}, nil
	}

	// Push event into dynamo for further processing
	id := request.Headers[deliveryHeader]
	payload := request.Body
	item := EventItem(os.Getenv("EVENT_TABLE"), id, payload)

	dbService := dynamodb.New(sess)
	_, err = dbService.PutItem(item)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case dynamodb.ErrCodeConditionalCheckFailedException:
				fmt.Println(dynamodb.ErrCodeConditionalCheckFailedException, aerr.Error())
			case dynamodb.ErrCodeProvisionedThroughputExceededException:
				fmt.Println(dynamodb.ErrCodeProvisionedThroughputExceededException, aerr.Error())
			case dynamodb.ErrCodeResourceNotFoundException:
				fmt.Println(dynamodb.ErrCodeResourceNotFoundException, aerr.Error())
			case dynamodb.ErrCodeItemCollectionSizeLimitExceededException:
				fmt.Println(dynamodb.ErrCodeItemCollectionSizeLimitExceededException, aerr.Error())
			case dynamodb.ErrCodeInternalServerError:
				fmt.Println(dynamodb.ErrCodeInternalServerError, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}

		return events.APIGatewayProxyResponse{Body: "event write error", StatusCode: 500}, nil
	}

	return events.APIGatewayProxyResponse{Body: "ok", StatusCode: 200}, nil
}

// Validate returns an error if the signature does not match
// the payload. Returns nil if signature check is successful.
func Validate(sig string, payload, key []byte) error {
	messageMAC, hashFunc, err := messageMAC(sig)
	if err != nil {
		return err
	}

	if !checkMAC(payload, messageMAC, key, hashFunc) {
		return errors.New("signature check failed")
	}

	return nil
}

func EventItem(table, id, payload string) *dynamodb.PutItemInput {
	// trim json payload
	var p string
	buf := new(bytes.Buffer)
	if err := json.Compact(buf, []byte(payload)); err != nil {
		// pass through, no trim
		p = payload
	} else {
		p = buf.String()
	}

	return &dynamodb.PutItemInput{
		TableName: aws.String(table),
		Item: map[string]*dynamodb.AttributeValue{
			"id":        {S: aws.String(id)},
			"timestamp": {S: aws.String(time.Now().String())},
			"payload":   {S: aws.String(p)},
		},
	}
}

//
// HMAC Helpers
// Shamelessly stolen from google/go-github's source.
// Their `ValidatePayload` function expects an incoming http.Request,
// whereas we have API Gateway requests.
//

// genMAC generates the HMAC signature for a message provided
// the secret key and hashFunc.
func genMAC(message, key []byte, hashFunc func() hash.Hash) []byte {
	mac := hmac.New(hashFunc, key)
	mac.Write(message)
	return mac.Sum(nil)
}

// checkMAC reports whether messageMAC is a valid HMAC tag for message.
func checkMAC(message, messageMAC, key []byte, hashFunc func() hash.Hash) bool {
	expectedMAC := genMAC(message, key, hashFunc)
	return hmac.Equal(messageMAC, expectedMAC)
}

// messageMAC returns the hex-decoded HMAC tag from the signature and its
// corresponding hash function.
func messageMAC(signature string) ([]byte, func() hash.Hash, error) {
	if signature == "" {
		return nil, nil, errors.New("missing signature")
	}

	sigParts := strings.SplitN(signature, "=", 2)
	if len(sigParts) != 2 {
		return nil, nil, fmt.Errorf("error parsing signature %q", signature)
	}

	var hashFunc func() hash.Hash
	switch sigParts[0] {
	case sha1Prefix:
		hashFunc = sha1.New
	case sha256Prefix:
		hashFunc = sha256.New
	case sha512Prefix:
		hashFunc = sha512.New
	default:
		return nil, nil, fmt.Errorf("unknown hash type prefix: %q", sigParts[0])
	}

	buf, err := hex.DecodeString(sigParts[1])
	if err != nil {
		return nil, nil, fmt.Errorf("error decoding signature %q: %v", signature, err)
	}

	return buf, hashFunc, nil
}
