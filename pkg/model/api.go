package model

import (
	"net/url"

	"github.com/Carbonfrost/joe-cli-http/httpclient"
	"github.com/Carbonfrost/joe-cli-http/uritemplates"
)

type Service struct {
	*Resource

	Name    string
	Servers []*Server
}

type Server struct {
	Name    string
	BaseURL string
}

type Resource struct {
	Resources   []*Resource
	Name        string
	Method      string
	URITemplate *uritemplates.URITemplate
}

func (r *Resource) ApplyToClient(c *httpclient.Client, baseURL *url.URL) error {
	vars := map[string]interface{}{}
	expanded, err := r.URITemplate.Expand(vars)
	if err != nil {
		return err
	}

	u, err := url.Parse(expanded)
	if err != nil {
		return err
	}

	c.SetMethod(r.Method)

	if baseURL != nil {
		u = baseURL.ResolveReference(u)
	}
	return c.SetURL(u)
}

func (r *Resource) Resource(name string) (*Resource, bool) {
	for _, c := range r.Resources {
		if c.Name == name {
			return c, true
		}
	}
	return nil, false
}
