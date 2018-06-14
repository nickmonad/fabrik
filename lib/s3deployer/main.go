package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/opolis/build/pipeline"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
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
	log := log.WithFields(log.Fields{"jobId": id})

	// Get input artifacts
	stackArtifact, buildArtifact, err := getArtifacts(sess, data)
	if err != nil {
		if err := pipeline.JobFailure(id, err.Error()); err != nil {
			log.Errorln("could not post failure", id, err.Error())
		}

		return nil
	}

	defer stackArtifact.Close()
	defer buildArtifact.Close()

	// Read deploy bucket name
	bucket, err := readDeployBucket(stackArtifact)
	if err != nil {
		if err := pipeline.JobFailure(id, err.Error()); err != nil {
			log.Errorln("could not post failure", id, err.Error())
		}

		return nil
	}

	// Deploy build artifact
	if err := deployBuild(sess, bucket, buildArtifact); err != nil {
		if err := pipeline.JobFailure(id, err.Error()); err != nil {
			log.Errorln("could not post failure", id, err.Error())
		}

		return nil
	}

	if err := pipeline.JobSuccess(id); err != nil {
		log.Errorln("could not post success", id, err.Error())
	}

	return nil
}

//
// Helpers
//

// getAritifacts - return (stack artifact, build artifact, error)
func getArtifacts(sess *session.Session, data events.CodePipelineData) (io.ReadCloser, io.ReadCloser, error) {
	if len(data.InputArtifacts) == 0 {
		return nil, nil, errors.New("no input artifacts")
	}

	// Check for deploy stack and build output artifacts
	if len(data.InputArtifacts) == 2 {
		stack, err := getS3(sess, data.InputArtifacts[0])
		if err != nil {
			return nil, nil, err
		}

		build, err := getS3(sess, data.InputArtifacts[1])
		if err != nil {
			return nil, nil, err
		}

		return stack, build, nil
	}

	return nil, nil, errors.New("invalid amount of parameters")
}

func getS3(sess *session.Session, input events.CodePipelineInputArtifact) (io.ReadCloser, error) {
	bucket := input.Location.S3Location.BucketName
	key := input.Location.S3Location.ObjectKey

	resp, err := s3.New(sess).GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})

	return resp.Body, err
}

func readDeployBucket(stackArtifact io.ReadCloser) (string, error) {
	buffer, err := ioutil.ReadAll(stackArtifact)
	if err != nil {
		return "", err
	}

	// open the zip archive for reading
	reader, err := zip.NewReader(bytes.NewReader(buffer), int64(len(buffer)))
	if err != nil {
		return "", err
	}

	// look for `outputs.json` and marshal into a map
	// extracting the `Bucket` key
	for _, f := range reader.File {
		if f.Name == "outputs.json" {
			body, err := f.Open()
			if err != nil {
				return "", err
			}

			content, _ := ioutil.ReadAll(body)
			body.Close()

			var object map[string]string
			if err := json.Unmarshal(content, &object); err != nil {
				return "", err
			}

			return object["Bucket"], nil
		}
	}

	return "", errors.New("outputs.json does not exist in stack artifact")
}

func deployBuild(sess *session.Session, bucket string, buildArtifact io.ReadCloser) error {
	buffer, err := ioutil.ReadAll(buildArtifact)
	if err != nil {
		return err
	}

	// open the zip archive for reading
	reader, err := zip.NewReader(bytes.NewReader(buffer), int64(len(buffer)))
	if err != nil {
		return err
	}

	// Iterate through the files in the archive, and upload them to S3
	for _, f := range reader.File {
		key := f.Name
		body, err := f.Open()
		if err != nil {
			return err
		}

		content, _ := ioutil.ReadAll(body)
		body.Close()

		contentType := http.DetectContentType(content)

		_, err = s3.New(sess).PutObject(&s3.PutObjectInput{
			Body:        bytes.NewReader(content),
			Bucket:      aws.String(bucket),
			Key:         aws.String(key),
			ContentType: aws.String(contentType),
		})

		if err != nil {
			return err
		}
	}

	return nil
}
