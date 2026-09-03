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

//go:generate go tool counterfeiter -generate

//counterfeiter:generate . ResolvedResource

type Model struct {
	Services []*Service
	VarSets  []*VarSet
	Flows    []*Flow
	Mixins   []*Mixin

	cacheByName        map[string]*Service
	cacheVarSetsByName map[string]*VarSet
	cacheMixinsByName  map[string]*Mixin
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
	Secrets     []*Secret
}

type Server struct {
	Name        string
	Comment     string
	Description string
	Tags        []string
	Title       string
	BaseURL     string
	Headers     Values
	Query       Values
	Form        Values
	Links       []Link
	Vars        map[string]any
	Auth        Auth
	Output      []*OutputConfig
	Secrets     []*Secret
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
	Headers     Values
	Query       Values
	Form        Values
	Links       []Link
	Command     []string
	Body        any
	RawBody     any
	Vars        map[string]any
	Auth        Auth
	Output      []*OutputConfig
}

type Endpoint struct {
	Name        string
	Comment     string
	Title       string
	Description string
	Tags        []string
	Method      string
	Headers     Values
	Query       Values
	Form        Values
	Links       []Link
	Body        any
	RawBody     any
	Vars        map[string]any
	Auth        Auth
	Output      []*OutputConfig
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
	Tags        []string
	Links       []Link
	Vars        map[string]map[string]any
}

