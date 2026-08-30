// Copyright 2023, 2025, 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package workspace

import (
	"context"

	"github.com/Carbonfrost/joe-cli/extensions/bind"
	"github.com/Carbonfrost/joe-cli/extensions/template"
	"github.com/Carbonfrost/pastiche/pkg/config"
	"sigs.k8s.io/yaml"
)

func (p *InitParams) toService() *config.Service {
	return &config.Service{
		Name:        p.Name,
		Title:       p.Title,
		Description: p.Description,
		Servers: []config.Server{
			{
				Name:    "default",
				BaseURL: "http://localhost:8000/",
			},
		},
		Resources: []config.Resource{
			{
				Name: "get",
				URI:  "/",
				Get:  &config.Endpoint{},
			},
		},
	}
}

func (p *InitParams) newGenerator() template.Generator {
	return template.Dir(".pastiche",
		template.Vars{
			"ServiceName": p.Name,
		},
		template.File("{{ .ServiceName }}.yml", yamlContents(p.toService())),
		template.File(".gitignore", template.ContentsString("/logs")),
	)
}

type generator struct {
	params bind.Binder[*InitParams]
}

func (g *generator) Generate(ctx context.Context, c *template.OutputContext) error {
	params, err := g.params.Bind(ctx)
	if err != nil {
		return err
	}
	return params.newGenerator().Generate(ctx, c)
}

func yamlContents(v any) template.FileGenerator {
	data, _ := yaml.Marshal(v)
	return template.Contents(data)
}

var _ template.Generator = (*generator)(nil)
