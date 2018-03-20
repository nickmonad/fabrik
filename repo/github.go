package repo

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
)

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

	fmt.Println("requesting: ", url)

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	request.Header.Set("Authorization", fmt.Sprintf("token %s", repo.token))

	// make request
	resp, err := repo.client.Do(request)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("error making request: %s", err.Error()))
	}
	defer resp.Body.Close()

	// return error for non-200 status code
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(fmt.Sprintf("error fetching %s: %s", path, resp.Status))
	}

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
