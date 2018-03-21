package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/opolis/build/repo"
	"github.com/opolis/build/secure"
	"github.com/opolis/build/stack"
	"github.com/opolis/build/types"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
)

const (
	keyHmac  = "opolis-build-hmac"
	keyToken = "opolis-build-token"
)

var (
	regexCompleted = regexp.MustCompile(`.*_COMPLETE`)
	regexFailed    = regexp.MustCompile(`.*_FAILED`)
)

func main() {
	lambda.Start(Handler)
}

// Handler serves as the integration point between the AWS event and business logic by
// preparing conrete types to satisfy the Handler's interface.
func Handler(event events.DynamoDBEvent) error {
	// AWS session
	sess := session.Must(session.NewSession())

	for _, record := range event.Records {
		// parse github event
		item := record.Change.NewImage
		eventType := item["type"].String()
		rawEvent := []byte(item["payload"].String())

		if eventType != types.EventTypePush {
			fmt.Println("received non-push event:", eventType, "- no action")
			return nil
		}

		var event types.GitHubEvent
		if err := json.Unmarshal(rawEvent, &event); err != nil {
			fmt.Println("error: json.Unmarshal", err.Error())
			return nil
		}

		// fetch secure repo token
		secureStore := secure.NewAWSSecureStore(sess)
		token, err := secureStore.Get(keyToken)
		if err != nil {
			fmt.Println("error: parameter.Get", err.Error())
			return nil
		}

		fmt.Println("repo:", event.Repository.Name, "ref:", event.Ref)

		// prepare processing dependencies and fire
		stackManager := stack.NewAWSStackManger(sess)
		repo := repo.NewGitHubRepository(event.Repository.Owner.Name, event.Repository.Name, token)

		if err := Process(event, repo, stackManager, token); err != nil {
			fmt.Println("error processing event:", err.Error())
			return nil
		}
	}

	return nil
}

// Process reacts to GitHub push event writes from the DynamoDB table stream
// and processes them for building. Each incoming event structure is the exact JSON from GitHub.
// We assume we are _only_ receiving push events at this time.
// Incoming refs are of the form 'ref/{heads|tag}/{value}'
//
// A repository's 'stack' in this context means an infrastructure template (i.e. CloudFormation)
// defining the CI pipeline and build projects.
//
// Each stack is parameterized via a parameters.json file in the repo. Each parameter set
// is keyed by 'dev', 'master', and 'release' - corresponding to the CodePipeline instance
// by the same name ('dev' is applied to all non master/tag refs)
//
//     if ref is tag:
//     		stack = opolis-build-{repo}-release-pipeline
//     if ref = 'master':
//     		stack = opolis-build-{repo}-master-pipeline
//     else:
//     		stack = opolis-build-{repo}-{ref}-pipeline
//
//     if event.deleted:
// 			if not exists(stack): warn and skip
//     		else: delete stack
//          return
//
//     create or update stack with parameters
//     if tag: call UpdatePipeline with tag
//     call StartPipeline
//
func Process(event types.GitHubEvent, repo types.Repository, manager types.StackManager, repoToken string) error {
	// fetch stack and parameter files from repoistory
	// pipeline.json - CI/CD pipeline stack spec
	// parameters.json - stack parameters
	template, err := repo.Get(event.Ref, "pipeline.json")
	if err != nil {
		return err
	}

	parameterSpec, err := repo.Get(event.Ref, "parameters.json")
	if err != nil {
		return err
	}

	parameters, err := parseParameters(parameterSpec)
	if err != nil {
		return err
	}

	// ammend parameter list with required parameters
	// (required parameter by all stacks)
	parameters = addRequiredParameters(parameters, event, repoToken)

	// create or update stack with ref specific parameters
	stack := stackName(event.Repository.Name, event.Ref)
	exists, _, err := manager.Status(stack)
	if err != nil {
		return err
	}

	if !exists {
		err = StackOp(manager.Create, manager, stack, parameters, template)
	} else {
		err = StackOp(manager.Update, manager, stack, parameters, template)
	}

	if err != nil {
		return err
	}

	return nil
}

// StackOp performs the given stack operation (Create or Update), but waits until
// the operation is either completed or failed.
func StackOp(op types.StackOperation, manager types.StackManager, stack string, parameters []types.Parameter, template []byte) error {
	if err := op(stack, parameters, template); err != nil {
		return err
	}

	for {
		_, status, err := manager.Status(stack)
		if err != nil {
			return err
		}

		// continue waiting if stack stauts is neither "completed" or "failed"
		if !(regexCompleted.MatchString(status) || regexFailed.MatchString(status)) {
			fmt.Println("stack status:", status)
			time.Sleep(time.Second)
			continue
		}

		fmt.Println("stack status:", status)
		return nil
	}
}

//
// Helpers
//

func stackName(repo, ref string) string {
	return fmt.Sprintf("opolis-build-pipeline-%s-%s", repo, parseRef(ref))
}

func parseRef(ref string) string {
	components := strings.Split(ref, "/")
	return components[len(components)-1]
}

func parseParameters(parameters []byte) ([]types.Parameter, error) {
	var parsed []types.Parameter
	if err := json.Unmarshal(parameters, &parsed); err != nil {
		return nil, err
	}

	return parsed, nil
}

func addRequiredParameters(params []types.Parameter, event types.GitHubEvent, repoToken string) []types.Parameter {
	required := []types.Parameter{
		types.Parameter{ParameterKey: "RepoOwner", ParameterValue: event.Repository.Owner.Name},
		types.Parameter{ParameterKey: "RepoName", ParameterValue: event.Repository.Name},
		types.Parameter{ParameterKey: "RepoBranch", ParameterValue: parseRef(event.Ref)},
		types.Parameter{ParameterKey: "RepoToken", ParameterValue: repoToken},
	}

	ret := params
	for _, r := range required {
		ret = append(ret, r)
	}

	return ret
}
