package model

import (
	"fmt"
	"strings"

	"github.com/Carbonfrost/joe-cli-http/uritemplates"
	"github.com/Carbonfrost/pastiche/pkg/config"
)

type Model struct {
	Services map[string]*Service
}

type Service struct {
	Name        string
	Title       string
	Description string
	Servers     []*Server
	Links       []Link

	Resource *Resource
}

type Server struct {
	Name    string
	BaseURL string
	Headers map[string][]string
	Links   []Link
}

type Resource struct {
	Name        string
	Description string
	Resources   []*Resource
	Endpoints   []*Endpoint
	URITemplate *uritemplates.URITemplate
	Headers     map[string][]string
	Links       []Link
	Command     []string
}

type Endpoint struct {
	Name        string
	Description string
	Method      string
	Headers     map[string][]string
	Links       []Link
}

type Link struct {
	HRef     string
	Audience string
	Rel      string
	Title    string
}

func New(c *config.Config) *Model {
	res := &Model{
		Services: map[string]*Service{},
	}
	for _, v := range c.Services {
		res.Services[v.Name] = service(v)
	}
	return res
}

func service(v config.Service) *Service {
	servers := make([]*Server, len(v.Servers))
	for i, s := range v.Servers {
		servers[i] = &Server{
			Name:    s.Name,
			BaseURL: s.BaseURL,
			Links:   links(s.Links),
		}
	}
	return &Service{
		Name:        v.Name,
		Title:       v.Title,
		Description: v.Description,
		Servers:     servers,
		Resource: &Resource{
			Name:        "/",
			URITemplate: mustParseURITemplate("/"),
			Resources:   resources(v.Resources),
			Endpoints: []*Endpoint{
				{
					Method: "GET",
				},
			},
		},
	}
}

func mustParseURITemplate(t string) *uritemplates.URITemplate {
	u, err := uritemplates.Parse(t)
	if err != nil {
		panic(err)
	}
	return u
}

func resource(r config.Resource) *Resource {
	uri, _ := uritemplates.Parse(r.URI)
	res := &Resource{
		Name:        r.Name,
		Description: r.Description,
		URITemplate: uri,
		Headers:     r.Headers,
		Links:       links(r.Links),
	}
	if r.Get != nil {
		res.Endpoints = append(res.Endpoints, endpoint("GET", r.Get))
	}
	if r.Put != nil {
		res.Endpoints = append(res.Endpoints, endpoint("PUT", r.Put))
	}
	if r.Post != nil {
		res.Endpoints = append(res.Endpoints, endpoint("POST", r.Post))
	}
	if r.Delete != nil {
		res.Endpoints = append(res.Endpoints, endpoint("DELETE", r.Delete))
	}
	if r.Options != nil {
		res.Endpoints = append(res.Endpoints, endpoint("OPTIONS", r.Options))
	}
	if r.Head != nil {
		res.Endpoints = append(res.Endpoints, endpoint("HEAD", r.Head))
	}
	if r.Trace != nil {
		res.Endpoints = append(res.Endpoints, endpoint("TRACE", r.Trace))
	}
	if r.Patch != nil {
		res.Endpoints = append(res.Endpoints, endpoint("PATCH", r.Patch))
	}

	// Implicitly create GET endpoint if none other was created
	if len(res.Endpoints) == 0 {
		res.Endpoints = append(res.Endpoints, &Endpoint{Method: "GET"})
	}
	res.Resources = resources(r.Resources)
	return res
}

func resources(resources []config.Resource) []*Resource {
	res := make([]*Resource, len(resources))
	for i, child := range resources {
		res[i] = resource(child)
	}
	return res
}

func endpoint(method string, r *config.Endpoint) *Endpoint {
	return &Endpoint{
		Name:        r.Name,
		Description: r.Description,
		Method:      method,
		Headers:     r.Headers,
		Links:       links(r.Links),
	}
}

func links(links []config.Link) []Link {
	res := make([]Link, len(links))
	for i, l := range links {
		res[i] = Link{
			HRef:     l.HRef,
			Audience: l.Audience,
			Rel:      l.Rel,
			Title:    l.Title,
		}
	}
	return res
}

func (s *Service) Server(name string) (*Server, bool) {
	for _, c := range s.Servers {
		if c.Name == name {
			return c, true
		}
	}
	return nil, false
}

func (c *Model) Service(name string) (*Service, bool) {
	svc, ok := c.Services[name]
	return svc, ok
}

func (c *Model) Resolve(s ServiceSpec) (*Service, *Resource, error) {
	if len(s) == 0 {
		return nil, nil, fmt.Errorf("no service specified")
	}
	svc, ok := c.Service(s[0])
	if !ok {
		return nil, nil, fmt.Errorf("service not found: %q", s[0])
	}

	current := svc.Resource
	for i, p := range s[1:] {
		current, ok = current.Resource(p)
		if !ok {
			path := ServiceSpec(s[0 : i+2]).Path()
			return nil, nil, fmt.Errorf("resource not found: %q", path)
		}
	}

	return svc, current, nil
}

func (r *Resource) Resource(name string) (*Resource, bool) {
	for _, c := range r.Resources {
		if c.Name == name {
			return c, true
		}
	}
	return nil, false
}

func (r *Resource) Endpoint(m string) (*Endpoint, bool) {
	for _, c := range r.Endpoints {
		if strings.EqualFold(c.Method, m) {
			return c, true
		}
	}
	return nil, false
}
