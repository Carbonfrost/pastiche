// Copyright 2025, 2026 The Pastiche Authors. All rights reserved.
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
	Services    []*Service
	cacheByName map[string]*Service
}

type Service struct {
	Name        string
	Title       string
	Description string
	Servers     []*Server
	Links       []Link
	Resource    *Resource
	Vars        map[string]any
	Client      Client
}

type Server struct {
	Name        string
	Description string
	Title       string
	BaseURL     string
	Headers     map[string][]string
	Form        map[string][]string
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
	Form        map[string][]string
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
	Form        map[string][]string
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
	Type     string
}

type Client interface {
	clientSigil()
}

type GRPCClient struct {
	DisableReflection bool
	ProtoSet          string
	Plaintext         bool
}

type HTTPClient struct {
}

// ResolvedResource represents the resource which was selected by its name
type ResolvedResource interface {
	Service() *Service
	Resource() *Resource
	Lineage() []*Resource
	Endpoint() *Endpoint
	Server() *Server
	Client() Client

	EvalRequest(baseURL *url.URL, vars map[string]any) (Request, error)
}

// Request represents a request that can be generated
type Request interface {
	URL() (*url.URL, error)
	Body() io.ReadCloser
	Headers() http.Header
	Vars() map[string]any
	Links() []Link
}

type resolvedResource struct {
	endpoint *Endpoint
	lineage  []*Resource
	server   *Server
	service  *Service
}

type request struct {
	baseURITemplate *uritemplates.URITemplate
	vars            map[string]any
	prefix          []string
	headers         http.Header
	body            io.ReadCloser
	links           []Link
}

func New(c *config.Config) *Model {
	res := &Model{
		Services: make([]*Service, len(c.Services)),
	}
	for i, v := range c.Services {
		res.Services[i] = service(v)
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

func (m *Model) Service(name string) (*Service, bool) {
	svc, ok := m.byName()[name]
	return svc, ok
}

func (m *Model) byName() map[string]*Service {
	if m.cacheByName == nil {
		m.cacheByName = map[string]*Service{}
		for _, v := range m.Services {
			if v.Name != "" {
				m.cacheByName[v.Name] = v
			}
		}
	}
	return m.cacheByName
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
	var baseURITemplate *uritemplates.URITemplate
	if baseURL == nil {
		// Treat server baseURL as a potential URI template
		baseURITemplate, err = uritemplates.Parse(r.Server().BaseURL)
		if err != nil {
			return nil, err
		}
	} else {
		baseURITemplate, _ = uritemplates.Parse(baseURL.String())
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
	links := r.combinedLinks()

	body := func() io.ReadCloser {
		content := r.bodyContent(combinedVars)
		if content == nil {
			return nil
		}

		return io.NopCloser(content.Read())
	}()

	return request{
		baseURITemplate: baseURITemplate,
		vars:            combinedVars,
		prefix:          prefix,
		headers:         headers,
		body:            body,
		links:           links,
	}, nil
}

func (r request) URL() (*url.URL, error) {
	base := r.baseURITemplate.String()
	if base != "" {
		base = base + "/"
	}
	template := base + path.Join(r.prefix...)

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

	return u.JoinPath(), nil
}

func (r request) Body() io.ReadCloser {
	return r.body
}

func (r request) Headers() http.Header {
	return r.headers
}

func (r request) Vars() map[string]any {
	return r.vars
}

func (r request) Links() []Link {
	return r.links
}

func (r *resolvedResource) Client() Client {
	var client Client = &HTTPClient{}

	if r.Service() != nil && r.Service().Client != nil {
		client = r.Service().Client
	}

	// TODO Allow combinations of client via lineage
	return client
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

func (r *resolvedResource) combinedLinks() []Link {
	var result []Link
	if r.Server() != nil {
		result = append(result, r.Server().Links...)
	}
	for _, l := range r.Lineage() {
		result = append(result, l.Links...)
	}
	if r.Endpoint() != nil {
		result = append(result, r.Endpoint().Links...)
	}
	return result
}

func (r *resolvedResource) bodyContent(vars map[string]any) httpclient.Content {
	if r.Endpoint().Form != nil {
		return newFormContent(r.Endpoint().Form, vars)
	}
	if r.Endpoint().Body != "" {
		return newTemplateContent(r.Endpoint().Body, vars)
	}
	if r.Endpoint().RawBody != "" {
		return newRawContent(r.Endpoint().RawBody)
	}
	if r.Resource().Form != nil {
		return newFormContent(r.Resource().Form, vars)
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

func (*GRPCClient) clientSigil() {}
func (*HTTPClient) clientSigil() {}