type Mixin struct {
	Name        string
	Comment     string
	Title       string
	Description string
	Tags        []string
	Links       []Link
	Method      string
	Headers     Values
	Query       Values
	Form        Values
	Body        any
	RawBody     any
	Vars        map[string]any
	Auth        Auth
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

type Secret struct {
	Name        string
	Comment     string
	Title       string
	Description string
	Links       []Link
	Provider    SecretProvider
}

type SecretProvider interface {
	secretProviderSigil()
}

type ExecSecret struct {
	Command string
}

type FileSecret struct {
	Path                       string
	PreserveTrailingWhitespace bool
}

type Flow struct {
	Name        string
	Comment     string
	Title       string
	Description string
	Tags        []string
	Links       []Link
	Steps       []*Step
	Vars        map[string]any
}

type Step struct {
	Name        string
	Comment     string
	Title       string
	Description string
	Tags        []string
	Links       []Link
	Method      string
	Headers     Values
	Form        Values
	Body        any
	RawBody     any
	Vars        map[string]any
	StepType    StepType
}

type StepType interface {
	stepTypeSigil()
}

type SpecStep struct {
	Spec string
}

type URLStep struct {
	URL string
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

type TSVOutput struct {
	Fields   []string
	Dig      string
	JMESPath string
	Comma    string
	UseCRLF  bool
}

type TableOutput struct {
	Fields              []string
	Dig                 string
	JMESPath            string
	MinWidth            int
	TabWidth            int
	Padding             int
	PadChar             string
	FilterHTML          bool
	AlignRight          bool
	Debug               bool
	StripEscape         bool
	DiscardEmptyColumns bool
}

type Client interface {
	Item
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

	// Mixins obtains the mixins which were selected for the request, in the
	// order that they are applied.
	Mixins() []*Mixin

	// TODO: These should probably be via request
	Output() []*OutputConfig
	Secrets() []*Secret
	Client() Client

	EvalRequest(baseURL *url.URL, vars map[string]any) (*Request, error)
}

type resolvedResource struct {
	endpoint *Endpoint
	lineage  []*Resource
	server   *Server
	service  *Service
	mixins   []*Mixin
}

var looksLikeURLPattern = regexp.MustCompile(`^(unix|https?)://`)

// New creates a new model from configuration files
func New(files ...*config.File) *Model {
	services := []*Service{}
	varSets := make([]*VarSet, 0)
	flows := make([]*Flow, 0)
	mixins := make([]*Mixin, 0)

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
		for _, v := range file.Flows {
			flows = append(flows, flow(v))
		}
		for _, v := range file.Mixins {
			mixins = append(mixins, mixin(v))
		}
	}

	slices.SortStableFunc(services, serviceByName2)
	return &Model{
		Services: services,
		VarSets:  varSets,
		Flows:    flows,
		Mixins:   mixins,
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
	svc, ok := m.servicesByName()[name]
	return svc, ok
}

func (m *Model) Server(spec ServiceSpec) (*Server, bool) {
	svc, ok := m.Service(spec.ServiceName())
	if !ok {
		return nil, false
	}
	if len(spec) < 2 {
		if len(svc.Servers) == 0 {
			return nil, false
		}
		return svc.Servers[0], true
	}
	return svc.Server(spec[1])
}

func (m *Model) VarSet(name string) (*VarSet, bool) {
	svc, ok := m.varSetsByName()[name]
	return svc, ok
}

// Mixin obtains the mixin with the given name.
func (m *Model) Mixin(name string) (*Mixin, bool) {
	mx, ok := m.mixinsByName()[name]
	return mx, ok
}

func (m *Model) Flow(name string) (*Flow, bool) {
	for _, f := range m.Flows {
		if f.Name == name {
			return f, true
		}
	}
	return nil, false
}

func (m *Model) servicesByName() map[string]*Service {
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

func (m *Model) varSetsByName() map[string]*VarSet {
	if m.cacheVarSetsByName == nil {
		m.cacheVarSetsByName = map[string]*VarSet{}
		for _, v := range m.VarSets {
			if v.Name != "" {
				m.cacheVarSetsByName[v.Name] = v
			}
		}
	}
	return m.cacheVarSetsByName
}

func (m *Model) mixinsByName() map[string]*Mixin {
	if m.cacheMixinsByName == nil {
		m.cacheMixinsByName = map[string]*Mixin{}
		for _, v := range m.Mixins {
			if v.Name != "" {
				m.cacheMixinsByName[v.Name] = v
			}
		}
	}
	return m.cacheMixinsByName
}

// Resolve locates the resource named by the spec, optionally within the named
// server and using the given request method.  Any mixins which are named are
// applied as the last layer of the resolution.
func (m *Model) Resolve(spec ServiceSpec, server string, method string, mixins ...string) (ResolvedResource, error) {
	if len(spec) == 0 {
		return nil, fmt.Errorf("no service specified")
	}
	svc, ok := m.Service(spec[0])
	if !ok {
		return nil, fmt.Errorf("service not found: %q", spec[0])
	}

	var svr *Server
	if len(svc.Servers) > 0 {
		svr = svc.Servers[0]
		if server != "" {
			svr, ok = svc.Server(server)
			if !ok {
				return nil, fmt.Errorf("no server %q defined for service %q", server, svc.Name)
			}
		}
	}

	selected, err := m.selectMixins(mixins)
	if err != nil {
		return nil, err
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

	// A mixin can name the request method, though an explicit method wins
	ep := findEndpointOrDefault(current, cmp.Or(method, mixinMethod(selected)), spec)
	if ep == nil {
		// TODO It may be the case that this implies GET
		return nil, fmt.Errorf("no endpoint defined for %v", spec.Path())
	}

	return &resolvedResource{
		service:  svc,
		lineage:  lineage,
		endpoint: ep,
		server:   svr,
		mixins:   selected,
	}, nil
}

// selectMixins looks up each mixin by name, preserving the order in which
// they were named
func (m *Model) selectMixins(names []string) ([]*Mixin, error) {
	if len(names) == 0 {
		return nil, nil
	}

	res := make([]*Mixin, len(names))
	for i, name := range names {
		mx, ok := m.Mixin(name)
		if !ok {
			return nil, fmt.Errorf("mixin not found: %q", name)
		}
		res[i] = mx
	}
	return res, nil
}

// mixinMethod obtains the request method which the mixins name, where the
// last one to name it wins
func mixinMethod(mixins []*Mixin) string {
	var res string
	for _, mx := range mixins {
		res = cmp.Or(mx.Method, res)
	}
	return res
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

func (r *resolvedResource) Mixins() []*Mixin {
	return r.mixins
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

func mergeQuery(u *url.URL, newVals url.Values) {
	if u == nil {
		return
	}
	// Parse existing query parameters
	q := u.Query()

	// Merge new values
	for key, vals := range newVals {
		for _, v := range vals {
			q.Add(key, v) // Append instead of overwriting
		}
	}

	// Encode back into the URL
	u.RawQuery = q.Encode()
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
		nil,
	)
}

func (r *resolvedResource) Secrets() []*Secret {
	return locate(
		r,
		reduceSecrets,
		[]*Secret{},
		nil,
		nil,
		(*Server).secrets,
		(*Service).secrets,
		nil,
	)
}

func resolveHeaders(r ResolvedResource) http.Header {
	return locate(
		r,
		reduceValues,
		Values{},
		func(d *Endpoint) Values { return d.Headers },
		func(r *Resource) Values { return r.Headers },
		func(s *Server) Values { return s.Headers },
		nil,
		func(m *Mixin) Values { return m.Headers },
	).ToHeader()
}

func resolveQuery(r ResolvedResource) url.Values {
	return locate(
		r,
		reduceValues,
		Values{},
		func(d *Endpoint) Values { return d.Query },
		func(r *Resource) Values { return r.Query },
		func(s *Server) Values { return s.Query },
		nil,
		func(m *Mixin) Values { return m.Query },
	).ToURLValues()
}

func resolveVars(r ResolvedResource) map[string]any {
	return locate(
		r,
		reduceVars,
		map[string]any{},
		func(d *Endpoint) map[string]any { return d.Vars },
		func(r *Resource) map[string]any { return r.Vars },
		func(s *Server) map[string]any { return s.Vars },
		func(s *Service) map[string]any { return s.Vars },
		func(m *Mixin) map[string]any { return m.Vars },
	)
}

func resolveLinks2(r ResolvedResource) []Link {
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
	for _, m := range r.Mixins() {
		result = append(result, m.Links...)
	}
	return result
}

func resolveAuth(r ResolvedResource) Auth {
	return locate(
		r,
		reduceAuth,
		nil,
		(*Endpoint).auth,
		(*Resource).auth,
		(*Server).auth,
		(*Service).auth,
		(*Mixin).auth,
	)
}

// locate reduces a value across each layer of the resolved resource, from the
// broadest to the most specific.  Mixins, which the caller selected explicitly,
// are the last layer of all.
func locate[T any](
	r ResolvedResource,
	reducer func(T, T) T,
	initial T,
	onEndpoint func(*Endpoint) T,
	onResource func(*Resource) T,
	onServer func(*Server) T,
	onService func(*Service) T,
	onMixin func(*Mixin) T) T {

	res := initial

	if onService != nil && r.Service() != nil {
		res = reducer(res, onService(r.Service()))
	}

	if onResource != nil {
		for _, l := range r.Lineage() {
			res = reducer(res, onResource(l))
		}
	}

	if onEndpoint != nil && r.Endpoint() != nil {
		res = reducer(res, onEndpoint(r.Endpoint()))
	}

	if onServer != nil && r.Server() != nil {
		res = reducer(res, onServer(r.Server()))
	}

	if onMixin != nil {
		for _, m := range r.Mixins() {
			res = reducer(res, onMixin(m))
		}
	}

	return res
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
func (m *Mixin) auth() Auth    { return m.Auth }

func (e *Endpoint) output() []*OutputConfig { return e.Output }
func (r *Resource) output() []*OutputConfig { return r.Output }
func (s *Server) output() []*OutputConfig   { return s.Output }
func (s *Service) output() []*OutputConfig  { return s.Output }

func (s *Server) secrets() []*Secret  { return s.Secrets }
func (s *Service) secrets() []*Secret { return s.Secrets }

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

func reduceVars(x, y map[string]any) map[string]any {
	maps.Copy(x, y)
	return x
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

func reduceSecrets(x, y []*Secret) []*Secret {
	byName := make(map[string]int)
	for i, s := range x {
		if s.Name != "" {
			byName[s.Name] = i
		}
	}

	// Closer level (y) wins for secrets with the same name
	for _, s := range y {
		if idx, ok := byName[s.Name]; ok && s.Name != "" {
			x[idx] = s
		} else {
			if s.Name != "" {
				byName[s.Name] = len(x)
			}
			x = append(x, s)
		}
	}
	return x
}

func sameType(x, y any) bool {
	return reflect.TypeOf(x) == reflect.TypeOf(y)
}

func (*GRPCClient) clientSigil() {}
func (*HTTPClient) clientSigil() {}

func (*GRPCClient) itemSigil() {}
func (*HTTPClient) itemSigil() {}

func (*BasicAuth) authSigil() {}

func (*ExecSecret) secretProviderSigil() {}
func (*FileSecret) secretProviderSigil() {}

func (*TemplateOutput) outputFilterSigil() {}
func (*JMESPathOutput) outputFilterSigil() {}
func (*XPathOutput) outputFilterSigil()    {}
func (*DigOutput) outputFilterSigil()      {}
func (*JSONOutput) outputFilterSigil()     {}
func (*XMLOutput) outputFilterSigil()      {}
func (*YAMLOutput) outputFilterSigil()     {}
func (*TSVOutput) outputFilterSigil()      {}
func (*TableOutput) outputFilterSigil()    {}

func (*SpecStep) stepTypeSigil() {}
func (*URLStep) stepTypeSigil()  {}
