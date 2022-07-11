package model

import (
	"github.com/Carbonfrost/joe-cli-http/uritemplates"
)

type Service struct {
	Name        string    `json:"name"`
	Title       string    `json:"title,omitempty"`
	Description string    `json:"description,omitempty"`
	Servers     []*Server `json:"servers,omitempty"`

	*Resource
}

type Server struct {
	Name    string `json:"name"`
	BaseURL string `json:"baseUrl"`
}

type Resource struct {
	Name        string                    `json:"name"`
	Resources   []*Resource               `json:"resources,omitempty"`
	Method      string                    `json:"method,omitempty"`
	URITemplate *uritemplates.URITemplate `json:"uriTemplate,omitempty"`
}

func (r *Resource) Resource(name string) (*Resource, bool) {
	for _, c := range r.Resources {
		if c.Name == name {
			return c, true
		}
	}
	return nil, false
}

func (s *Service) Server(name string) (*Server, bool) {
	for _, c := range s.Servers {
		if c.Name == name {
			return c, true
		}
	}
	return nil, false
}
