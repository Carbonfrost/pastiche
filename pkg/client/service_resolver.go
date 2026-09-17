// Copyright 2025, 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/Carbonfrost/joe-cli-http/httpclient"
	"github.com/Carbonfrost/pastiche/pkg/model"
)

// Location represents an HTTP client location and its resolved resource or
// operation in the configuration
type Location interface {
	httpclient.Location

	Resolved() model.ResolvedResource
}

// LocationResolver represents an HTTP client location resolver
type LocationResolver interface {
	httpclient.LocationResolver

	BaseURL() *url.URL
	Vars() map[string]any

	// AddContextParam records that the variable named name should be
	// resolved from path within the context varset, unless an explicit
	// value for name was set via AddVar.
	AddContextParam(name, path string) error
}

type contextParam struct {
	name string
	path string
}

type serviceResolver struct {
	root   func(context.Context) *model.ServiceSpec
	server func(context.Context) string
	method func(context.Context) string
	mixins func(context.Context) []string
	vars   map[string]any
	base   *url.URL
	config func(context.Context) *model.Model

	context       func(context.Context) string
	contextParams []contextParam
}

type pasticheLocation struct {
	httpclient.Middleware

	u        *url.URL
	resolved model.ResolvedResource
}

type contextKey string

var looksLikeURLPattern = regexp.MustCompile(`^(unix|https?)://`)

// NewServiceResolver creates a service resolver compatible with the client.
func NewServiceResolver(
	c func(context.Context) *model.Model,
	root func(context.Context) *model.ServiceSpec,
	server func(context.Context) string,
	method func(context.Context) string,
	mixins func(context.Context) []string,
	context func(context.Context) string,
) LocationResolver {
	return &serviceResolver{
		root:    root,
		server:  server,
		method:  method,
		mixins:  mixins,
		context: context,
		config:  c,
		vars:    map[string]any{},
	}
}

func (s *serviceResolver) selectedMixins(c context.Context) []string {
	if s.mixins == nil {
		return nil
	}
	return s.mixins(c)
}

func (s *serviceResolver) Add(location string) error {
	return fmt.Errorf("multiple locations not supported")
}

func (s *serviceResolver) AddVar(name string, value any) error {
	s.vars[name] = value
	return nil
}

func (s *serviceResolver) AddContextParam(name, path string) error {
	s.contextParams = append(s.contextParams, contextParam{name: name, path: path})
	return nil
}

func (s *serviceResolver) Vars() map[string]any {
	return s.vars
}

// contextName obtains the name of the varset selected as context or implied
// from the requested service
func (s *serviceResolver) contextName(ctx context.Context) string {
	if s.context != nil {
		if name := s.context(ctx); name != "" {
			return name
		}
	}
	return (*s.root(ctx)).ServiceName()
}

func (s *serviceResolver) contextVarSet(ctx context.Context) *model.VarSet {
	vs, _ := s.config(ctx).VarSet(s.contextName(ctx))
	return vs
}

func (s *serviceResolver) resolveContextVars(ctx context.Context) error {
	if len(s.contextParams) == 0 {
		return nil
	}

	vs := s.contextVarSet(ctx)
	if vs == nil {
		return fmt.Errorf("varset not found: %q", s.contextName(ctx))
	}

	for _, p := range s.contextParams {
		if _, seen := s.vars[p.name]; seen {
			continue
		}
		v, ok := vs.Resolve(p.name, p.path)
		if !ok {
			return fmt.Errorf("cannot resolve context param %q using %q in varset %q", p.name, p.path, vs.Name)
		}
		s.vars[p.name] = v
	}
	s.contextParams = nil
	return nil
}

func (s *serviceResolver) BaseURL() *url.URL {
	return s.base
}

func (s *serviceResolver) SetBaseURL(base *url.URL) error {
	if base == nil {
		s.base = base
		return nil
	}

	s.base = s.base.ResolveReference(base)
	return nil
}

