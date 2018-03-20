package stack

import (
	"encoding/json"
	"fmt"

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

func (m *AWSStackManager) Create(name string, template, parameters []byte) error {
	params, err := parseParameters(parameters)
	if err != nil {
		return err
	}

	response, err := m.client.CreateStack(&cloudformation.CreateStackInput{
		// Set IAM capabilities
		Capabilities: aws.StringSlice([]string{"CAPABILITY_IAM", "CAPABILITY_NAMED_IAM"}),
		// RoleARN - for ease of development, we are depending on the environment credentials,
		// which are open to all actions
		StackName:    aws.String(name),
		TemplateBody: aws.String(string(template)),
		Parameters:   params,
	})

	if err != nil {
		return err
	}

	fmt.Println("cloudformation stack create started:", response.StackId)
	return nil
}

func (m *AWSStackManager) Update(name string, template, parameters []byte) error {
	params, err := parseParameters(parameters)
	if err != nil {
		return err
	}

	response, err := m.client.UpdateStack(&cloudformation.UpdateStackInput{
		// Set IAM capabilities
		Capabilities: aws.StringSlice([]string{"CAPABILITY_IAM", "CAPABILITY_NAMED_IAM"}),
		// RoleARN - for ease of development, we are depending on the environment credentials,
		// which are open to all actions
		StackName:    aws.String(name),
		TemplateBody: aws.String(string(template)),
		Parameters:   params,
	})

	if err != nil {
		return err
	}

	fmt.Println("cloudformation stack udpate started:", response.StackId)
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

func (m *AWSStackManager) Exists(name string) (bool, string, error) {
	response, err := m.client.DescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: aws.String(name),
	})

	if err != nil {
		return false, "ERROR", err
	}

	if len(response.Stacks) == 0 {
		return false, "DOES_NOT_EXIST", nil
	}

	return true, *(response.Stacks[0].StackStatus), nil
}

//
// Helpers
//

func parseParameters(parameters []byte) ([]*cloudformation.Parameter, error) {
	var rawParams []types.Parameter
	if err := json.Unmarshal(parameters, &rawParams); err != nil {
		return nil, err
	}

	// map parameter list to cloudformation parameter list
	returnParams := make([]*cloudformation.Parameter, 0)
	for _, raw := range rawParams {
		returnParams = append(returnParams, &cloudformation.Parameter{
			ParameterKey:   aws.String(raw.ParameterKey),
			ParameterValue: aws.String(raw.ParameterValue),
		})
	}

	return returnParams, nil
}
