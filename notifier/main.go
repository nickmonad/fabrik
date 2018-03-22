package main

import (
	"encoding/json"

	"github.com/opolis/build/types"

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

// Handler serves as the integration point between the AWS event and business logic by
// preparing conrete types to satisfy the Process interface.
func Handler(event events.CloudWatchEvent) error {
	defer func() {
		if r := recover(); r != nil {
			log.Errorln("recovered from panic:", r)
		}
	}()

	// AWS session
	// sess := session.Must(session.NewSession())

	// Pull the pipeline event detail
	var detail types.PipelineStageDetail
	if err := json.Unmarshal(event.Detail, &detail); err != nil {
		log.Errorln("json.Unmarshal", err.Error())
		return nil
	}

	log.Println(detail.Pipeline, detail.Stage, detail.State)
	return nil
}
