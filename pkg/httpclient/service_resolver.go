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
	"github.com/Carbonfrost/pastiche/pkg/internal/log"
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
	spec     model.ServiceSpec
	resource *model.Resource
	service  *model.Service
	server   *model.Server
	ep       *model.Endpoint
	u        *url.URL
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

	service, resource, err := s.config.Resolve(spec)
	if err != nil {
		return nil, err
	}

	// Detection of the method from the context is a hack that breaks the
	// encapsulation of the resolver, but is currently
	// necessary to apply the semantics of the chosen or default method until
	// an alternative is available from the joe-cli-http API
	method, _ := c.Value("method").(string)
	ep := findEndpointOrDefault(resource, method, spec)
	tt := resource.URITemplate
	expanded, err := tt.Expand(s.vars)

	if err != nil {
		return nil, err
	}
	loc, err := url.Parse(expanded)
	if err != nil {
		return nil, err
	}

	res := []*url.URL{loc}
	server, base, err := s.findBaseURL(service, s.server(c))
	if err != nil {
		return nil, err
	}

	for i := range res {
		if i > 0 {
			res[i] = res[i-1].ResolveReference(res[i])

		} else if base != nil {
			res[i] = base.ResolveReference(res[i])
		}
	}

	ll := make([]httpclient.Location, len(res))
	for i := range res {
		ll[i] = &pasticheLocation{
			spec:     spec,
			service:  service,
			resource: resource,
			server:   server,
			ep:       ep,
			u:        res[i],
		}
	}
	return ll, nil
}

func (s *serviceResolver) findBaseURL(svc *model.Service, server string) (*model.Server, *url.URL, error) {
	if s.base != nil {
		return nil, s.base, nil
	}
	if server != "" {
		svr, ok := svc.Server(server)
		if !ok {
			return nil, nil, fmt.Errorf("no server %q defined for service %q", server, svc.Name)
		}
		u, err := url.Parse(svr.BaseURL)
		return svr, u, err
	}

	if len(svc.Servers) == 0 {
		return nil, nil, fmt.Errorf("no servers defined for service %q", svc.Name)
	}
	u, err := url.Parse(svc.Servers[0].BaseURL)
	return svc.Servers[0], u, err
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

	if loc.ep == nil {
		return fmt.Errorf("no endpoint defined for %v", loc.spec.Path())
	}
	r.Method = loc.ep.Method
	httpclient.WithHeaders(loc.resource.Headers).Handle(r)
	httpclient.WithHeaders(loc.ep.Headers).Handle(r)
	if loc.server != nil {
		httpclient.WithHeaders(loc.server.Headers).Handle(r)
	}

	return nil
}

func findEndpointOrDefault(resource *model.Resource, method string, spec model.ServiceSpec) *model.Endpoint {
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

func looksLikeURL(s string) bool {
	// This works because service names are not allowed to contain dot
	// This should therefore be a valid IPv4 or IPv6 address
	return strings.HasPrefix(s, "/") ||
		strings.ContainsAny(s, ".:") ||
		looksLikeURLPattern.MatchString(s) ||
		s == "localhost"
}

var _ httpclient.LocationResolver = (*serviceResolver)(nil)
