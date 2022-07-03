package httpclient

import (
	"context"
	"fmt"
	"net/url"

	"github.com/Carbonfrost/joe-cli-http/httpclient"
	"github.com/Carbonfrost/joe-cli-http/uritemplates"
	"github.com/Carbonfrost/pastiche/pkg/config"
	"github.com/Carbonfrost/pastiche/pkg/model"
)

type serviceResolver struct {
	root   func(context.Context) *model.ServiceSpec
	vars   uritemplates.Vars
	base   *url.URL
	config *config.ServiceConfig
}

func NewServiceResolver(c *config.ServiceConfig, root func(context.Context) *model.ServiceSpec) httpclient.LocationResolver {
	return &serviceResolver{
		root:   root,
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
	svc, resource, err := s.config.Resolve(*s.root(c))
	if err != nil {
		return nil, err
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
	base := s.base
	if base == nil {
		base, _ = url.Parse(svc.Servers[0].BaseURL)
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

var _ httpclient.LocationResolver = (*serviceResolver)(nil)
