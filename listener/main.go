package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
)

// Repository provides a means to fetch data from
// the version control repository.
type Repository interface {
	Get(ref string, path string) ([]byte, error)
}

type GitHubRepository struct {
	client *http.Client
	base   string
	token  string
	owner  string
	name   string
}

func NewGitHubRepository(owner, name, token string) *GitHubRepository {
	return &GitHubRepository{
		client: http.DefaultClient,
		base:   "https://api.github.com",
		token:  token,
		owner:  owner,
		name:   name,
	}
}

func (repo *GitHubRepository) Get(ref, path string) ([]byte, error) {
	url := fmt.Sprintf(
		"%s/repos/%s/%s/contents/%s?ref=%s",
		repo.base, repo.owner, repo.name, path, ref,
	)

	fmt.Println("url: ", url)

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	request.Header.Set("Authorization", fmt.Sprintf("token %s", repo.token))

	// make request
	resp, err := repo.client.Do(request)
	if err != nil {
		fmt.Println("error making request")
		return nil, err
	}
	defer resp.Body.Close()

	// read json
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("error reading body")
		return nil, err
	}

	// decode base64 content
	var parsed map[string]interface{}
	if err := json.Unmarshal(body, &parsed); err != nil {
		fmt.Println("error decoding json")
		return nil, err
	}

	return base64.StdEncoding.DecodeString(parsed["content"].(string))
}

func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// AWS Session
	sess := session.Must(session.NewSession(&aws.Config{Region: aws.String(endpoints.UsWest2RegionID)}))
	svc := ssm.New(sess)

	// Get secure token
	token, err := svc.GetParameter(&ssm.GetParameterInput{
		Name: aws.String("opolis-build-token"), WithDecryption: aws.Bool(true)})

	if err != nil {
		fmt.Println(err.Error())
		return events.APIGatewayProxyResponse{Body: "could not get token", StatusCode: 500}, nil
	}

	// GitHub Session
	repo := NewGitHubRepository("opolis", "build", *(token.Parameter.Value))
	data, err := repo.Get("working", "Dockerfile")
	if err != nil {
		fmt.Println(err.Error())
		return events.APIGatewayProxyResponse{Body: "could not fetch content", StatusCode: 500}, nil
	}

	return events.APIGatewayProxyResponse{Body: string(data), StatusCode: 200}, nil
}

func main() {
	lambda.Start(Handler)
}
