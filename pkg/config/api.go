package config

type Service struct {
	Name        string   `json:"name"`
	Title       string   `json:"title,omitempty"`
	Description string   `json:"description,omitempty"`
	Servers     []Server `json:"servers,omitempty"`

	Resources []Resource `json:"resources,omitempty"`
}

type Server struct {
	Name    string `json:"name"`
	BaseURL string `json:"baseUrl"`
	Headers Header `json:"headers"`
	Links   []Link `json:"links,omitempty"`
}

type Resource struct {
	Name        string     `json:"name,omitempty"`
	Description string     `json:"description,omitempty"`
	Resources   []Resource `json:"resources,omitempty"`
	URI         string     `json:"uri,omitempty"`
	Headers     Header     `json:"headers,omitempty"`
	Links       []Link     `json:"links,omitempty"`
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
	Headers     Header `json:"headers,omitempty"`
	Links       []Link `json:"links,omitempty"`
}

type Link struct {
	HRef     string `json:"href,omitempty"`
	Audience string `json:"audience,omitempty"`
	Rel      string `json:"rel,omitempty"`
	Title    string `json:"title,omitempty"`
}
