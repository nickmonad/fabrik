package parameters

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
)

type AWSParameterStore struct {
	client *ssm.SSM
}

func NewAWSParameterStore(session *session.Session) *AWSParameterStore {
	return &AWSParameterStore{
		client: ssm.New(session),
	}
}

func (store *AWSParameterStore) Get(key string) (string, error) {
	resp, err := store.client.GetParameter(&ssm.GetParameterInput{
		Name: aws.String(key), WithDecryption: aws.Bool(true)})

	if err != nil {
		return "", err
	}

	return *(resp.Parameter.Value), nil
}
