package stack

import (
	"errors"
	"strings"

	"github.com/opolis/build/types"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/codepipeline"
	log "github.com/sirupsen/logrus"
)

const (
	// AWS sdk code does not make specific distinctions amongst various
	// types of ValidationErrors, other than their message
	// ...so we have to match them
	ErrValidationError = "ValidationError"
	ErrNoUpdate        = "No updates are to be performed."
	ErrDoesNotExist    = "does not exist"
)

type AWSStackManager struct {
	client   *cloudformation.CloudFormation
	pipeline *codepipeline.CodePipeline
	log      *log.Entry
}

func NewAWSStackManger(log *log.Entry, session *session.Session) *AWSStackManager {
	return &AWSStackManager{
		client:   cloudformation.New(session),
		pipeline: codepipeline.New(session),
		log:      log,
	}
}

func (m *AWSStackManager) Create(name string, parameters []types.Parameter, template []byte) error {
	response, err := m.client.CreateStack(&cloudformation.CreateStackInput{
		// Set IAM capabilities
		Capabilities: aws.StringSlice([]string{"CAPABILITY_IAM", "CAPABILITY_NAMED_IAM"}),
		// RoleARN - for ease of development, we are depending on the environment credentials,
		// which are open to all actions
		StackName:    aws.String(name),
		TemplateBody: aws.String(string(template)),
		Parameters:   mapParameters(parameters),
	})

	if err != nil {
		return err
	}

	m.log.Infoln("cloudformation stack create started:", *(response.StackId))
	return nil
}

func (m *AWSStackManager) Update(name string, parameters []types.Parameter, template []byte) error {
	response, err := m.client.UpdateStack(&cloudformation.UpdateStackInput{
		// Set IAM capabilities
		Capabilities: aws.StringSlice([]string{
			cloudformation.CapabilityCapabilityIam,
			cloudformation.CapabilityCapabilityNamedIam,
		}),
		// RoleARN - for ease of development, we are depending on the environment credentials,
		// which are open to all actions
		StackName:    aws.String(name),
		TemplateBody: aws.String(string(template)),
		Parameters:   mapParameters(parameters),
	})

	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == ErrValidationError {
				if strings.Contains(awsErr.Message(), ErrNoUpdate) {
					// stack does not need updating, continue
					return nil
				}
			}
		}

		return err
	}

	m.log.Infoln("cloudformation stack udpate started:", *(response.StackId))
	return nil
}

func (m *AWSStackManager) Delete(name string) error {
	_, err := m.client.DeleteStack(&cloudformation.DeleteStackInput{
		StackName: aws.String(name),
	})

	if err != nil {
		return err
	}

	m.log.Infoln("cloudformation stack delete started:", name)
	return nil
}

func (m *AWSStackManager) Status(name string) (bool, string, error) {
	response, err := m.client.DescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: aws.String(name),
	})

	if err != nil {
		if strings.Contains(err.Error(), ErrDoesNotExist) {
			return false, "", nil
		}

		return false, "ERROR", err
	}

	if len(response.Stacks) == 0 {
		return false, "", nil
	}

	return true, *(response.Stacks[0].StackStatus), nil
}

func (m *AWSStackManager) StartBuild(name string) error {
	response, err := m.pipeline.StartPipelineExecution(&codepipeline.StartPipelineExecutionInput{
		Name: aws.String(name),
	})

	if err != nil {
		return err
	}

	m.log.Infoln("codepipeline execution id:", *(response.PipelineExecutionId))
	return nil
}

func (m *AWSStackManager) UpdateBuild(name, ref string) error {
	return errors.New("not implemented")
}

//
// Helpers
//

// mapParameters - Parameter list to cloudformation.Parameter list
func mapParameters(parameters []types.Parameter) []*cloudformation.Parameter {
	returnParams := make([]*cloudformation.Parameter, 0)
	for _, p := range parameters {
		returnParams = append(returnParams, &cloudformation.Parameter{
			ParameterKey:   aws.String(p.ParameterKey),
			ParameterValue: aws.String(p.ParameterValue),
		})
	}

	return returnParams
}
