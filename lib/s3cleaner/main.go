package main

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"

	log "github.com/sirupsen/logrus"
)

const (
	RequestTypeDelete     = "Delete"
	ResponseStatusSuccess = "SUCCESS"
	ResponseStatusFailed  = "FAILED"
)

type CloudFormationEvent struct {
	RequestId             string          `json:"RequestId"`
	StackId               string          `json:"StackId"`
	RequestType           string          `json:"RequestType"`
	ResourceType          string          `json:"ResourceType"`
	LogicalResourceId     string          `json:"LogicalResourceId"`
	PhysicalResourceId    string          `json:"PhysicalResourceId"`
	ResourceProperties    json.RawMessage `json:"ResourceProperties"`
	OldResourceProperties json.RawMessage `json:"OldResourceProperties"`
	ResponseURL           string          `json:"ResponseURL"`
	ServiceToken          string          `json:"ServiceToken"`
}

type CloudFormationResponse struct {
	Status             string `json:"Status"`
	Reason             string `json:"Reason"`
	StackId            string `json:"StackId"`
	RequestId          string `json:"RequestId"`
	LogicalResourceId  string `json:"LogicalResourceId"`
	PhysicalResourceId string `json:"PhysicalResourceId"`
}

func init() {
	log.SetFormatter(&log.JSONFormatter{DisableTimestamp: true})
}

func main() {
	lambda.Start(Handler)
}

func Handler(event CloudFormationEvent) error {
	defer func() {
		if r := recover(); r != nil {
			log.Errorln("recovered from panic:", r)
		}
	}()

	// AWS session
	sess := session.Must(session.NewSession())
	log := log.WithFields(log.Fields{"stackId": event.StackId})
	logLocation := lambdacontext.LogGroupName + "/" + lambdacontext.LogStreamName

	// prepare required repsonse parameters
	response := CloudFormationResponse{
		StackId:            event.StackId,
		RequestId:          event.RequestId,
		LogicalResourceId:  event.LogicalResourceId,
		PhysicalResourceId: event.PhysicalResourceId,
	}

	if event.RequestType != RequestTypeDelete {
		// ignore non-delete requests
		log.Infoln("ignoring RequestType", event.RequestType)

		response.Status = ResponseStatusSuccess
		response.PhysicalResourceId = logLocation

		return Response(event.ResponseURL, response)
	}

	// parse properties, get bucket name
	var properties map[string]string
	if err := json.Unmarshal(event.ResourceProperties, &properties); err != nil {
		response.Status = ResponseStatusFailed
		response.Reason = logLocation
		log.Errorln("unable to unmarshal resource properties", err.Error())

		return Response(event.ResponseURL, response)
	}

	// delete all objects
	bucketName := properties["Bucket"]
	objects, err := s3.New(sess).ListObjects(&s3.ListObjectsInput{
		Bucket: aws.String(bucketName),
	})

	if err != nil {
		response.Status = ResponseStatusFailed
		response.Reason = logLocation
		log.Errorln("unable to list s3 objects", err.Error())

		return Response(event.ResponseURL, response)
	}

	for _, obj := range objects.Contents {
		_, err := s3.New(sess).DeleteObject(&s3.DeleteObjectInput{
			Bucket: aws.String(bucketName),
			Key:    obj.Key,
		})

		if err != nil {
			response.Status = ResponseStatusFailed
			response.Reason = logLocation
			log.Errorln("unabled to delete object", err.Error())

			return Response(event.ResponseURL, response)
		}
	}

	response.Status = ResponseStatusSuccess
	return Response(event.ResponseURL, response)
}

func Response(url string, response CloudFormationResponse) error {
	payload, err := json.Marshal(response)
	if err != nil {
		return err
	}

	request, err := http.NewRequest("PUT", url, bytes.NewReader(payload))
	if err != nil {
		return err
	}

	_, err = http.DefaultClient.Do(request)
	return err
}
