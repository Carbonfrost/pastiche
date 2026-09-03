// Copyright 2023, 2025, 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package workspace

import (
	"context"
	"fmt"

	"github.com/Carbonfrost/joe-cli/extensions/bind"
	"github.com/Carbonfrost/joe-cli/extensions/marshal"
	"github.com/Carbonfrost/joe-cli/extensions/template"
	"github.com/Carbonfrost/pastiche/pkg/config"
	"sigs.k8s.io/yaml"
)

func (p *InitParams) toService() *config.Service {
	return &config.Service{
		Name:        p.Name,
		Title:       p.Title,
		Description: p.Description,
		Tags:        p.Tags,
		Comment:     p.Comment,
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
	// TODO joe-cli@futures makes it viable to use DisallowUnknownFields
	yamlOutput, err := marshal.YAML.New()
	if err != nil {
		panic(fmt.Errorf("unexpected codec not registered: %w", err))
	}
	return template.Dir(".pastiche",
		template.Vars{
			"ServiceName": p.Name,
		},
		template.File("{{ .ServiceName }}.yml", template.ContentsMarshal(p.toService(), yamlOutput)),
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
