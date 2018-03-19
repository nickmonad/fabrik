package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"net/http"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
)

const (
	// sha1Prefix is the prefix used by GitHub before the HMAC hexdigest.
	sha1Prefix = "sha1"
	// sha256Prefix and sha512Prefix are provided for future compatibility.
	sha256Prefix = "sha256"
	sha512Prefix = "sha512"

	signatureHeader = "X-Hub-Signature"
	eventHeader     = "X-GitHub-Event"
)

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

func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// AWS Session
	sess := session.Must(session.NewSession(&aws.Config{Region: aws.String(endpoints.UsWest2RegionID)}))
	svc := ssm.New(sess)

	// Validate request with HMAC key
	hmacKey, err := svc.GetParameter(&ssm.GetParameterInput{
		Name: aws.String("opolis-build-hmac"), WithDecryption: aws.Bool(true)})

	if err != nil {
		fmt.Println("could not read hmac key: ", err.Error())
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, nil
	}

	signature := request.Headers[signatureHeader]
	err = Validate(signature, []byte(request.Body), []byte(*(hmacKey.Parameter.Value)))
	if err != nil {
		fmt.Println(err.Error())
		return events.APIGatewayProxyResponse{StatusCode: http.StatusUnauthorized}, nil
	}

	fmt.Println("Got Event: ", request.Headers[eventHeader])
	return events.APIGatewayProxyResponse{Body: "ok", StatusCode: 200}, nil
}

func main() {
	lambda.Start(Handler)
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
