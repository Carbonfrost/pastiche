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
	"github.com/Carbonfrost/joe-cli-http/uritemplates"
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
}

type serviceResolver struct {
	root   func(context.Context) *model.ServiceSpec
	server func(context.Context) string
	method func(context.Context) string
	vars   uritemplates.Vars
	base   *url.URL
	config *model.Model
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
	c *model.Model,
	root func(context.Context) *model.ServiceSpec,
	server func(context.Context) string,
	method func(context.Context) string,
) LocationResolver {
	return &serviceResolver{
		root:   root,
		server: server,
		method: method,
		config: c,
		vars:   map[string]any{},
	}
}

func (s *serviceResolver) Add(location string) error {
	return fmt.Errorf("multiple locations not supported")
}

func (s *serviceResolver) AddVar(v *uritemplates.Var) error {
	s.vars.Add(v)
	return nil
}

func (s *serviceResolver) Vars() map[string]any {
	return s.vars
}

func (s *serviceResolver) BaseURL() *url.URL {
	return s.base
}

func (s *serviceResolver) SetBase(base *url.URL) error {
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

	merged, err := s.config.Resolve(spec, s.server(c), s.method(c))
	if err != nil {
		return nil, err
	}

	location, err := newLocation(s.base, s.vars, merged)
	if err != nil {
		return nil, err
	}

	return []httpclient.Location{
		location,
	}, nil
}

func (s *serviceResolver) resolveRequest(c context.Context) (model.Request, error) {
	spec := *s.root(c)
	merged, err := s.config.Resolve(spec, s.server(c), s.method(c))
	if err != nil {
		return nil, err
	}

	return merged.EvalRequest(s.base, s.vars)
}

func newLocation(base *url.URL, vars map[string]any, resolved model.ResolvedResource) (*pasticheLocation, error) {
	merged, err := resolved.EvalRequest(base, vars)
	if err != nil {
		return nil, err
	}

	loc, err := merged.URL()
	if err != nil {
		return nil, err
	}
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
			httpclient.WithHeaders(merged.Headers()),
			endpointMethod,
			withBody(merged.Body()),
			withAuth(merged.Auth()),
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
