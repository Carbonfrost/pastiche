package httpclient

import (
	"context"
	"fmt"
	"net/url"
	"os"

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

func (s *serviceResolver) Resolve(c context.Context) ([]*url.URL, error) {
	spec := s.root(c)
	service, resource, err := s.config.Resolve(*spec)
	if err != nil {
		return nil, err
	}

	// Detection of the method from the context is a hack that breaks the
	// encapsulation of the resolver and mutates the client, but is currently
	// necessary to apply the semantics of the chosen or default method until
	// an alternative is available from the joe-cli-http API
	method, _ := c.Value("method").(string)
	ep := findEndpointOrDefault(resource, method, *spec)

	if ep != nil {
		httpclient.FromContext(c).SetMethod(ep.Method)
	}

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
	base, err := s.findBaseURL(service, s.server(c))
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
	return res, nil
}

func (s *serviceResolver) findBaseURL(svc *model.Service, server string) (*url.URL, error) {
	if s.base != nil {
		return s.base, nil
	}
	if server != "" {
		if svr, ok := svc.Server(server); ok {
			return url.Parse(svr.BaseURL)
		}
		return nil, fmt.Errorf("no server %q defined for service %q", server, svc.Name)
	}
	return url.Parse(svc.Servers[0].BaseURL)
}

func findEndpointOrDefault(resource *model.Resource, method string, spec model.ServiceSpec) *model.Endpoint {
	if method != "" {
		ep, ok := resource.Endpoint(method)
		if !ok {
			logWarning(fmt.Sprintf("warning: method %s is not defined for resource %s\n", method, spec.Path()))
		}
		return ep
	}
	if len(resource.Endpoints) > 0 {
		return resource.Endpoints[0]
	}
	return nil
}

func logWarning(v interface{}) {
	fmt.Fprint(os.Stderr, v)
}

var _ httpclient.LocationResolver = (*serviceResolver)(nil)
