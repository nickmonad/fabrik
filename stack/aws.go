package stack

import (
	"fmt"
	"strings"

	"github.com/opolis/build/types"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
)

type AWSStackManager struct {
	client *cloudformation.CloudFormation
}

func NewAWSStackManger(session *session.Session) *AWSStackManager {
	return &AWSStackManager{
		client: cloudformation.New(session),
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

	fmt.Println("cloudformation stack create started:", *(response.StackId))
	return nil
}

func (m *AWSStackManager) Update(name string, parameters []types.Parameter, template []byte) error {
	response, err := m.client.UpdateStack(&cloudformation.UpdateStackInput{
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

	fmt.Println("cloudformation stack udpate started:", *(response.StackId))
	return nil
}

func (m *AWSStackManager) Delete(name string) error {
	_, err := m.client.DeleteStack(&cloudformation.DeleteStackInput{
		StackName: aws.String(name),
	})

	if err != nil {
		return err
	}

	fmt.Println("cloudformation stack delete started:", name)
	return nil
}

func (m *AWSStackManager) Status(name string) (bool, string, error) {
	response, err := m.client.DescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: aws.String(name),
	})

	if err != nil {
		// janky solution to determining if stack does not exist,
		// AWS docs on exactly _what_ err this should be when the stack isn't found sucks
		if strings.Contains(err.Error(), "does not exist") {
			return false, "", nil
		}

		return false, "ERROR", err
	}

	if len(response.Stacks) == 0 {
		return false, "", nil
	}

	return true, *(response.Stacks[0].StackStatus), nil
}

//
// Helpers
//

// mapParameters - Parameter list to cloudformation parameter list
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
