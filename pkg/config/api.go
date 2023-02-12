package config

type Service struct {
	Name        string    `json:"name"`
	Title       string    `json:"title,omitempty"`
	Description string    `json:"description,omitempty"`
	Servers     []*Server `json:"servers,omitempty"`

	Resource
}

type Server struct {
	Name    string `json:"name"`
	BaseURL string `json:"baseUrl"`
}

type Resource struct {
	Name        string     `json:"name,omitempty"`
	Description string     `json:"description,omitempty"`
	Resources   []Resource `json:"resources,omitempty"`
	URI         string     `json:"uri,omitempty"`
	Get         *Endpoint  `json:"get,omitempty"`
	Put         *Endpoint  `json:"put,omitempty"`
	Post        *Endpoint  `json:"post,omitempty"`
	Delete      *Endpoint  `json:"delete,omitempty"`
	Options     *Endpoint  `json:"options,omitempty"`
	Head        *Endpoint  `json:"head,omitempty"`
	Trace       *Endpoint  `json:"trace,omitempty"`
	Patch       *Endpoint  `json:"patch,omitempty"`
}

type Endpoint struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}
