package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/opolis/build/parameters"
	"github.com/opolis/build/repo"
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
		parameter := parameters.NewAWSParameterStore(sess)
		token, err := parameter.Get(keyToken)
		if err != nil {
			fmt.Println("error: parameter.Get", err.Error())
			return nil
		}

		fmt.Println("repo:", event.Repository.Name, "ref:", event.Ref)

		// prepare processing dependencies and fire
		stackManager := stack.NewAWSStackManger(sess)
		repo := repo.NewGitHubRepository(event.Repository.Owner.Name, event.Repository.Name, token)

		if err := Process(event, repo, stackManager); err != nil {
			fmt.Println("error processing event:", err.Error())
			return nil
		}
	}

	return nil
}

// Process reacts to GitHub push event writes from the DynamoDB table stream
// and processes them for building. Each incoming event structure is the exact JSON from GitHub.
// We assume we are _only_ receiving push events at this time.
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
func Process(event types.GitHubEvent, repo types.Repository, stackManager types.StackManager) error {
	// fetch stack and parameter files from repoistory
	// pipeline.json - CI/CD pipeline stack spec
	// parameters.json - stack parameters
	// pipelineTemplate, err := repo.Get(event.Ref, "pipeline.json")
	// if err != nil {
	// 	return err
	// }

	// parameterSpec, err := repo.Get(event.Ref, "parameters.json")
	// if err != nil {
	// 	return err
	// }

	// create or update stack with (TODO) ref specific parameters
	stack := stackPipeline(event.Repository.Name, event.Ref)
	fmt.Println(stackManager.Exists(stack))

	return nil
}

//
// Helpers
//

func stackPipeline(repo, ref string) string {
	return fmt.Sprintf("opolis-build-%s-%s-pipeline", repo, parseRef(ref))
}

func parseRef(ref string) string {
	components := strings.Split(ref, "/")
	return components[len(components)-1]
}
