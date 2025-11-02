// Copyright 2025 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package httpclient

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/Carbonfrost/joe-cli-http/httpclient"
	"github.com/Carbonfrost/joe-cli-http/uritemplates"
	"github.com/Carbonfrost/pastiche/pkg/model"
)

type serviceResolver struct {
	root   func(context.Context) *model.ServiceSpec
	server func(context.Context) string
	vars   uritemplates.Vars
	base   *url.URL
	config *model.Model
}

type pasticheLocation struct {
	spec   model.ServiceSpec
	merged model.ResolvedResource
	u      *url.URL
}

type pasticheMiddleware struct {
}

type contextKey string

const locationKey contextKey = "pastiche.location"

var looksLikeURLPattern = regexp.MustCompile(`^https?://`)

func NewServiceResolver(
	c *model.Model,
	root func(context.Context) *model.ServiceSpec,
	server func(context.Context) string,
) httpclient.LocationResolver {
	return &serviceResolver{
		root:   root,
		server: server,
		config: c,
		vars:   uritemplates.Vars{},
	}
}

func NewServiceResolverMiddleware() httpclient.Middleware {
	return &pasticheMiddleware{}
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

	// Detection of the method from the context is a hack that breaks the
	// encapsulation of the resolver, but is currently
	// necessary to apply the semantics of the chosen or default method until
	// an alternative is available from the joe-cli-http API
	method, _ := c.Value("method").(string)

	merged, err := s.config.Resolve(spec, s.server(c), method)
	if err != nil {
		return nil, err
	}

	loc, err := merged.URL(s.base, s.vars)
	if err != nil {
		return nil, err
	}

	return []httpclient.Location{
		&pasticheLocation{
			spec:   spec,
			merged: merged,
			u:      loc,
		},
	}, nil
}

func (l *pasticheLocation) URL(ctx context.Context) (context.Context, *url.URL, error) {
	return context.WithValue(ctx, locationKey, l), l.u, nil
}

func (l *pasticheMiddleware) Handle(r *http.Request) error {
	loc, ok := r.Context().Value(locationKey).(*pasticheLocation)
	if !ok {
		// Skip this request because there was no Pastiche location requested
		return nil
	}

	ep := loc.merged.Endpoint()
	resource := loc.merged.Resource()
	server := loc.merged.Server()

	if ep == nil {
		return fmt.Errorf("no endpoint defined for %v", loc.spec.Path())
	}
	r.Method = ep.Method
	httpclient.WithHeaders(resource.Headers).Handle(r)
	httpclient.WithHeaders(ep.Headers).Handle(r)
	if loc.merged.Server() != nil {
		httpclient.WithHeaders(server.Headers).Handle(r)
	}

	return nil
}

func looksLikeURL(s string) bool {
	// This works because service names are not allowed to contain dot
	// This should therefore be a valid IPv4 or IPv6 address
	return strings.HasPrefix(s, "/") ||
		strings.ContainsAny(s, ".:") ||
		looksLikeURLPattern.MatchString(s) ||
		s == "localhost"
}

var _ httpclient.LocationResolver = (*serviceResolver)(nil)
