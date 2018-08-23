package main

import (
	"github.com/opolis/build/secure"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/nlopes/slack"

	log "github.com/sirupsen/logrus"
)

// TODO: these need to be configurable by the user in some fashion
// Ultimately, I think it will boil down to decouple these lib/ functions
// from the main serverless deploy spec, and allow them to be deployed separately,
// where env variables and more specific IAM roles can be set
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

func Handler(event events.CloudwatchLogsEvent) error {
	defer func() {
		if r := recover(); r != nil {
			log.Errorln("recovered from panic:", r)
		}
	}()

	// AWS Session
	sesh := session.Must(session.NewSession())

	// Slack OAuth token
	secureStore := secure.NewAWSSecureStore(sesh)
	token, err := secureStore.Get(tokenKey)
	if err != nil {
		log.Errorln("could not fetch slack token:", err.Error())
		return nil
	}

	logs, err := event.AWSLogs.Parse()
	if err != nil {
		log.Errorln(err.Error())
		return nil
	}

	for _, logEvent := range logs.LogEvents {
		if err := PostMessage(token, logs.LogStream, logEvent.Message); err != nil {
			log.Errorln("could not post message:", err.Error())
			return nil
		}
	}

	return nil
}

func PostMessage(token, stream, message string) error {
	// slack session
	api := slack.New(token)

	msg := "<!channel> Event from *" + stream + "*"
	attach := slack.Attachment{Text: message}
	params := slack.PostMessageParameters{
		Attachments: []slack.Attachment{attach},
		Markdown:    true,
	}

	_, _, err := api.PostMessage(channelID, msg, params)
	return err
}
