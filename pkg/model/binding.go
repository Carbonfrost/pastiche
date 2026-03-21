// Copyright 2025, 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package model

import (
	"github.com/Carbonfrost/joe-cli-http/uritemplates"
	"github.com/Carbonfrost/pastiche/pkg/config"
)

func service(v config.Service) *Service {
	servers := make([]*Server, len(v.Servers))
	for i, s := range v.Servers {
		servers[i] = server(s)
	}
	return &Service{
		Name:        v.Name,
		Comment:     v.Comment,
		Title:       v.Title,
		Description: v.Description,
		Tags:        v.Tags,
		Servers:     servers,
		Resource: &Resource{
			Name:        "/",
			URITemplate: mustParseURITemplate(""),
			Resources:   resources(v.Resources),
			Endpoints: []*Endpoint{
				{
					Method: "GET",
				},
			},
		},
		Links:  links(v.Links),
		Vars:   v.Vars,
		Client: client(v.Client),
		Auth:   auth(v.Auth),
		Output: outputs(v.Output),
	}
}

func mustParseURITemplate(t string) *uritemplates.URITemplate {
	u, err := uritemplates.Parse(t)
	if err != nil {
		panic(err)
	}
	return u
}

func server(s config.Server) *Server {
	return &Server{
		Name:        s.Name,
		Comment:     s.Comment,
		BaseURL:     s.BaseURL,
		Description: s.Description,
		Tags:        s.Tags,
		Title:       s.Title,
		Headers:     s.Headers,
		Links:       links(s.Links),
		Vars:        s.Vars,
		Auth:        auth(s.Auth),
		Output:      outputs(s.Output),
	}
}

func resource(r config.Resource) *Resource {
	uri, _ := uritemplates.Parse(r.URI)
	res := &Resource{
		Name:        r.Name,
		Comment:     r.Comment,
		Title:       r.Title,
		Description: r.Description,
		Tags:        r.Tags,
		URITemplate: uri,
		Headers:     r.Headers,
		Links:       links(r.Links),
		Body:        r.Body,
		RawBody:     r.RawBody,
		Vars:        r.Vars,
		Form:        r.Form,
		Auth:        auth(r.Auth),
		Output:      outputs(r.Output),
	}
	if r.Get != nil {
		res.Endpoints = append(res.Endpoints, endpoint("GET", r.Get))
	}
	if r.Put != nil {
		res.Endpoints = append(res.Endpoints, endpoint("PUT", r.Put))
	}
	if r.Post != nil {
		res.Endpoints = append(res.Endpoints, endpoint("POST", r.Post))
	}
	if r.Delete != nil {
		res.Endpoints = append(res.Endpoints, endpoint("DELETE", r.Delete))
	}
	if r.Options != nil {
		res.Endpoints = append(res.Endpoints, endpoint("OPTIONS", r.Options))
	}
	if r.Head != nil {
		res.Endpoints = append(res.Endpoints, endpoint("HEAD", r.Head))
	}
	if r.Trace != nil {
		res.Endpoints = append(res.Endpoints, endpoint("TRACE", r.Trace))
	}
	if r.Patch != nil {
		res.Endpoints = append(res.Endpoints, endpoint("PATCH", r.Patch))
	}
	if r.Query != nil {
		res.Endpoints = append(res.Endpoints, endpoint("QUERY", r.Query))
	}

	// Implicitly create GET endpoint if none other was created
	if len(res.Endpoints) == 0 {
		res.Endpoints = append(res.Endpoints, &Endpoint{Method: "GET"})
	}
	res.Resources = resources(r.Resources)
	return res
}

func resources(resources []config.Resource) []*Resource {
	res := make([]*Resource, len(resources))
	for i, child := range resources {
		res[i] = resource(child)
	}
	return res
}

func endpoint(method string, r *config.Endpoint) *Endpoint {
	return &Endpoint{
		Name:        r.Name,
		Comment:     r.Comment,
		Title:       r.Title,
		Description: r.Description,
		Tags:        r.Tags,
		Method:      method,
		Headers:     r.Headers,
		Links:       links(r.Links),
		Body:        r.Body,
		RawBody:     r.RawBody,
		Vars:        r.Vars,
		Form:        r.Form,
		Auth:        auth(r.Auth),
		Output:      outputs(r.Output),
	}
}

func links(links []config.Link) []Link {
	res := make([]Link, len(links))
	for i, l := range links {
		res[i] = Link{
			HRef:       l.HRef,
			HRefLang:   l.HRefLang,
			Audience:   l.Audience,
			Rel:        l.Rel,
			Title:      l.Title,
			Type:       l.Type,
			IsTemplate: l.IsTemplate,
		}
	}
	return res
}

func client(c *config.Client) Client {
	if c == nil {
		return nil
	}
	if c.GRPC != nil {
		return &GRPCClient{
			DisableReflection: c.GRPC.DisableReflection,
			ProtoSet:          c.GRPC.ProtoSet,
			Plaintext:         c.GRPC.Plaintext,
		}
	}
	if c.HTTP != nil {
		return &HTTPClient{}
	}

	return &HTTPClient{}
}

func auth(a *config.Auth) Auth {
	if a == nil {
		return nil
	}
	if a.Basic != nil {
		return &BasicAuth{
			User:     a.Basic.User,
			Password: a.Basic.Password,
		}
	}

	return nil
}

func outputs(outs []config.Output) []*OutputConfig {
	res := make([]*OutputConfig, len(outs))
	for i, o := range outs {
		res[i] = output(o)
	}
	return res
}

func output(o config.Output) *OutputConfig {
	return &OutputConfig{
		Name:        o.Name,
		Comment:     o.Comment,
		Title:       o.Title,
		Description: o.Description,
		Links:       links(o.Links),
		Filter:      outputFilter(o),
	}
}

func outputFilter(o config.Output) OutputFilter {
	if o.Template != nil {
		return &TemplateOutput{
			Text: o.Template.Text,
			File: o.Template.File,
		}
	}
	if o.JMESPath != nil {
		return &JMESPathOutput{
			Query: o.JMESPath.Query,
		}
	}
	if o.XPath != nil {
		return &XPathOutput{
			Query: o.XPath.Query,
		}
	}
	if o.Dig != nil {
		return &DigOutput{
			Query: o.Dig.Query,
		}
	}
	if o.JSON != nil {
		return &JSONOutput{
			Pretty: o.JSON.Pretty,
		}
	}
	if o.XML != nil {
		return &XMLOutput{
			Pretty: o.XML.Pretty,
		}
	}
	if o.YAML != nil {
		return &YAMLOutput{}
	}

	return nil
}
