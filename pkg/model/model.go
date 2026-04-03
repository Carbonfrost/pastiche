// Copyright 2025, 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package model

import (
	"cmp"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"path"
	"reflect"
	"regexp"
	"slices"
	"strings"

	"github.com/Carbonfrost/joe-cli-http/httpclient"
	"github.com/Carbonfrost/joe-cli-http/uritemplates"
	"github.com/Carbonfrost/pastiche/pkg/config"
	"github.com/Carbonfrost/pastiche/pkg/internal/log"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

//counterfeiter:generate . ResolvedResource

type Model struct {
	Services []*Service
	VarSets  []*VarSet

	cacheByName map[string]*Service
}

type Service struct {
	Name        string
	Comment     string
	Title       string
	Description string
	Tags        []string
	Servers     []*Server
	Links       []Link
	Resource    *Resource
	Vars        map[string]any
	Client      Client
	Auth        Auth
	Output      []*OutputConfig
	VarSets     []*VarSet
}

type Server struct {
	Name        string
	Comment     string
	Description string
	Tags        []string
	Title       string
	BaseURL     string
	Headers     map[string][]string
	Form        map[string][]string
	Links       []Link
	Vars        map[string]any
	Auth        Auth
	Output      []*OutputConfig
	VarSets     []*VarSet
}

type Resource struct {
	Name        string
	Comment     string
	Title       string
	Description string
	Tags        []string
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
	Output      []*OutputConfig
	VarSets     []*VarSet
}

type Endpoint struct {
	Name        string
	Comment     string
	Title       string
	Description string
	Tags        []string
	Method      string
	Headers     map[string][]string
	Form        map[string][]string
	Links       []Link
	Body        any
	RawBody     any
	Vars        map[string]any
	Auth        Auth
	Output      []*OutputConfig
	VarSets     []*VarSet
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

type VarSet struct {
	Name        string
	Comment     string
	Title       string
	Description string
	Links       []Link
	Vars        map[string]map[string]any
}

type OutputConfig struct {
	Name            string
	Comment         string
	Title           string
	Description     string
	Links           []Link
	Filter          OutputFilter
	IncludeMetadata bool
}

type OutputFilter interface {
	outputFilterSigil()
}

type TemplateOutput struct {
	Text string
	File string
}

type JMESPathOutput struct {
	Query string
}

type XPathOutput struct {
	Query string
}

type DigOutput struct {
	Query string
}

type JSONOutput struct {
	Pretty bool
}

type XMLOutput struct {
	Pretty bool
}

type YAMLOutput struct {
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

	Auth() Auth
	Output() []*OutputConfig
	Headers() http.Header
	Vars() map[string]any
	VarSets() []*VarSet
	Links() []Link

	EvalRequest(baseURL *url.URL, vars map[string]any) (*Request, error)
}

type resolvedResource struct {
	endpoint *Endpoint
	lineage  []*Resource
	server   *Server
	service  *Service
}

var looksLikeURLPattern = regexp.MustCompile(`^(unix|https?)://`)

// New creates a new model from configuration files
func New(files ...*config.File) *Model {
	services := []*Service{}
	varSets := make([]*VarSet, 0)

	for _, file := range files {
		if file.Service != nil {
			services = append(services, service(*file.Service))
		}
		for _, s := range file.Services {
			services = append(services, service(s))
		}
		for _, v := range file.VarSets {
			varSets = append(varSets, varSet(v))
		}
	}

	slices.SortStableFunc(services, serviceByName2)
	return &Model{
		Services: services,
		VarSets:  varSets,
	}
}

func serviceByName2(x, y *Service) int {
	return cmp.Compare(x.Name, y.Name)
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

func (m *Model) Resolve(spec ServiceSpec, server string, method string) (ResolvedResource, error) {
	if len(spec) == 0 {
		return nil, fmt.Errorf("no service specified")
	}
	svc, ok := m.Service(spec[0])
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

func (r *resolvedResource) EvalRequest(baseURL *url.URL, vars map[string]any) (*Request, error) {
	opts := []RequestOption{
		WithVars(vars),
	}
	if baseURL != nil {
		opts = append(opts, WithBaseURL(baseURL))
	}
	return NewRequest(r, opts...)
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
		base += "/"
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

func (r *resolvedResource) Client() Client {
	var client Client = &HTTPClient{}

	if r.Service() != nil && r.Service().Client != nil {
		client = r.Service().Client
	}

	// TODO Allow combinations of client via lineage
	return client
}

func (r *resolvedResource) Output() []*OutputConfig {
	return locate(
		r,
		reduceOutput,
		[]*OutputConfig{},
		(*Endpoint).output,
		(*Resource).output,
		(*Server).output,
		(*Service).output,
	)
}

func (r *resolvedResource) Headers() http.Header {
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

func (r *resolvedResource) Vars() map[string]any {
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

func (r *resolvedResource) VarSets() []*VarSet {
	return locate(
		r,
		reduceVarSet,
		make([]*VarSet, 0),
		(*Endpoint).varSets,
		(*Resource).varSets,
		(*Server).varSets,
		(*Service).varSets,
	)
}

func (r *resolvedResource) Links() []Link {
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

func (r *resolvedResource) Auth() Auth {
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

func (e *Endpoint) output() []*OutputConfig { return e.Output }
func (r *Resource) output() []*OutputConfig { return r.Output }
func (s *Server) output() []*OutputConfig   { return s.Output }
func (s *Service) output() []*OutputConfig  { return s.Output }

func (e *Endpoint) varSets() []*VarSet { return e.VarSets }
func (r *Resource) varSets() []*VarSet { return r.VarSets }
func (s *Server) varSets() []*VarSet   { return s.VarSets }
func (s *Service) varSets() []*VarSet  { return s.VarSets }

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

func reduceVarSet(x, y []*VarSet) []*VarSet {
	// TODO Duplicate names between varsets should be consolidated
	return append(x, y...)
}

func reduceOutput(x, y []*OutputConfig) []*OutputConfig {
	byName := make(map[string]*OutputConfig)
	for _, o := range x {
		if o.Name != "" {
			byName[o.Name] = o
		}
	}

	// Merge or append from y
	for _, o := range y {
		if o.Name != "" {
			if existing, ok := byName[o.Name]; ok {
				// Filter at closer level (y) wins
				if o.Filter != nil {
					existing.Filter = o.Filter
				}
			} else {
				byName[o.Name] = o
				x = append(x, o)
			}
		} else {
			// Unnamed outputs are always appended
			x = append(x, o)
		}
	}

	return x
}

func sameType(x, y any) bool {
	return reflect.TypeOf(x) == reflect.TypeOf(y)
}

func (*GRPCClient) clientSigil() {}
func (*HTTPClient) clientSigil() {}

func (*BasicAuth) authSigil() {}

func (*TemplateOutput) outputFilterSigil() {}
func (*JMESPathOutput) outputFilterSigil() {}
func (*XPathOutput) outputFilterSigil()    {}
func (*DigOutput) outputFilterSigil()      {}
func (*JSONOutput) outputFilterSigil()     {}
func (*XMLOutput) outputFilterSigil()      {}
func (*YAMLOutput) outputFilterSigil()     {}
