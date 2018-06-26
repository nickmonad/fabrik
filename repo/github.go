package repo

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/opolis/build/types"

	log "github.com/sirupsen/logrus"
)

type GitHubRepository struct {
	log    *log.Entry
	client *http.Client
	base   string
	token  string
	owner  string
	name   string
}

func NewGitHubRepository(log *log.Entry, owner, name, token string) *GitHubRepository {
	return &GitHubRepository{
		log:    log,
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

	repo.log.Infoln("requesting:", path)

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

	// return 'not found' for 404
	if resp.StatusCode == http.StatusNotFound {
		return nil, types.RepoNotFoundError{}
	}

	// return error for non-200 status code
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(fmt.Sprintf("error fetching %s: %s", path, resp.Status))
	}

	// read json
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading body: %s", err.Error())
	}

	// decode base64 content
	var parsed map[string]interface{}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("error decoding json: %s", err.Error())
	}

	return base64.StdEncoding.DecodeString(parsed["content"].(string))
}

func (repo *GitHubRepository) Status(sha string, status types.GitHubStatus) error {
	payload, err := json.Marshal(status)
	if err != nil {
		return err
	}

	url := fmt.Sprintf(
		"%s/repos/%s/%s/statuses/%s",
		repo.base, repo.owner, repo.name, sha,
	)

	repo.log.Infoln("posting status", status.Context, status.State)

	request, err := http.NewRequest("POST", url, bytes.NewReader(payload))
	if err != nil {
		return err
	}

	request.Header.Set("Authorization", fmt.Sprintf("token %s", repo.token))

	// make request
	resp, err := repo.client.Do(request)
	if err != nil {
		return errors.New(fmt.Sprintf("error making request: %s", err.Error()))
	}
	defer resp.Body.Close()

	// return error for non-200 status code
	if resp.StatusCode < 200 || resp.StatusCode > 300 {
		return errors.New(fmt.Sprintf("error posting status %s", resp.Status))
	}

	return nil
}
