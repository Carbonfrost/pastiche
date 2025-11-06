// Copyright 2025 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package httpclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/Carbonfrost/joe-cli-http/httpclient"
	"github.com/Carbonfrost/joe-cli-http/httpclient/expr"
	"github.com/Carbonfrost/joe-cli-http/uritemplates"
	"github.com/Carbonfrost/pastiche/pkg/model"
)

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

	u *url.URL
}

type contextKey string

var looksLikeURLPattern = regexp.MustCompile(`^https?://`)

func NewServiceResolver(
	c *model.Model,
	root func(context.Context) *model.ServiceSpec,
	server func(context.Context) string,
	method func(context.Context) string,
) httpclient.LocationResolver {
	return &serviceResolver{
		root:   root,
		server: server,
		method: method,
		config: c,
		vars:   uritemplates.Vars{},
	}
}

func (s *serviceResolver) Add(location string) error {
	return fmt.Errorf("multiple locations not supported")
}

func (s *serviceResolver) AddVar(v *uritemplates.Var) error {
	s.vars.Add(v)
	return nil
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

func newLocation(base *url.URL, vars map[string]any, merged model.ResolvedResource) (*pasticheLocation, error) {
	loc, err := merged.URL(base, vars)
	if err != nil {
		return nil, err
	}
	var (
		serverHeaders, resourceHeaders, endpointHeaders httpclient.Middleware

		endpointMethod  httpclient.Middleware
		requireEndpoint httpclient.MiddlewareFunc = func(req *http.Request) error {
			if merged.Endpoint() == nil {
				return errors.New("no endpoint defined for service/spec")
			}
			return nil
		}
	)

	expander := expr.ComposeExpanders(
		expr.Prefix("env", func(k string) any {
			return os.Getenv(k)
		}),
		expr.ExpandMap(vars),
	)
	varLookup := func(k string) string {
		return fmt.Sprint(expander(k))
	}

	if merged.Server() != nil {
		serverHeaders = withHeaders(merged.Server().Headers, varLookup)
	}
	if merged.Resource() != nil {
		resourceHeaders = withHeaders(merged.Resource().Headers, varLookup)
	}
	if merged.Endpoint() != nil {
		endpointHeaders = withHeaders(merged.Endpoint().Headers, varLookup)
		endpointMethod = withMethod(merged.Endpoint().Method)
	}

	return &pasticheLocation{
		Middleware: httpclient.ComposeMiddleware(
			requireEndpoint,
			serverHeaders,
			resourceHeaders,
			endpointHeaders,
			endpointMethod,
			withBody(merged.Body(vars)),
		),
		u: loc,
	}, nil
}

func (l *pasticheLocation) URL(ctx context.Context) (context.Context, *url.URL, error) {
	return ctx, l.u, nil
}

func withHeaders(header http.Header, expander func(string) string) httpclient.Middleware {
	copy := http.Header{}
	for k, v := range header {
		result := make([]string, len(v))
		for i, str := range v {
			result[i] = os.Expand(str, expander)
		}
		copy[k] = result
	}

	return httpclient.WithHeaders(copy)
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

func looksLikeURL(s string) bool {
	// This works because service names are not allowed to contain dot
	// This should therefore be a valid IPv4 or IPv6 address
	return strings.HasPrefix(s, "/") ||
		strings.ContainsAny(s, ".:") ||
		looksLikeURLPattern.MatchString(s) ||
		s == "localhost"
}

var (
	_ httpclient.LocationResolver = (*serviceResolver)(nil)
	_ httpclient.Middleware       = (*pasticheLocation)(nil)
)
