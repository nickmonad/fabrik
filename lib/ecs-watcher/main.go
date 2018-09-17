package main

import (
	"encoding/json"
	"fmt"

	"github.com/opolis/build/secure"
	"github.com/opolis/build/stack"
	"github.com/opolis/build/types"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/nlopes/slack"

	log "github.com/sirupsen/logrus"
)

const (
	tokenKey  = "bot.slack.token"
	channelID = "CCDAY0552"
)

func init() {
	log.SetFormatter(&log.JSONFormatter{DisableTimestamp: true})
}

func main() {
	lambda.Start(Handler)
}

// Handler serves as the integration point between the AWS event and business logic by
// preparing conrete types to satisfy the Process interface.
func Handler(raw events.CloudWatchEvent) error {
	defer func() {
		if r := recover(); r != nil {
			log.Errorln("recovered from panic:", r)
		}
	}()

	// AWS session
	sess := session.Must(session.NewSession())

	// Parse ECS event detail
	var event types.ECSEvent
	if err := json.Unmarshal(raw.Detail, &event); err != nil {
		log.Errorln("json.Unmarshal:", err.Error())
		return nil
	}

	// sanity checks
	if len(event.Containers) == 0 {
		log.Errorln("got event with no container specs")
		return nil
	}

	log := log.WithFields(log.Fields{"container": event.Containers[0].Name})
	manager := stack.NewAWSStackManager(log, sess)

	// Slack OAuth token
	secureStore := secure.NewAWSSecureStore(sess)
	token, err := secureStore.Get(tokenKey)
	if err != nil {
		log.Errorln("could not fetch slack token:", err.Error())
		return nil
	}

	// Request stack update rollback if container failed to start.
	// Note: this will also capture events when containers fail outside of an update context. In that case,
	// we just allow this function to exit gracefully and fire off an event to slack.
	if (event.Containers[0].LastStatus == types.EcsStateStopped) && (event.StoppedReason == types.EcsFailureReason) {
		// TODO: we are assuming the container name corresponds to the cloudformation stack name
		// this has been true for Opolis deploys, but will need a better generalization in the future
		if err := manager.CancelUpdate(event.Containers[0].Name); err != nil {
			log.Errorln("error cancelling stack update:", err.Error())
			return nil
		}

		if err := PostMessage(token, event.Containers[0].Name); err != nil {
			log.Errorln("error posting message to slack:", err.Error())
			return nil
		}
	}

	return nil
}

func PostMessage(token, stack string) error {
	// slack session
	api := slack.New(token)

	msg := fmt.Sprintf("<!channel> Container failed to start for *%s* - Rolling back stack update", stack)
	params := slack.PostMessageParameters{
		Markdown: true,
	}

	_, _, err := api.PostMessage(channelID, msg, params)
	return err
}
