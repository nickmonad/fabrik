package secure

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
)

type AWSSecureStore struct {
	client *ssm.SSM
}

func NewAWSSecureStore(session *session.Session) *AWSSecureStore {
	return &AWSSecureStore{
		client: ssm.New(session),
	}
}

func (store *AWSSecureStore) Get(key string) (string, error) {
	resp, err := store.client.GetParameter(&ssm.GetParameterInput{
		Name: aws.String(key), WithDecryption: aws.Bool(true)})

	if err != nil {
		return "", err
	}

	return *(resp.Parameter.Value), nil
}
