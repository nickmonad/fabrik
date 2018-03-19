package main

import (
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

// Handler listens to GitHub event updates from the DynamoDB table stream
// and processes them accordingly. Each incoming event is the exact JSON from GitHub.
// We are interested in three events related to refs. A ref is a generic term
// for either a branch name or tag.
//
// Each stack is parameterized via a parameters.json file in the repo. Each parameter set
// is keyed by 'dev', 'master', and 'release' - corresponding to the CodePipeline instance
// by the same name ('dev' is applied to all non master/tag refs)
//
// Ref Events
// (1) ref create
//     if stack (opolis-build-{repo}-{ref}-pipeline) exists: warn and skip
//     else: create stack
//     if tag: call UpdatePipeline with tag
//     call StartPipeline
//
// (2) ref push
//     if master and stack (opolis-build-{repo}-master-pipeline) not exists: create stack
//     call StartPipeline
//
// (3) ref delete
//     if stack (opolis-build-{repo}-{ref}-pipeline) not exists: warn and skip
//     else: delete stacks
//
func Handler(event events.DynamoDBEvent) error {
	for _, record := range event.Records {
		fmt.Println("Got Event ", record.EventName, " from ", record.EventSource)
	}

	return nil
}

func stackPipeline(repo, ref string) string {
	return fmt.Sprintf("opolis-build-%s-%s-pipeline", repo, ref)
}

func stackDeploy(repo, ref string) string {
	return fmt.Sprintf("%s-%s", repo, ref)
}
