package main

import (
	"github.com/opolis/build/pipeline"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"

	log "github.com/sirupsen/logrus"
)

func init() {
	log.SetFormatter(&log.JSONFormatter{DisableTimestamp: true})
}

func main() {
	lambda.Start(Handler)
}

func Handler(event events.CodePipelineEvent) error {
	defer func() {
		if r := recover(); r != nil {
			log.Errorln("recovered from panic:", r)
		}
	}()

	// AWS session
	sess := session.Must(session.NewSession())
	pipeline := pipeline.NewAWSPipelineManager(sess)

	id := event.CodePipelineJob.ID
	data := event.CodePipelineJob.Data
	creds := data.ArtifactCredentials

	// get s3 bucket creds from pipeline data
	session.Must(session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(
			creds.AccessKeyID,
			creds.SecretAccessKey,
			creds.SessionToken,
		),
	}))

	// get deployment bucket
	// string (json), should hold 'bucket' key: data.ActionConfiguration.Configuration.UserParameters

	// get input artifacts
	// data.InputArtifacts[0].Location.S3Location.(BucketName|ObjectKey)

	// upload

	// Open a zip archive for reading.
	// r, err := zip.OpenReader("test.zip")
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// defer r.Close()

	// // Iterate through the files in the archive,
	// // printing some of their contents.
	// for _, f := range r.File {
	// 	fmt.Printf("Found %s:\n", f.Name)
	// }

	if err := pipeline.JobSuccess(id); err != nil {
		log.Errorln("could not post success", id, err.Error())
	}

	return nil
}
