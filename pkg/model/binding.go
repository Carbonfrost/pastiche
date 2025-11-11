// Copyright 2025 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package model

import (
	"fmt"
	"io"
	"net/url"
	"path"
	"strings"

	"github.com/Carbonfrost/joe-cli-http/httpclient"
	"github.com/Carbonfrost/joe-cli-http/uritemplates"
	"github.com/Carbonfrost/pastiche/pkg/config"
	"github.com/Carbonfrost/pastiche/pkg/internal/log"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

//counterfeiter:generate . ResolvedResource

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
	Body        any
	RawBody     any
}

type Endpoint struct {
	Name        any
	Description string
	Method      string
	Headers     map[string][]string
	Links       []Link
	Body        any
	RawBody     any
}

type Link struct {
	HRef     string
	HRefLang string
	Audience string
	Rel      string
	Title    string
}

type ResolvedResource interface {
	Service() *Service
	Resource() *Resource
	Endpoint() *Endpoint
	Server() *Server

	URL(baseURL *url.URL, vars uritemplates.Vars) (*url.URL, error)
	Body(vars uritemplates.Vars) io.ReadCloser
}

type resolvedResource struct {
	endpoint *Endpoint
	resource *Resource
	server   *Server
	service  *Service
	uri      []string
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
		servers[i] = server(s)
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

func server(s config.Server) *Server {
	return &Server{
		Name:    s.Name,
		BaseURL: s.BaseURL,
		Headers: s.Headers,
		Links:   links(s.Links),
	}
}

func resource(r config.Resource) *Resource {
	uri, _ := uritemplates.Parse(r.URI)
	res := &Resource{
		Name:        r.Name,
		Description: r.Description,
		URITemplate: uri,
		Headers:     r.Headers,
		Links:       links(r.Links),
		Body:        r.Body,
		RawBody:     r.RawBody,
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
	if r.Query != nil {
		res.Endpoints = append(res.Endpoints, endpoint("QUERY", r.Query))
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
		Body:        r.Body,
		RawBody:     r.RawBody,
	}
}

func links(links []config.Link) []Link {
	res := make([]Link, len(links))
	for i, l := range links {
		res[i] = Link{
			HRef:     l.HRef,
			HRefLang: l.HRefLang,
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

func (c *Model) Resolve(spec ServiceSpec, server string, method string) (ResolvedResource, error) {
	if len(spec) == 0 {
		return nil, fmt.Errorf("no service specified")
	}
	svc, ok := c.Service(spec[0])
	if !ok {
		return nil, fmt.Errorf("service not found: %q", spec[0])
	}

	if len(svc.Servers) == 0 {
		return nil, fmt.Errorf("no servers defined for service %q", svc.Name)
	}

	svr := svc.Servers[0]
	if server != "" {
		svr, ok = svc.Server(server)
		if !ok {
			return nil, fmt.Errorf("no server %q defined for service %q", server, svc.Name)
		}
	}

	current := svc.Resource
	prefix := []string{current.URITemplate.String()}
	for i, p := range spec[1:] {
		current, ok = current.Resource(p)
		if !ok {
			path := ServiceSpec(spec[0 : i+2]).Path()
			return nil, fmt.Errorf("resource not found: %q", path)
		}
		prefix = append(prefix, current.URITemplate.String())
	}

	ep := findEndpointOrDefault(current, method, spec)
	if ep == nil {
		// TODO It may be the case that this implies GET
		return nil, fmt.Errorf("no endpoint defined for %v", spec.Path())
	}

	return &resolvedResource{
		service:  svc,
		resource: current,
		endpoint: ep,
		server:   svr,
		uri:      prefix,
	}, nil
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

func (r *resolvedResource) Service() *Service {
	return r.service
}

func (r *resolvedResource) Resource() *Resource {
	return r.resource
}

func (r *resolvedResource) Endpoint() *Endpoint {
	return r.endpoint
}

func (r *resolvedResource) Server() *Server {
	return r.server
}

func (r *resolvedResource) URL(baseURL *url.URL, vars uritemplates.Vars) (*url.URL, error) {
	template := path.Join(r.uri...)
	tt, err := uritemplates.Parse(template)
	if err != nil {
		return nil, err
	}

	expanded, err := tt.Expand(vars)
	if err != nil {
		return nil, err
	}

	u, err := url.Parse(expanded)
	if err != nil {
		return nil, err
	}

	if baseURL == nil {
		baseURL, err = url.Parse(r.Server().BaseURL)
		if err != nil {
			return nil, err
		}
	}

	u = baseURL.ResolveReference(u)
	return u, err
}

func (r *resolvedResource) Body(vars uritemplates.Vars) io.ReadCloser {
	content := r.bodyContent(vars)
	if content == nil {
		return nil
	}
	return io.NopCloser(content.Read())
}

func (r *resolvedResource) bodyContent(vars uritemplates.Vars) httpclient.Content {
	if r.Endpoint().Body != "" {
		return newTemplateContent(r.Endpoint().Body, vars)
	}
	if r.Endpoint().RawBody != "" {
		return newRawContent(r.Endpoint().RawBody)
	}
	if r.Resource().Body != "" {
		return newTemplateContent(r.Resource().Body, vars)
	}
	if r.Resource().RawBody != "" {
		return newRawContent(r.Resource().RawBody)
	}

	return nil
}

func newRawContent(data any) httpclient.Content {
	return httpclient.NewRawContent(bodyToBytes(data))
}

func findEndpointOrDefault(resource *Resource, method string, spec ServiceSpec) *Endpoint {
	if method != "" {
		ep, ok := resource.Endpoint(method)
		if !ok {
			log.Warnf("warning: method %s is not defined for resource %s", method, spec.Path())
		}
		return ep
	}
	if len(resource.Endpoints) > 0 {
		return resource.Endpoints[0]
	}
	return nil
}
