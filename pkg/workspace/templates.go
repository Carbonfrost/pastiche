// Copyright 2023, 2025, 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package workspace

import (
	"os"
	"path/filepath"

	cli "github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/bind"
	"github.com/Carbonfrost/joe-cli/extensions/template"
	"github.com/Carbonfrost/pastiche/pkg/config"
	"sigs.k8s.io/yaml"
)

type InitServiceCommand struct {
	Title       string
	Name        string
	Description string

	cli.Action
}

func NewInitServiceCommand() *InitServiceCommand {
	req := &InitServiceCommand{}
	req.Action = cli.Pipeline(
		bind.SetPointer(&req.Title, bind.String("title")),
		bind.SetPointer(&req.Name, bind.Func[string](fallbackServiceName)),
		bind.SetPointer(&req.Description, bind.String("description")),
		bind.Action(applyInitTemplate, bind.Exact(req)),
	)
	return req
}

func (c *InitServiceCommand) toService() *config.Service {
	return &config.Service{
		Title:       c.Title,
		Name:        c.Name,
		Description: c.Description,
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

func fallbackServiceName(c *cli.Context) (string, error) {
	name := c.String("name")
	if name == "" {
		name = "service"
		wd, err := os.Getwd()
		if err == nil {
			return filepath.Base(wd), nil
		}
	}
	return name, nil
}

func applyInitTemplate(cmd *InitServiceCommand) cli.Action {
	return template.New(
		template.Dir(".pastiche",
			template.Vars{
				"ServiceName": cmd.Name,
			},
			template.File("{{ .ServiceName }}.yml", yamlContents(cmd.toService())),
		),
	)
}

func yamlContents(v any) template.FileGenerator {
	data, _ := yaml.Marshal(v)
	return template.Contents(data)
}
