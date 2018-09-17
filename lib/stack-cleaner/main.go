package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/opolis/build/lambda"
	"github.com/opolis/build/stack"
	"github.com/opolis/build/types"

	awsLambda "github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/aws/aws-sdk-go/aws/session"

	log "github.com/sirupsen/logrus"
)

const (
	// Execution timeout in seconds
	ExecutionTimeout = 300
)

func init() {
	log.SetFormatter(&log.JSONFormatter{DisableTimestamp: true})
}

func main() {
	awsLambda.Start(Handler)
}

func Handler(event types.CloudFormationEvent) error {
	defer func() {
		if r := recover(); r != nil {
			log.Errorln("recovered from panic:", r)
		}
	}()

	// AWS session
	sess := session.Must(session.NewSession())
	log := log.WithFields(log.Fields{"stackId": event.StackId})
	logLocation := lambdacontext.LogGroupName + "/" + lambdacontext.LogStreamName

	// prepare processing dependencies
	stackManager := stack.NewAWSStackManager(log, sess)
	lambdaManager := lambda.NewAWSLambdaManager(sess)

	// prepare required repsonse parameters
	response := types.CloudFormationResponse{
		StackId:            event.StackId,
		RequestId:          event.RequestId,
		LogicalResourceId:  event.LogicalResourceId,
		PhysicalResourceId: event.PhysicalResourceId,
	}

	if event.RequestType != types.CloudFormationRequestDelete {
		// ignore non-delete requests
		log.Infoln("ignoring RequestType", event.RequestType)

		response.Status = types.CloudFormationResponseSuccess
		response.PhysicalResourceId = logLocation

		return Response(event.ResponseURL, response)
	}

	// parse properties, get stack name
	var properties map[string]string
	if err := json.Unmarshal(event.ResourceProperties, &properties); err != nil {
		response.Status = types.CloudFormationResponseFailed
		response.Reason = logLocation
		log.Errorln("unable to unmarshal resource properties", err.Error())

		return Response(event.ResponseURL, response)
	}

	// wait until we get a concrete stack status
	// or 90% of the execution timeout has been used, in which case, restart
	stop := make(chan struct{})
	status := Process(log, stop, properties["Stack"], stackManager)

	select {
	case err := <-status:
		if err != nil {
			log.Errorln("error processing event:", err.Error())
			response.Status = types.CloudFormationResponseFailed
			response.Reason = logLocation
			return Response(event.ResponseURL, response)
		}

	case <-time.After(0.9 * ExecutionTimeout * time.Second):
		log.Infoln("execution timeout reached, restarting function!")
		close(stop)

		return lambdaManager.Invoke(lambdacontext.FunctionName, event)
	}

	// ok
	response.Status = types.CloudFormationResponseSuccess
	return Response(event.ResponseURL, response)
}

func Process(log *log.Entry, stop <-chan struct{}, stack string, manager types.StackManager) <-chan error {
	result := make(chan error)
	go func() {
		exists, status, _ := manager.Status(stack)
		if !exists {
			log.Infoln(fmt.Sprintf("stack %s not found, operation complete", stack))
			result <- nil
			return
		}

		if !statusInProgress(status) {
			if err := manager.Delete(stack); err != nil {
				log.Infoln("stack delete failed")
				result <- err
				return
			}
		}

		for {
			select {
			case <-stop:
				log.Infoln("stack monitor received stop signal")
				result <- nil
				return

			default:
				exists, status, _ := manager.Status(stack)
				if !exists {
					log.Infoln("done!")
					result <- nil
					return
				}

				log.Infoln("stack status", status)

				if statusRollback(status) || statusFailed(status) {
					result <- errors.New("stack rollback or failure")
					return
				}

				// in progress, wait for 1 second
				time.Sleep(time.Second)
				continue
			}
		}
	}()

	return result
}

func Response(url string, response types.CloudFormationResponse) error {
	payload, err := json.Marshal(response)
	if err != nil {
		return err
	}

	request, err := http.NewRequest("PUT", url, bytes.NewReader(payload))
	if err != nil {
		return err
	}

	_, err = http.DefaultClient.Do(request)
	return err
}

func statusComplete(status string) bool {
	return types.RegexCompleted.MatchString(status)
}

func statusInProgress(status string) bool {
	return types.RegexInProgress.MatchString(status)
}

func statusRollback(status string) bool {
	return types.RegexRollback.MatchString(status)
}

func statusFailed(status string) bool {
	return types.RegexFailed.MatchString(status)
}
