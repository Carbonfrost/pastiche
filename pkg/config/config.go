package config

import (
	"fmt"

	"github.com/Carbonfrost/pastiche/pkg/model"
)

type ServiceConfig struct {
	Services map[string]*model.Service
}

func Load() (*ServiceConfig, error) {
	return &ServiceConfig{
		Services: map[string]*model.Service{
			"httpbin": ExampleHTTPBinorg,
		},
	}, nil
}

func (c *ServiceConfig) Resolve(s model.ServiceSpec) (*model.Service, *model.Resource, error) {
	if len(s) == 0 {
		return nil, nil, fmt.Errorf("no service specified")
	}
	svc, ok := c.Services[s[0]]
	if !ok {
		return nil, nil, fmt.Errorf("service not found: %q", s[0])
	}

	current := svc.Resource
	for i, p := range s[1:] {
		current, ok = current.Resource(p)
		if !ok {
			path := model.ServiceSpec(s[0 : i+2]).Path()
			return svc, current, fmt.Errorf("resource not found: %q", path)
		}
	}
	return svc, current, nil
}
