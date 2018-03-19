package main

import (
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func main() {
	lambda.Start(Handler)
}

func Handler(event events.DynamoDBEvent) error {
	for _, record := range event.Records {
		fmt.Println("Got Event ", record.EventName, " from ", record.EventSource)
	}

	return nil
}
