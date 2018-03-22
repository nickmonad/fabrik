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

func (m *AWSPipelineManager) GetRevision(name string) (string, error) {
	resp, err := m.client.GetPipelineState(&codepipeline.GetPipelineStateInput{
		Name: aws.String(name),
	})

	if err != nil {
		return "", err
	}

	// find the Source stage, and pull the current revision
	// from the GitHub action
	for _, stage := range resp.StageStates {
		if *(stage.StageName) == StageNameSource {
			for _, action := range stage.ActionStates {
				if *(action.ActionName) == ActionNameGitHub {
					return *(action.CurrentRevision.RevisionId), nil
				}
			}
		}
	}

	return "", errors.New("source stage not found")
}
