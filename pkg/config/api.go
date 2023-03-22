package config

type Service struct {
	Name        string   `json:"name" yaml:"name"`
	Title       string   `json:"title,omitempty" yaml:"title,omitempty"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`
	Servers     []Server `json:"servers,omitempty" yaml:"servers,omitempty"`

	Resources []Resource `json:"resources,omitempty" yaml:"resources,omitempty"`
}

type Server struct {
	Name    string `json:"name" yaml:"name"`
	BaseURL string `json:"baseUrl" yaml:"baseUrl"`
	Headers Header `json:"headers" yaml:"headers"`
	Links   []Link `json:"links,omitempty" yaml:"links,omitempty"`
}

type Resource struct {
	Name        string     `json:"name,omitempty" yaml:"name,omitempty"`
	Description string     `json:"description,omitempty" yaml:"description,omitempty"`
	Resources   []Resource `json:"resources,omitempty" yaml:"resources,omitempty"`
	URI         string     `json:"uri,omitempty" yaml:"uri,omitempty"`
	Headers     Header     `json:"headers,omitempty" yaml:"headers,omitempty"`
	Links       []Link     `json:"links,omitempty" yaml:"links,omitempty"`
	Get         *Endpoint  `json:"get,omitempty" yaml:"get,omitempty"`
	Put         *Endpoint  `json:"put,omitempty" yaml:"put,omitempty"`
	Post        *Endpoint  `json:"post,omitempty" yaml:"post,omitempty"`
	Delete      *Endpoint  `json:"delete,omitempty" yaml:"delete,omitempty"`
	Options     *Endpoint  `json:"options,omitempty" yaml:"options,omitempty"`
	Head        *Endpoint  `json:"head,omitempty" yaml:"head,omitempty"`
	Trace       *Endpoint  `json:"trace,omitempty" yaml:"trace,omitempty"`
	Patch       *Endpoint  `json:"patch,omitempty" yaml:"patch,omitempty"`
}

type Endpoint struct {
	Name        string `json:"name,omitempty" yaml:"name,omitempty"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Headers     Header `json:"headers,omitempty" yaml:"headers,omitempty"`
	Links       []Link `json:"links,omitempty" yaml:"links,omitempty"`
}

type Link struct {
	HRef     string `json:"href,omitempty yaml:"href,omitempty"`
	Audience string `json:"audience,omitempty yaml:"audience,omitempty"`
	Rel      string `json:"rel,omitempty yaml:"rel,omitempty"`
	Title    string `json:"title,omitempty yaml:"title,omitempty"`
}
