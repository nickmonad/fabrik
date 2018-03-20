package types

const (
	EventTypePush = "push"
)

// Repository provides a means to fetch data from
// the version control repository.
type Repository interface {
	Get(ref string, path string) ([]byte, error)
}

// StackManager provides a means of managing infrastructure 'stacks'
// A stack is a collection of resources typically specified by a version
// controlled file.
type StackManager interface {
	Create(name string, template []byte, parameters []byte) error
	Update(name string, template []byte, parameters []byte) error
	Delete(name string) error
	Exists(name string) (bool, string, error)
}

// ParameterStore accesses secure parameters.
type ParameterStore interface {
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

// Parameter defines a common format for expressing stack parameters.
type Parameter struct {
	ParameterKey   string `json:"ParameterKey"`
	ParameterValue string `json:"ParameterValue"`
}
