// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package model

import (
	"fmt"
	"strings"

	"github.com/Carbonfrost/pastiche/pkg/config"
)

func ToConfigFile(m *Model) *config.File {
	var services []config.Service
	for _, s := range m.Services {
		services = append(services, configService(s))
	}
	var varSets []config.VarSet
	for _, v := range m.VarSets {
		varSets = append(varSets, configVarSet(v))
	}
	var flows []config.Flow
	for _, f := range m.Flows {
		flows = append(flows, configFlow(f))
	}
	return &config.File{
		Schema:   config.SchemaFile,
		Services: services,
		VarSets:  varSets,
		Flows:    flows,
	}
}

// ToConfig converts a model value into the corresponding configuration value.
func ToConfig(v Item) any {
	switch value := v.(type) {
	case *Model:
		return ToConfigFile(value)
	case *Service:
		return configService(value)
	case *Server:
		return configServer(value)
	case *Resource:
		return configResource(value)
	case *Endpoint:
		return configEndpoint(value)
	case *VarSet:
		return configVarSet(value)
	case *Flow:
		return configFlow(value)
	case *Step:
		return configStep(value)
	case Link:
		return configLink(value)
	case Client:
		return configClient(value)
	}
	panic(fmt.Errorf("unexpected type %T", v))
}

// Item is a value within the model.
type Item interface {
	itemSigil()
}

func (*Service) itemSigil()  {}
func (*Server) itemSigil()   {}
func (*Resource) itemSigil() {}
func (*Endpoint) itemSigil() {}
func (*VarSet) itemSigil()   {}
func (*Flow) itemSigil()     {}
func (*Step) itemSigil()     {}
func (Link) itemSigil()      {}
func (*Model) itemSigil()    {}

func configService(v *Service) config.Service {
	servers := make([]config.Server, len(v.Servers))
	for i, s := range v.Servers {
		servers[i] = configServer(s)
	}
	return config.Service{
		Schema:      config.SchemaService,
		Name:        v.Name,
		Title:       v.Title,
		Description: v.Description,
		Links:       configLinks(v.Links),
		Servers:     servers,
		Resources:   singleton(configResource(v.Resource)),
		Vars:        v.Vars,
		Client:      configClient(v.Client),
	}
}

func configServer(s *Server) config.Server {
	return config.Server{
		Schema:      config.SchemaServer,
		Name:        s.Name,
		Title:       s.Title,
		Description: s.Description,
		Links:       configLinks(s.Links),
		BaseURL:     s.BaseURL,
		Headers:     headerFromValues(s.Headers),
		Query:       headerFromValues(s.Query),
		Vars:        s.Vars,
	}
}

func configResource(r *Resource) *config.Resource {
	uri := ""
	if r.URITemplate != nil {
		uri = r.URITemplate.String()
	}
	res := &config.Resource{
		Schema:      config.SchemaResource,
		Name:        r.Name,
		Title:       r.Title,
		Description: r.Description,
		Links:       configLinks(r.Links),
		URI:         uri,
		Headers:     headerFromValues(r.Headers),
		Query:       headerFromValues(r.Query),
		Body:        r.Body,
		RawBody:     r.RawBody,
		Vars:        r.Vars,
		Form:        headerFromValues(r.Form),
	}

	for _, e := range r.Endpoints {
		ep := configEndpoint(e)

		switch strings.ToLower(e.Method) {
		case "get":
			res.Get = ep

		case "put":
			res.Put = ep

		case "post":
			res.Post = ep

		case "delete":
			res.Delete = ep

		case "options":
			res.Options = ep

		case "head":
			res.Head = ep

		case "trace":
			res.Trace = ep

		case "patch":
			res.Patch = ep

		default:
			panic("not implemented: custom endpoints")
		}
	}

	res.Resources = configResources(r.Resources)
	return res
}

func configResources(resources []*Resource) []config.Resource {
	res := make([]config.Resource, len(resources))
	for i, child := range resources {
		res[i] = *configResource(child)
	}
	return res
}

func configEndpoint(r *Endpoint) *config.Endpoint {
	return &config.Endpoint{
		Schema:      config.SchemaEndpoint,
		Name:        r.Name,
		Title:       r.Title,
		Description: r.Description,
		Links:       configLinks(r.Links),
		Headers:     headerFromValues(r.Headers),
		Query:       headerFromValues(r.Query),
		Body:        r.Body,
		RawBody:     r.RawBody,
		Vars:        r.Vars,
		Form:        headerFromValues(r.Form),
	}
}

func configVarSet(v *VarSet) config.VarSet {
	return config.VarSet{
		Schema:      config.SchemaVarSet,
		Name:        v.Name,
		Title:       v.Title,
		Description: v.Description,
		Links:       configLinks(v.Links),
		Vars:        v.Vars,
	}
}

func configFlow(f *Flow) config.Flow {
	return config.Flow{
		Schema:      config.SchemaFlow,
		Name:        f.Name,
		Title:       f.Title,
		Description: f.Description,
		Links:       configLinks(f.Links),
		Steps:       configSteps(f.Steps),
		Vars:        f.Vars,
	}
}

func configSteps(steps []*Step) []config.Step {
	res := make([]config.Step, len(steps))
	for i, s := range steps {
		res[i] = configStep(s)
	}
	return res
}

func configStep(s *Step) config.Step {
	step := config.Step{
		Name:        s.Name,
		Title:       s.Title,
		Description: s.Description,
		Links:       configLinks(s.Links),
		Method:      s.Method,
		Headers:     headerFromValues(s.Headers),
		Form:        headerFromValues(s.Form),
		Body:        s.Body,
		RawBody:     s.RawBody,
		Vars:        s.Vars,
	}

	switch t := s.StepType.(type) {
	case *SpecStep:
		step.Spec = t.Spec
	case *URLStep:
		step.URL = t.URL
	}
	return step
}

func headerFromValues(v Values) config.Values {
	if v == nil {
		return nil
	}
	h := make(config.Values, len(v))
	for i, e := range v {
		h[i] = config.Value{
			Values:   e.Values,
			Name:     e.Name,
			Value:    e.Value,
			Optional: e.Optional,
			Merge:    config.MergeMode(e.Merge), // Allowed because these underlying values are the same
		}
	}
	return h
}

func configValues(v Values) config.Values {
	if v == nil {
		return nil
	}
	res := make(config.Values, len(v))
	for i, e := range v {
		res[i] = config.Value{
			Name:     e.Name,
			Value:    e.Value,
			Values:   e.Values,
			Optional: e.Optional,
			Merge:    config.MergeMode(e.Merge),
		}
	}
	return res
}

func configLinks(links []Link) []config.Link {
	res := make([]config.Link, len(links))
	for i, l := range links {
		res[i] = configLink(l)
	}
	return res
}

func configLink(l Link) config.Link {
	return config.Link{
		HRef:       l.HRef,
		HRefLang:   l.HRefLang,
		Audience:   l.Audience,
		Rel:        l.Rel,
		Title:      l.Title,
		IsTemplate: l.IsTemplate,
	}
}

func configClient(c Client) *config.Client {
	if c == nil {
		return nil
	}
	switch client := c.(type) {
	case *HTTPClient:
		return &config.Client{
			HTTP: new(config.HTTPClient),
		}
	case *GRPCClient:
		return &config.Client{
			GRPC: &config.GRPCClient{
				DisableReflection: client.DisableReflection,
				ProtoSet:          client.ProtoSet,
				Plaintext:         client.Plaintext,
			},
		}
	}
	return nil
}

func singleton[T any](t *T) []T {
	if t == nil {
		return nil
	}
	return []T{*t}
}
