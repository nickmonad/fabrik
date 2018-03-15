package main

import (
	"log"

	"github.com/aws/aws-lambda-go/lambda"
)

func main() {
	lambda.Start(Handler)
}

func Handler() error {
	log.Println("hello lambda")
	return nil
}
