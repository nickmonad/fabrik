package main

import (
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	log "github.com/sirupsen/logrus"
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

	logs, err := event.AWSLogs.Parse()
	if err != nil {
		log.Errorln(err.Error())
		return nil
	}

	for _, logEvent := range logs.LogEvents {
		log.Infoln("match: " + logEvent.Message)
	}

	return nil
}
