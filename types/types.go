package types

// Repository provides a means to fetch data from
// the version control repository.
type Repository interface {
	Get(ref string, path string) ([]byte, error)
}

// Pipeline provides a means to manage pipelines.
type Pipeline interface {
	Create(name string) error
	Delete(name string) error
	Exists(name string) bool
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
		Name string `json:"name"`
	} `json:"repository"`
}
