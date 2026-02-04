// Copyright 2025, 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package model

import (
	"cmp"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"os"
	"path"
	"reflect"
	"regexp"
	"slices"
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
	Comment     string
	Title       string
	Description string
	Servers     []*Server
	Links       []Link
	Resource    *Resource
	Vars        map[string]any
	Client      Client
	Auth        Auth
}

type Server struct {
	Name        string
	Comment     string
	Description string
	Title       string
	BaseURL     string
	Headers     map[string][]string
	Form        map[string][]string
	Links       []Link
	Vars        map[string]any
	Auth        Auth
}

type Resource struct {
	Name        string
	Comment     string
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
	Auth        Auth
}

type Endpoint struct {
	Name        string
	Comment     string
	Title       string
	Description string
	Method      string
	Headers     map[string][]string
	Form        map[string][]string
	Links       []Link
	Body        any
	RawBody     any
	Vars        map[string]any
	Auth        Auth
}

type Link struct {
	HRef       string
	HRefLang   string
	Audience   string
	Rel        string
	Title      string
	Type       string
	IsTemplate bool
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

type Auth interface {
	authSigil()
}

type BasicAuth struct {
	User     string
	Password string
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
	Auth() Auth
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
	auth            Auth
}

var looksLikeURLPattern = regexp.MustCompile(`^(unix|https?)://`)

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
	links := resolveLinks(
		expandLinks(r.combinedLinks(), expander),
		baseURITemplate.String(),
		combinedVars,
	)

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
		auth:            expandAuth(r.combinedAuth(), expander),
	}, nil
}

func resolveLinks(links []Link, base string, vars map[string]any) []Link {
	for i := range links {
		if links[i].IsTemplate {
			url, err := resolveURL(base, []string{links[i].HRef}, vars)
			if err == nil {
				links[i].HRef = url.String()
			}
		}
	}
	return links
}

func resolveURL(base string, prefix []string, vars map[string]any) (*url.URL, error) {
	// Treat as absolute URI when it is qualified
	if len(prefix) > 0 && looksLikeURLPattern.MatchString(prefix[0]) {
		base = prefix[0]
		prefix = prefix[1:]
	}

	if base != "" && len(prefix) > 0 {
		base = base + "/"
	}

	template := base + path.Join(prefix...)

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

	return u.JoinPath(), nil
}

func (r request) URL() (*url.URL, error) {
	base := r.baseURITemplate.String()
	return resolveURL(base, r.prefix, r.vars)
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

func (r request) Auth() Auth {
	return r.auth
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
	return locate(
		r,
		reduceHeader,
		http.Header{},
		func(d *Endpoint) http.Header { return d.Headers },
		func(r *Resource) http.Header { return r.Headers },
		func(s *Server) http.Header { return s.Headers },
		nil,
	)
}

func (r *resolvedResource) combinedVars() map[string]any {
	return locate(
		r,
		reduceVars,
		map[string]any{},
		func(d *Endpoint) map[string]any { return d.Vars },
		func(r *Resource) map[string]any { return r.Vars },
		func(s *Server) map[string]any { return s.Vars },
		func(s *Service) map[string]any { return s.Vars },
	)
}

func (r *resolvedResource) combinedLinks() []Link {
	var result []Link
	if r.Server() != nil {
		result = append(result, r.Server().Links...)
	}
	if r.Service() != nil {
		result = append(result, r.Service().Links...)
	}
	for _, l := range r.Lineage() {
		result = append(result, l.Links...)
	}
	if r.Endpoint() != nil {
		result = append(result, r.Endpoint().Links...)
	}
	return result
}

func (r *resolvedResource) combinedAuth() Auth {
	return locate(
		r,
		reduceAuth,
		nil,
		(*Endpoint).auth,
		(*Resource).auth,
		(*Server).auth,
		(*Service).auth,
	)
}

func locate[T any](
	r *resolvedResource,
	reducer func(T, T) T,
	initial T,
	onEndpoint func(*Endpoint) T,
	onResource func(*Resource) T,
	onServer func(*Server) T,
	onService func(*Service) T) T {

	res := initial

	if onEndpoint != nil && r.Endpoint() != nil {
		res = reducer(res, onEndpoint(r.Endpoint()))
	}

	if onResource != nil {
		if r.Resource() != nil {
			res = reducer(res, onResource(r.Resource()))
		}
		for _, l := range r.Lineage() {
			res = reducer(res, onResource(l))
		}
	}
	if onServer != nil && r.Server() != nil {
		res = reducer(res, onServer(r.Server()))
	}
	if onService != nil && r.Service() != nil {
		res = reducer(res, onService(r.Service()))
	}

	return res
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

func (e *Endpoint) auth() Auth { return e.Auth }
func (r *Resource) auth() Auth { return r.Auth }
func (s *Server) auth() Auth   { return s.Auth }
func (s *Service) auth() Auth  { return s.Auth }

func reduceAuth(x, y Auth) Auth {
	if y == nil {
		return x
	}

	// If the operand specifies a different value from the union, it
	// automatically wins
	if sameType(x, y) {
		switch bx := x.(type) {
		case *BasicAuth:
			by := y.(*BasicAuth)
			return &BasicAuth{
				User:     cmp.Or(by.User, bx.User),
				Password: cmp.Or(by.Password, bx.Password),
			}
		}
	}
	return y
}

func reduceHeader(x, y http.Header) http.Header {
	for k, v := range y {
		if name, ok := strings.CutPrefix(k, "+"); ok {
			x[name] = append(x[name], v...)

		} else if name, ok := strings.CutPrefix(k, "-"); ok {
			x[name] = slices.DeleteFunc(
				x[name],
				func(m string) bool {
					return slices.Contains(v, m)
				},
			)
		} else {
			x[k] = v
		}
	}
	return x
}

func reduceVars(x, y map[string]any) map[string]any {
	maps.Copy(x, y)
	return x
}

func sameType(x, y any) bool {
	return reflect.TypeOf(x) == reflect.TypeOf(y)
}

func (*GRPCClient) clientSigil() {}
func (*HTTPClient) clientSigil() {}
func (*BasicAuth) authSigil()    {}
