package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/opolis/build/repo"
	"github.com/opolis/build/secure"
	"github.com/opolis/build/stack"
	"github.com/opolis/build/types"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/aws/aws-sdk-go/aws/session"
	log "github.com/sirupsen/logrus"
)

var (
	regexCompleted = regexp.MustCompile(`.*_COMPLETE`)
	regexFailed    = regexp.MustCompile(`.*_FAILED`)
)

func init() {
	log.SetFormatter(&log.JSONFormatter{DisableTimestamp: true})
}

func main() {
	lambda.Start(Handler)
}

// Handler serves as the integration point between the AWS event and business logic by
// preparing conrete types to satisfy the Handler's interface.
func Handler(event events.DynamoDBEvent) error {
	defer func() {
		if r := recover(); r != nil {
			log.Errorln("recovered from panic:", r)
		}
	}()

	// AWS session
	sess := session.Must(session.NewSession())

	for _, record := range event.Records {
		// skip modify and remove events from dynamo
		if record.EventName != types.DynamoDBEventInsert {
			log.Warnln("received non INSERT event from dynamo - no action")
			return nil
		}

		// parse github event
		item := record.Change.NewImage
		eventType := item["type"].String()
		rawEvent := []byte(item["payload"].String())

		if eventType != types.EventTypePush {
			log.Warnln("received non-push event:", eventType, "- no action")
			return nil
		}

		var event types.GitHubEvent
		if err := json.Unmarshal(rawEvent, &event); err != nil {
			log.Errorln("json.Unmarshal", err.Error())
			return nil
		}

		log := log.WithFields(log.Fields{
			"ref":    parseRef(event.Ref),
			"commit": shortHash(event.After),
			"repo":   event.Repository.Name,
		})

		// fetch secure repo token
		secureStore := secure.NewAWSSecureStore(sess)
		token, err := secureStore.Get(types.KeyToken)
		if err != nil {
			log.Errorln("parameter.Get", err.Error())
			return nil
		}

		// prepare processing dependencies and fire status
		stackManager := stack.NewAWSStackManger(log, sess)
		repo := repo.NewGitHubRepository(log, event.Repository.Owner.Name, event.Repository.Name, token)
		shortHash := shortHash(event.After)

		// status - pending
		repo.Status(event.After, prepStatus(types.GitStatePending, shortHash))

		if err := Process(log, event, repo, stackManager, token); err != nil {
			log.Errorln("error processing event:", err.Error())

			// status - failure
			repo.Status(event.After, prepStatus(types.GitStateFailure, shortHash))

			return nil
		}

		// status - ok
		repo.Status(event.After, prepStatus(types.GitStateSuccess, shortHash))
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
//       stack = opolis-build-{repo}-release-pipeline
//     if ref = 'master':
//       stack = opolis-build-{repo}-master-pipeline
//     else:
//       stack = opolis-build-{repo}-{ref}-pipeline
//
//     if event.deleted:
//       if not exists(stack): warn and skip
//       else: delete stack
//       return
//
//     prepare context and set parameters
//
//     create or update stack with parameters
//     if tag: call UpdatePipeline with tag
//     call StartPipeline
//
func Process(log *log.Entry, event types.GitHubEvent, repo types.Repository, manager types.StackManager, repoToken string) error {
	// Get stack state, delete if necessary
	stack := stackName(event.Repository.Name, event.Ref)
	exists, _, err := manager.Status(stack)
	if err != nil {
		return err
	}

	if event.Deleted {
		if !exists {
			log.Warnln("received push/deleted event for non-existant stack")
			return nil
		}

		return manager.Delete(stack)
	}

	// fetch stack and parameter files from repoistory
	// pipeline.json - CI/CD pipeline stack spec
	// parameters.json - stack parameters
	template, parameters, err := buildContext(event, repo, "pipeline.json", "parameters.json")
	if err != nil {
		return err
	}

	// ammend parameter list with required parameters
	parameters = append(parameters, requiredParameters(event, repoToken, os.Getenv("ARTIFACT_STORE"))...)

	// create or update stack with ref specific parameters
	if !exists {
		// create - pipeline is started automatically when created
		log.Infoln("stack create", stack)
		if err := StackOp(log, manager.Create, manager, stack, parameters, template); err != nil {
			return err
		}
	} else {
		// update - manually start pipeline
		log.Infoln("stack update", stack)
		if err := StackOp(log, manager.Update, manager, stack, parameters, template); err != nil {
			return err
		}

		log.Infoln("start build")
		if err := manager.StartBuild(stack); err != nil {
			return err
		}
	}

	return nil
}

// StackOp performs the given stack operation (Create or Update), but waits until
// the operation is either completed or failed.
func StackOp(log *log.Entry, op types.StackOperation, manager types.StackManager, stack string, parameters []types.Parameter, template []byte) error {
	if err := op(stack, parameters, template); err != nil {
		return err
	}

	for {
		_, status, err := manager.Status(stack)
		if err != nil {
			return err
		}

		// continue waiting if stack status is neither "completed" or "failed"
		if !(regexCompleted.MatchString(status) || regexFailed.MatchString(status)) {
			log.Infoln("stack status", status)
			time.Sleep(time.Second)
			continue
		}

		log.Infoln("stack status", status)
		return nil
	}
}

//
// Helpers
//

func stackName(repo, ref string) string {
	return fmt.Sprintf("opolis-build-pipeline-%s-%s", repo, parseRef(ref))
}

func shortHash(hash string) string {
	if len(hash) < 6 {
		return hash
	}

	return hash[:6]
}

func statusUrl(logGroup, logStream, shortHash string) string {
	base := fmt.Sprintf("https://%s.console.aws.amazon.com", os.Getenv("AWS_REGION"))
	path := fmt.Sprintf("/cloudwatch/home?region=%s#logEventViewer:group=%s;stream=%s;filter=%s",
		os.Getenv("AWS_REGION"),
		logGroup,
		logStream,
		shortHash,
	)

	return base + path
}

func parseRef(ref string) string {
	components := strings.Split(ref, "/")
	return components[len(components)-1]
}

func parseParameters(parameters []byte) (types.ParameterManifest, error) {
	var parsed types.ParameterManifest
	if err := json.Unmarshal(parameters, &parsed); err != nil {
		return parsed, err
	}

	return parsed, nil
}

func prepStatus(state, shortHash string) types.GitHubStatus {
	return types.GitHubStatus{
		State:     state,
		Context:   types.GitContextPrep,
		TargetUrl: statusUrl(lambdacontext.LogGroupName, lambdacontext.LogStreamName, shortHash),
	}
}

func requiredParameters(event types.GitHubEvent, repoToken, artifactStore string) []types.Parameter {
	return []types.Parameter{
		types.Parameter{ParameterKey: "ArtifactStore", ParameterValue: artifactStore},
		types.Parameter{ParameterKey: "RepoOwner", ParameterValue: event.Repository.Owner.Name},
		types.Parameter{ParameterKey: "RepoName", ParameterValue: event.Repository.Name},
		types.Parameter{ParameterKey: "RepoBranch", ParameterValue: parseRef(event.Ref)},
		types.Parameter{ParameterKey: "RepoToken", ParameterValue: repoToken},
	}
}

func buildContext(event types.GitHubEvent, repo types.Repository, templatePath, parameterPath string) ([]byte, []types.Parameter, error) {
	template, err := repo.Get(event.Ref, templatePath)
	if err != nil {
		return nil, nil, err
	}

	parameterSpec, err := repo.Get(event.Ref, parameterPath)
	if err != nil {
		return nil, nil, err
	}

	// TODO(ngmiller): Needs to handle dev vs master vs release distinction
	parameters, err := parseParameters(parameterSpec)
	if err != nil {
		return nil, nil, err
	}

	return template, parameters.Development, nil
}
