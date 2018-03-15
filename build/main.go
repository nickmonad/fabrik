package main

import (
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	fmt.Println("Body: ", request.Body)
	return events.APIGatewayProxyResponse{Body: "hello: " + request.Body, StatusCode: 200}, nil
}

func main() {
	lambda.Start(Handler)
}
