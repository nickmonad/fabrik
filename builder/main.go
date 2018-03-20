package main

import (
	"encoding/json"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

const (
	eventInsert = "INSERT"
)

func main() {
	lambda.Start(Handler)
}

// Repository provides a means to fetch data from
// the version control repository.
type Repository interface {
	Get(ref string, path string) ([]byte, error)
}

// Pipeline provides a means to manage
type Pipeline interface {
	Create()
	Delete()
}

// GitHubEvent
type GitHubEvent struct {
	Ref        string `json:"ref"`
	Repository struct {
		Name string `json:"name"`
	} `json:"repository"`
}

// Handler serves as the integration point between the AWS event and business logic by
// preparing conrete types to satisfy the Handler's interface.
func Handler(event events.DynamoDBEvent) error {
	for _, record := range event.Records {
		fmt.Println("Got Event ", record.EventName, " from ", record.EventSource)

		item := record.Change.NewImage
		rawEvent := []byte(item["payload"].String())

		var event GitHubEvent
		if err := json.Unmarshal(rawEvent, &event); err != nil {
			fmt.Println("error: json.Unmarshal ", err.Error())
			return nil
		}

		fmt.Println("repo:", event.Repository.Name, "ref:", event.Ref)
	}

	return nil
}

// Process reacts to GitHub push event writes from the DynamoDB table stream
// and processes them for building. Each incoming event structure is the exact JSON from GitHub.
// We assume we are _only_ receiving push events at this time.
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
func Process(eventrepo Repository) error {

	return nil
}

//
// Helpers
//

func stackPipeline(repo, ref string) string {
	return fmt.Sprintf("opolis-build-%s-%s-pipeline", repo, ref)
}
