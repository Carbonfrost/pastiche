// Copyright 2023 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package pastiche

import (
	"fmt"
	"os"

	cli "github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli-http/httpclient"
	"github.com/Carbonfrost/joe-cli/extensions/color"
	"github.com/Carbonfrost/joe-cli/extensions/table"
	phttpclient "github.com/Carbonfrost/pastiche/pkg/client"
	"github.com/Carbonfrost/pastiche/pkg/internal/build"
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
			phttpclient.New(),
			cli.ImplicitCommand("fetch"),
		),
		Before: cli.Pipeline(
			suppressHTTPClientHelpByDefault(),
		),
		Commands: []*cli.Command{
			{
				Name:     "init",
				HelpText: "Initialize the current directory with a new service definition",
				Uses:     InitCommand(),
			},
			{
				Name:     "describe",
				HelpText: "Describe resources within Pastiche workspace",
				Subcommands: []*cli.Command{
					{
						Name:    "service",
						Aliases: []string{"services", "svc"},
						Uses:    DescribeServiceCommand(),
						Before:  disallowPersistentHTTPFlags,
					},
				},
				Uses: cli.HandleCommandNotFound(nil),
			},
			{Name: "fetch", Uses: phttpclient.Do()},
		},
		Flags: []*cli.Flag{
			{
				Name:     "chdir",
				HelpText: "Change directory into the specified working {DIRECTORY}",
				Value:    new(cli.File),
				Options:  cli.WorkingDirectory | cli.NonPersistent,
			},
			{
				Name:        "all",
				HelpText:    "Facilitates displaying help text that is suppressed by default (used with --help)",
				Value:       cli.Bool(),
				Options:     cli.NonPersistent,
				Description: "'pastiche --help --all' displays information about HTTP client options and other advanced features",
			},
		},
	}
}

func suppressHTTPClientHelpByDefault() cli.ActionFunc {
	// There are quite a number of options to display for the HTTP client, so
	// hide these until they are requested explicitly
	return func(c *cli.Context) error {
		if c.Seen("all") {
			return nil
		}
		n, v := httpclient.SourceAnnotation()
		return c.Walk(func(cmd *cli.Context) error {
			for _, f := range cmd.Command().Flags {

				if f.Data[n] == v {
					f.SetHidden(true)
				}
			}
			return nil
		})
	}
}

// disallowPersistentHTTPFlags is an action which returns an error if one
// of the httpclient flags is present.  This is to allow them to be persistently
// defined but not actually usable within certain contexts
func disallowPersistentHTTPFlags(c *cli.Context) error {
	src, tag := httpclient.SourceAnnotation()
	for _, k := range c.BindingLookup().BindingNames() {
		f, ok := c.LookupFlag(k)
		if !ok {
			continue
		}

		if v, ok := f.LookupData(src); ok {
			if tag == v {
				return fmt.Errorf("unknown option %v", k)
			}
		}
	}

	return nil
}
