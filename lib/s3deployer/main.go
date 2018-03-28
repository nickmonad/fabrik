package main

import (
	"archive/zip"
	"bytes"
	"io/ioutil"
	"net/http"

	"github.com/opolis/build/pipeline"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"

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

	log := log.WithFields(log.Fields{"jobId": id})

	// get input artifacts
	if len(data.InputArtifacts) == 0 {
		log.Warnln("no input artifacts")

		if err := pipeline.JobFailure(id, "no input artifacts"); err != nil {
			log.Errorln("could not post failure", id, err.Error())
		}

		return nil
	}

	artifactBucket := data.InputArtifacts[0].Location.S3Location.BucketName
	artifactKey := data.InputArtifacts[0].Location.S3Location.ObjectKey

	resp, err := s3.New(sess).GetObject(&s3.GetObjectInput{
		Bucket: aws.String(artifactBucket),
		Key:    aws.String(artifactKey),
	})

	if err != nil {
		status := "could not fetch input artifact " + artifactBucket + "/" + artifactKey + " " + err.Error()
		log.Errorln(status)

		if err := pipeline.JobFailure(id, status); err != nil {
			log.Errorln("could not post failure", id, err.Error())
		}

		return nil
	}

	// upload
	defer resp.Body.Close()
	deployBucket := data.ActionConfiguration.Configuration.UserParameters

	buffer, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorln(err.Error())
		if err := pipeline.JobFailure(id, err.Error()); err != nil {
			log.Errorln("could not post failure", id, err.Error())
		}
		return nil
	}

	// Open a zip archive for reading.
	reader, err := zip.NewReader(bytes.NewReader(buffer), int64(len(buffer)))
	if err != nil {
		log.Errorln(err.Error())
		if err := pipeline.JobFailure(id, err.Error()); err != nil {
			log.Errorln("could not post failure", id, err.Error())
		}
		return nil
	}

	// Iterate through the files in the archive, and upload them to S3
	for _, f := range reader.File {
		key := f.Name
		body, err := f.Open()
		if err != nil {
			log.Errorln(err.Error())
			if err := pipeline.JobFailure(id, err.Error()); err != nil {
				log.Errorln("could not post failure", id, err.Error())
			}
			return nil
		}

		content, _ := ioutil.ReadAll(body)
		body.Close()

		contentType := http.DetectContentType(content)

		_, err = s3.New(sess).PutObject(&s3.PutObjectInput{
			Body:        bytes.NewReader(content),
			Bucket:      aws.String(deployBucket),
			Key:         aws.String(key),
			ContentType: aws.String(contentType),
		})

		if err != nil {
			log.Errorln(err.Error())
			if err := pipeline.JobFailure(id, err.Error()); err != nil {
				log.Errorln("could not post failure", id, err.Error())
			}
			return nil
		}
	}

	if err := pipeline.JobSuccess(id); err != nil {
		log.Errorln("could not post success", id, err.Error())
	}

	return nil
}
