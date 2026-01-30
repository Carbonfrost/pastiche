// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package model

import (
	"strings"

	"github.com/Carbonfrost/pastiche/pkg/config"
)

func ToConfig(m *Model) *config.File {
	var services []config.Service
	for _, s := range m.Services {
		services = append(services, configService(s))
	}
	return &config.File{
		Services: services,
	}
}

func configService(v *Service) config.Service {
	servers := make([]config.Server, len(v.Servers))
	for i, s := range v.Servers {
		servers[i] = configServer(s)
	}
	return config.Service{
		Name:        v.Name,
		Title:       v.Title,
		Description: v.Description,
		Servers:     servers,
		Resources:   singleton(configResource(v.Resource)),
		Links:       configLinks(v.Links),
		Vars:        v.Vars,
		Client:      configClient(v.Client),
	}
}

func configServer(s *Server) config.Server {
	return config.Server{
		Name:        s.Name,
		BaseURL:     s.BaseURL,
		Description: s.Description,
		Title:       s.Title,
		Headers:     s.Headers,
		Links:       configLinks(s.Links),
		Vars:        s.Vars,
	}
}

func configResource(r *Resource) *config.Resource {
	uri := ""
	if r.URITemplate != nil {
		uri = r.URITemplate.String()
	}
	res := &config.Resource{
		Name:        r.Name,
		Title:       r.Title,
		Description: r.Description,
		URI:         uri,
		Headers:     r.Headers,
		Links:       configLinks(r.Links),
		Body:        r.Body,
		RawBody:     r.RawBody,
		Vars:        r.Vars,
		Form:        r.Form,
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

		case "query":
			res.Query = ep

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
		Name:        r.Name,
		Title:       r.Title,
		Description: r.Description,
		Headers:     r.Headers,
		Links:       configLinks(r.Links),
		Body:        r.Body,
		RawBody:     r.RawBody,
		Vars:        r.Vars,
		Form:        r.Form,
	}
}

func configLinks(links []Link) []config.Link {
	res := make([]config.Link, len(links))
	for i, l := range links {
		res[i] = config.Link{
			HRef:     l.HRef,
			HRefLang: l.HRefLang,
			Audience: l.Audience,
			Rel:      l.Rel,
			Title:    l.Title,
		}
	}
	return res
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