func (s *serviceResolver) Resolve(c context.Context) ([]httpclient.Location, error) {
	spec := *s.root(c)

	if looksLikeURL(spec[0]) {
		r := httpclient.NewDefaultLocationResolver()
		for _, s := range spec {
			r.Add(s)
		}
		return r.Resolve(c)
	}

	if err := s.resolveContextVars(c); err != nil {
		return nil, err
	}

	merged, err := s.config(c).Resolve(spec, s.server(c), s.method(c), s.selectedMixins(c)...)
	if err != nil {
		return nil, err
	}

	location, err := newLocation(merged, s.requestOptions(c)...)
	if err != nil {
		return nil, err
	}

	return []httpclient.Location{
		location,
	}, nil
}

func (s *serviceResolver) resolveRequest(c context.Context) (*model.Request, error) {
	if err := s.resolveContextVars(c); err != nil {
		return nil, err
	}

	spec := *s.root(c)
	merged, err := s.config(c).Resolve(spec, s.server(c), s.method(c), s.selectedMixins(c)...)
	if err != nil {
		return nil, err
	}
	return model.NewRequest(merged, s.requestOptions(c)...)
}

func (s *serviceResolver) requestOptions(c context.Context) []model.RequestOption {
	m := s.config(c)
	return []model.RequestOption{
		model.WithBaseURL(s.base),
		model.WithVars(s.vars),
		model.WithModel(m),
		model.WithContext(s.contextVarSet(c)),
	}
}

func (s *serviceResolver) resolveResource(c context.Context) (model.ResolvedResource, error) {
	spec := *s.root(c)
	return s.config(c).Resolve(spec, s.server(c), s.method(c), s.selectedMixins(c)...)
}

func newLocation(resolved model.ResolvedResource, opts ...model.RequestOption) (*pasticheLocation, error) {
	merged, err := model.NewRequest(resolved, opts...)
	if err != nil {
		return nil, err
	}

	loc := merged.URL
	var (
		endpointMethod  httpclient.Middleware
		requireEndpoint httpclient.MiddlewareFunc = func(req *http.Request) error {
			if resolved.Endpoint() == nil {
				return errors.New("no endpoint defined for service/spec")
			}
			return nil
		}
	)

	if resolved.Endpoint() != nil {
		endpointMethod = withMethod(resolved.Endpoint().Method)
	}

	return &pasticheLocation{
		Middleware: httpclient.ComposeMiddleware(
			requireEndpoint,
			httpclient.WithHeaders(merged.Headers),
			endpointMethod,
			withBody(merged.Body),
			withAuth(merged.Auth),
		),
		resolved: resolved,
		u:        loc,
	}, nil
}

func (l *pasticheLocation) URL(ctx context.Context) (context.Context, *url.URL, error) {
	return ctx, l.u, nil
}

func (l *pasticheLocation) Resolved() model.ResolvedResource {
	return l.resolved
}

func withMethod(method string) httpclient.MiddlewareFunc {
	return func(r *http.Request) error {
		r.Method = method
		return nil
	}
}

func withBody(body io.ReadCloser) httpclient.MiddlewareFunc {
	return func(r *http.Request) error {
		if body != nil {
			r.Body = body
		}
		return nil
	}
}

func withAuth(a model.Auth) httpclient.MiddlewareFunc {
	if a == nil {
		return nil
	}
	return func(r *http.Request) error {
		switch auth := a.(type) {
		case *model.BasicAuth:
			encodedAuth := base64.StdEncoding.EncodeToString([]byte(auth.User + ":" + auth.Password))
			r.Header.Add("authorization", "Basic "+encodedAuth)
		}
		return nil
	}
}

func looksLikeURL(s string) bool {
	if strings.HasPrefix(s, "@") {
		return false
	}
	// This works because service names are not allowed to contain dot
	// This should therefore be a valid IPv4 or IPv6 address
	return strings.HasPrefix(s, "/") ||
		strings.ContainsAny(s, ".:") ||
		looksLikeURLPattern.MatchString(s) ||
		s == "localhost"
}

var (
	_ LocationResolver      = (*serviceResolver)(nil)
	_ httpclient.Middleware = (*pasticheLocation)(nil)
	_ Location              = (*pasticheLocation)(nil)
)
