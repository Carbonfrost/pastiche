// Copyright 2025 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package model

import (
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/Carbonfrost/joe-cli-http/httpclient"
	"github.com/Carbonfrost/joe-cli-http/httpclient/expr"
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
	Resource    *Resource
	Vars        map[string]any
}

type Server struct {
	Name        string
	Description string
	Title       string
	BaseURL     string
	Headers     map[string][]string
	Links       []Link
	Vars        map[string]any
}

type Resource struct {
	Name        string
	Title       string
	Description string
	Resources   []*Resource
	Endpoints   []*Endpoint
	URITemplate *uritemplates.URITemplate
	Headers     map[string][]string
	Links       []Link
	Command     []string
	Body        any
	RawBody     any
	Vars        map[string]any
}

type Endpoint struct {
	Name        string
	Title       string
	Description string
	Method      string
	Headers     map[string][]string
	Links       []Link
	Body        any
	RawBody     any
	Vars        map[string]any
}

type Link struct {
	HRef     string
	HRefLang string
	Audience string
	Rel      string
	Title    string
}

// ResolvedResource represents the resource which was selected by its name
type ResolvedResource interface {
	Service() *Service
	Resource() *Resource
	Lineage() []*Resource
	Endpoint() *Endpoint
	Server() *Server

	EvalRequest(baseURL *url.URL, vars map[string]any) (Request, error)
}

// Request represents a request that can be generated
type Request interface {
	URL() (*url.URL, error)
	Body() io.ReadCloser
	Header() http.Header
	Vars() map[string]any
}

type resolvedResource struct {
	endpoint *Endpoint
	lineage  []*Resource
	server   *Server
	service  *Service
}

type request struct {
	baseURL *url.URL
	vars    map[string]any
	prefix  []string
	headers http.Header
	body    io.ReadCloser
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

	lineage := []*Resource{svc.Resource}
	current := svc.Resource
	for i, p := range spec[1:] {
		current, ok = current.Resource(p)
		if !ok {
			path := ServiceSpec(spec[0 : i+2]).Path()
			return nil, fmt.Errorf("resource not found: %q", path)
		}
		lineage = append(lineage, current)
	}

	ep := findEndpointOrDefault(current, method, spec)
	if ep == nil {
		// TODO It may be the case that this implies GET
		return nil, fmt.Errorf("no endpoint defined for %v", spec.Path())
	}

	return &resolvedResource{
		service:  svc,
		lineage:  lineage,
		endpoint: ep,
		server:   svr,
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
	return r.lineage[len(r.lineage)-1]
}

func (r *resolvedResource) Lineage() []*Resource {
	return r.lineage
}

func (r *resolvedResource) Endpoint() *Endpoint {
	return r.endpoint
}

func (r *resolvedResource) Server() *Server {
	return r.server
}

func (r *resolvedResource) EvalRequest(baseURL *url.URL, vars map[string]any) (Request, error) {
	prefix := make([]string, len(r.lineage))
	for i, c := range r.lineage {
		prefix[i] = c.URITemplate.String()
	}

	var err error
	if baseURL == nil {
		baseURL, err = url.Parse(r.Server().BaseURL)
		if err != nil {
			return nil, err
		}
	}

	combinedVars := r.combinedVars()
	maps.Copy(combinedVars, vars)

	expander := expr.ComposeExpanders(
		expr.Prefix("env", func(k string) any {
			return os.Getenv(k)
		}),
		expr.Prefix("var", expr.ExpandMap(combinedVars)),
		expr.ExpandMap(combinedVars),
	)

	headers := expandHeader(r.combinedHeaders(), expander)

	body := func() io.ReadCloser {
		content := r.bodyContent(combinedVars)
		if content == nil {
			return nil
		}

		return io.NopCloser(content.Read())
	}()

	return request{
		baseURL: baseURL,
		vars:    vars,
		prefix:  prefix,
		headers: headers,
		body:    body,
	}, nil
}

func (r request) URL() (*url.URL, error) {
	template := path.Join(r.prefix...)
	tt, err := uritemplates.Parse(template)
	if err != nil {
		return nil, err
	}

	expanded, err := tt.Expand(r.vars)
	if err != nil {
		return nil, err
	}

	u, err := url.Parse(expanded)
	if err != nil {
		return nil, err
	}

	u = r.baseURL.ResolveReference(u)
	return u, err
}

func (r request) Body() io.ReadCloser {
	return r.body
}

func (r request) Header() http.Header {
	return r.headers
}

func (r request) Vars() map[string]any {
	return r.vars
}

func (r *resolvedResource) combinedHeaders() http.Header {
	result := http.Header{}
	if r.Server() != nil {
		maps.Copy(result, r.Server().Headers)
	}
	for _, l := range r.Lineage() {
		maps.Copy(result, l.Headers)
	}
	if r.Endpoint() != nil {
		maps.Copy(result, r.Endpoint().Headers)
	}
	return result
}

func (r *resolvedResource) combinedVars() map[string]any {
	result := map[string]any{}
	if r.Server() != nil {
		maps.Copy(result, r.Server().Vars)
	}
	for _, l := range r.Lineage() {
		maps.Copy(result, l.Vars)
	}
	if r.Endpoint() != nil {
		maps.Copy(result, r.Endpoint().Vars)
	}
	return result
}

func (r *resolvedResource) bodyContent(vars map[string]any) httpclient.Content {
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
