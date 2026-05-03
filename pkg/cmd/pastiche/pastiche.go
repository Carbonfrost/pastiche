// Copyright 2023, 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pastiche

import (
	"os"

	cli "github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/color"
	"github.com/Carbonfrost/joe-cli/extensions/table"
	"github.com/Carbonfrost/pastiche/pkg/client"
	"github.com/Carbonfrost/pastiche/pkg/internal/build"
	"github.com/Carbonfrost/pastiche/pkg/server"
	"github.com/Carbonfrost/pastiche/pkg/workspace"
)

const (
	serviceTemplate = `Services:
{{ Table "Unformatted" -}}
    {{ range .Services }}
    {{- Row -}}
        {{- Cell "  " -}}
        {{- Cell (.Name | Bold) -}}
        {{- Cell .Title -}}
    {{- end -}}
{{- EndTable -}}`
)

func Run() {
	NewApp().Run(os.Args)
}

func NewApp() *cli.App {
	return &cli.App{
		Name:     "pastiche",
		HelpText: "Make requests to HTTP APIs using their OpenAPI schemas and definitions.",
		Comment:  "Smart OpenAPI client",
		Options:  cli.Sorted,
		Version:  build.Version,
		Uses: cli.Pipeline(
			&color.Options{},
			&table.Options{
				Features: table.AllFeatures &^ table.UseTablesInHelpTemplate,
			},
			cli.RegisterTemplate("PasticheServices", serviceTemplate),
			cli.ImplicitCommand("fetch"),

			workspace.New(),
		),
		Commands: []*cli.Command{
			{Name: "init", Uses: workspace.Init()},
			{Name: "env", Uses: workspace.Env()},
			{Name: "describe", Uses: client.Describe()},
			{Name: "serve", Uses: server.Serve()},
			{Name: "log", Uses: workspace.Log()},
			{Name: "fetch", Uses: client.Do(),
				Flags: []*cli.Flag{
					{Uses: client.SetVarFromEnvVar()},
				}},
			{Name: "import", Uses: client.Import()},
			{
				Name: "open",
				Uses: client.Open(),
			},
		},
	}
}
