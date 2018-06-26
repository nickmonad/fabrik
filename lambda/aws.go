package lambda

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
)

type AWSLambdaManager struct {
	client *lambda.Lambda
}

func NewAWSLambdaManager(session *session.Session) *AWSLambdaManager {
	return &AWSLambdaManager{
		client: lambda.New(session),
	}
}

func (m *AWSLambdaManager) Invoke(name string, payload interface{}) error {
	// encode the payload to JSON
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := m.client.Invoke(&lambda.InvokeInput{
		FunctionName:   aws.String(name),
		InvocationType: aws.String(lambda.InvocationTypeEvent),
		Payload:        encoded,
	})

	if err != nil {
		return err
	}

	if *(resp.StatusCode) != 202 {
		if resp.FunctionError != nil {
			return errors.New(*(resp.FunctionError))
		}

		return errors.New(fmt.Sprintf("lambda invocation resulted in status code: %d", *(resp.StatusCode)))
	}

	return nil
}
