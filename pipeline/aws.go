package pipeline

import (
	"errors"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/codepipeline"
)

const (
	ActionNameGitHub = "GitHub"
	StageNameSource  = "Source"
)

type AWSPipelineManager struct {
	client *codepipeline.CodePipeline
}

func NewAWSPipelineManager(session *session.Session) *AWSPipelineManager {
	return &AWSPipelineManager{
		client: codepipeline.New(session),
	}
}

func (m *AWSPipelineManager) GetRepoInfo(name string) (string, string, error) {
	resp, err := m.client.GetPipeline(&codepipeline.GetPipelineInput{
		Name: aws.String(name),
	})

	if err != nil {
		return "", "", err
	}

	for _, stage := range resp.Pipeline.Stages {
		if *(stage.Name) == StageNameSource {
			if len(stage.Actions) == 0 {
				return "", "", errors.New("no source stage actions")
			}

			action := stage.Actions[0]
			return *(action.Configuration["Owner"]), *(action.Configuration["Repo"]), nil
		}
	}

	return "", "", errors.New("source stage not found")
}

func (m *AWSPipelineManager) GetRevision(execId, name string) (string, error) {
	resp, err := m.client.GetPipelineExecution(&codepipeline.GetPipelineExecutionInput{
		PipelineExecutionId: aws.String(execId),
		PipelineName:        aws.String(name),
	})

	if err != nil {
		return "", err
	}

	// Get the current revision
	for _, revision := range resp.PipelineExecution.ArtifactRevisions {
		return *(revision.RevisionId), nil
	}

	return "", errors.New("revision not found")
}

func (m *AWSPipelineManager) JobSuccess(id string) error {
	_, err := m.client.PutJobSuccessResult(&codepipeline.PutJobSuccessResultInput{
		JobId: aws.String(id),
	})

	return err
}

func (m *AWSPipelineManager) JobFailure(id, message string) error {
	_, err := m.client.PutJobFailureResult(&codepipeline.PutJobFailureResultInput{
		JobId: aws.String(id),
		FailureDetails: &codepipeline.FailureDetails{
			Message: aws.String(message),
			Type:    aws.String(codepipeline.FailureTypeJobFailed),
		},
	})

	return err
}
