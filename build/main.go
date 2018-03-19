package main

import (
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

type Manager interface {
	Create()
	Update()
	Delete()
	Exists()
	Start()
}

func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	fmt.Println("Body: ", request.Body)
	return events.APIGatewayProxyResponse{Body: "hello: " + request.Body, StatusCode: 200}, nil
}

func main() {
	lambda.Start(Handler)
}
