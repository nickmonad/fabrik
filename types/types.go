package types

import "regexp"

const (
	DynamoDBEventInsert = "INSERT"

	EventTypePush = "push"

	GitContextPrep  = "pipeline/Prep"
	GitRefBranch    = "branch"
	GitRefMaster    = "master"
	GitRefRelease   = "release"
	GitStateError   = "error"
	GitStateFailure = "failure"
	GitStatePending = "pending"
	GitStateSuccess = "success"

	KeyHmac  = "opolis-build-hmac"
	KeyToken = "opolis-build-token"

	PipelineStateStarted   = "STARTED"
	PipelineStateResumed   = "RESUMED"
	PipelineStateSucceeded = "SUCCEEDED"
	PipelineStateFailed    = "FAILED"
)

var (
	RegexReleaseRef = regexp.MustCompile(`v[0-9]+\.[0-9]+\.[0-9]+$`)
)

// Repository provides a means to fetch data from
// the version control repository.
type Repository interface {
	Get(ref string, path string) ([]byte, error)
	Status(sha string, status GitHubStatus) error
}

// RepoNotFoundError - semantic type to represent '404' from a repo fetch
type RepoNotFoundError struct{}

func (e RepoNotFoundError) Error() string {
	return "not found"
}

// BuildContext represents the templates and parameters required to deploy
// a pipeline.
type BuildContext struct {
	PipelineTemplate    []byte
	DeployStackTemplate []byte
	Parameters          []Parameter
}

// StackManager provides a means of managing infrastructure 'stacks'
// A stack is a collection of resources typically specified by a version
// controlled file.
type StackManager interface {
	Create(name string, parameters []Parameter, template []byte) error
	Update(name string, parameters []Parameter, template []byte) error
	Delete(name string) error
	Status(name string) (bool, string, error)

	StartBuild(name string) error
	UpdateBuild(name, ref string) error
}

// PipelineManger provides a means of interacting with and querying
// active CI/CD pipelines.
type PipelineManager interface {
	GetRepoInfo(name string) (string, string, error)
	GetRevision(execId, name string) (string, error)
	JobSuccess(id string) error
	JobFailure(id, message string) error
}

// SecureStore accesses secure parameters.
type SecureStore interface {
	Get(key string) (string, error)
}

// GitHubEvent references relevant fields from the push event.
type GitHubEvent struct {
	Ref        string `json:"ref"`
	Before     string `json:"before"`
	After      string `json:"after"`
	Created    bool   `json:"created"`
	Deleted    bool   `json:"deleted"`
	Repository struct {
		Name  string `json:"name"`
		Owner struct {
			Name string `json:"name"`
		} `json:"owner"`
	} `json:"repository"`
}

// GitHubStatus stores status context for a particular repo commit hash
type GitHubStatus struct {
	State       string `json:"state"`
	TargetUrl   string `json:"target_url"`
	Description string `json:"description"`
	Context     string `json:"context"`
}

// Parameter defines a common format for expressing stack parameters.
type Parameter struct {
	ParameterKey   string `json:"ParameterKey"`
	ParameterValue string `json:"ParameterValue"`
}

// ParameterManifest defines a common format for expressing a _set_ of stack parameters.
type ParameterManifest struct {
	Development []Parameter `json:"development"`
	Master      []Parameter `json:"master"`
	Release     []Parameter `json:"release"`
}

// PipelineStageDetail represents a stage change event metadata
type PipelineStageDetail struct {
	Version     float32 `json:"version"`
	Pipeline    string  `json:"pipeline"`
	ExecutionId string  `json:"execution-id"`
	Stage       string  `json:"stage"`
	State       string  `json:"state"`
}
