// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package workspace

import (
	"cmp"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"slices"

	cli "github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/bind"
	"github.com/Carbonfrost/joe-cli/extensions/config"
	"github.com/Carbonfrost/joe-cli/extensions/template"
	"github.com/Carbonfrost/pastiche/pkg/model"
	"github.com/Carbonfrost/pastiche/pkg/workspace/logs"
)

// Action provides a workspace action
type Action = cli.Action

type DescribeParams struct {
	Spec   *model.ServiceSpec
	Kind   model.ItemKind
	Tags   []string
	Method string
}

type InitParams struct {
	Name        string
	Title       string
	Description string
	Comment     string
	Tags        []string
}

func newParams[T any](action cli.Action, binder bind.Func[T]) bind.ActionBinder[T] {
	return bind.NewActionBinder(action, binder)
}

var httpMethods = []string{
	"GET", "HEAD", "POST", "PUT", "PATCH", "DELETE", "CONNECT", "OPTIONS", "TRACE",
}

// Init provides the action for initializing a new service definition
func Init(paramsopt ...*InitParams) Action {
	params := exactOrUse(useInitParams, paramsopt)

	root := template.New(&generator{params: params})

	// TODO A bug in joe-cli@v0.21.0 requires this custom action
	root.Action = cli.Pipeline(
		cli.AddFlags([]*cli.Flag{
			{Uses: root.DryRunFlag()},
			{Uses: root.OverwriteFlag()},
		}...),
		cli.At(cli.ActionTiming, cli.ActionOf(root.Generate)),
	)

	return cli.Pipeline(
		cli.Prototype{
			Name:     "init",
			HelpText: "Initialize the current directory with a new service definition",
		},
		root,
		params,
	)
}

func useInitParams() bind.ActionBinder[*InitParams] {
	return newParams(cli.Pipeline(
		cli.Setup{
			Uses: cli.AddFlags([]*cli.Flag{
				{
					Name:     "name",
					HelpText: "Name of the service",
				},
				{
					Name:     "title",
					HelpText: "Title of the service",
				},
				{
					Name:     "description",
					HelpText: "Short description of the service",
				},
				{
					Name:     "comment",
					HelpText: "Short comment for the service",
				},
				{
					Name:     "tags",
					Aliases:  []string{"T"},
					Value:    cli.List(),
					HelpText: "Short comment for the service",
				},
				{
					Name:     "set",
					Aliases:  []string{"S"},
					HelpText: "Set a {0:PROPERTY} to {1:VALUE} on the service",
					Options:  cli.EachOccurrence,
					Uses: bind.Action2(
						func(name, value string) Action {
							return bind.Redirect(name, value)
						},
						bind.NameValue().Name(),
						bind.NameValue().Value(),
					),
				},
			}...),
		},
	),
		func(c *cli.Context) (*InitParams, error) {
			name, err := fallbackServiceName(c)
			return &InitParams{
				Name:        name,
				Title:       c.String("title"),
				Description: c.String("description"),
				Comment:     c.String("comment"),
				Tags:        c.List("tags"),
			}, err
		},
	)
}

// fallbackServiceName obtains the name of the service, which is the name of the
// current directory when the flag was not specified
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

func Env() Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "env",
			HelpText: "Display information about the Pastiche environment",
			Options:  cli.Exits,
			Value:    new(bool),
			Uses: cli.Pipeline(
				cli.AddFlags([]*cli.Flag{
					{
						Name:     "json",
						HelpText: "Print the env vars in json format",
						Value:    new(bool),
					},
				}...),
				cli.AddArgs([]*cli.Arg{
					{
						Name:  "vars",
						Value: cli.List(),
					},
				}...),
			),
			Action: cli.IfMatch(
				cli.ContextFilterFunc(seenOutputFlags),
				bind.Call3(dumpEnv, bind.FromContext(FromContext), bind.Stdout(), bind.List("vars")),
				config.PrintEnv(),
			),
		},
	)
}

func seenOutputFlags(c *cli.Context) bool {
	return c.Seen("json")
}

func dumpEnv(w *Workspace, out io.Writer, vars []string) error {
	env := w.environ()

	if len(vars) > 0 {
		env = filterMap(env, vars)
	}
	return json.NewEncoder(out).Encode(env)
}

func filterMap(in map[string]string, vars []string) map[string]string {
	result := map[string]string{}
	for _, v := range vars {
		result[v] = in[v]
	}
	return result
}

// Log provides the action to access logs
func Log() Action {
	return cli.Pipeline(
		cli.Prototype{
			Name:     "log",
			Aliases:  []string{"logs"},
			HelpText: "Access request logs for the workspace",
			Uses: cli.AddFlags([]*cli.Flag{
				{Uses: logDelegate(logs.Clear)},
			}...),
			Action: cli.DisplayHelpScreen(),
		},
	)
}

func logDelegate(fn func(*logs.Log) Action) cli.ActionFunc {
	return func(c *cli.Context) error {
		return cli.Do(c, fn(FromContext(c).Log()))
	}
}

// SetDisableValidation disables validattion of configuration in the workspace
func SetDisableValidation() Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "disable-validation",
			HelpText: "Disable validation of configuration files",
			Value:    new(bool),
		},
		cli.At(cli.ActionTiming, DisableValidation()),
	)
}

// Describe provides the action for describing a resource
func Describe(paramsopt ...*DescribeParams) cli.Action {
	params := exactOrUse(useDescribeParams, paramsopt)
	return cli.Pipeline(
		cli.Prototype{
			Name:     "describe",
			HelpText: "Describe resources within Pastiche workspace",
		},
		cli.HandleCommandNotFound(nil),
		bind.Call2(describeSpec, bind.Context(), params),
	)
}

func describeSpec(c *cli.Context, params *DescribeParams) error {
	return FromContext(c).Describe(
		params.SearchCriteria(),
	)
}

// SearchCriteria obtains the criteria which the describe parameters select
func (p *DescribeParams) SearchCriteria() *model.SearchCriteria {
	return &model.SearchCriteria{
		Spec:        p.Spec,
		Kind:        p.Kind,
		IncludeTags: p.Tags,
		Method:      p.Method,
	}
}

func useDescribeParams() bind.ActionBinder[*DescribeParams] {
	return newParams(cli.Pipeline(
		cli.Setup{
			Uses: cli.Pipeline(
				cli.AddArg(&cli.Arg{
					Name:       "spec",
					Value:      new(model.ServiceSpec),
					Completion: completeServices(),
					Uses:       setDescription,
				}),
				cli.AddFlags([]*cli.Flag{
					{
						Name:     "tags",
						HelpText: "Filter items by tags",
						Aliases:  []string{"t"},
						Value:    new([]string),
					},
					{
						Name:       "method",
						Aliases:    []string{"X"},
						UsageText:  "NAME",
						HelpText:   "Search for endpoints which use the request method {NAME}",
						Value:      new(string),
						Completion: cli.ValueCompletion(httpMethods...),
					},
					{
						Name:     "endpoint",
						HelpText: "Search for endpoints",
						Value:    new(bool),
						Uses:     cli.Mutex("varset", "mixin", "flow", "all"),
					},
					{
						Name:     "varset",
						Aliases:  []string{"V"},
						HelpText: "Search for variable sets",
						Value:    new(bool),
						Uses:     cli.Mutex("endpoint", "mixin", "flow", "all"),
					},
					{
						Name:     "mixin",
						Aliases:  []string{"M"},
						HelpText: "Search for mixins",
						Value:    new(bool),
						Uses:     cli.Mutex("endpoint", "varset", "flow", "all"),
					},
					{
						Name:     "flow",
						Aliases:  []string{"F"},
						HelpText: "Search for flows",
						Value:    new(bool),
						Uses:     cli.Mutex("endpoint", "varset", "mixin", "all"),
					},
					{
						Name:     "all",
						Aliases:  []string{"A"},
						HelpText: "Search for all items",
						Value:    new(bool),
						Uses:     cli.Mutex("endpoint", "varset", "mixin", "flow"),
					},
				}...),
			),
		},
	),
		func(c *cli.Context) (*DescribeParams, error) {
			spec := c.Value("spec").(*model.ServiceSpec)
			return &DescribeParams{
				Spec:   spec,
				Kind:   describeItemKind(c, spec),
				Tags:   c.List("tags"),
				Method: c.String("method"),
			}, nil
		},
	)
}

func describeItemKind(c *cli.Context, spec *model.ServiceSpec) model.ItemKind {
	switch {
	case c.Bool("all"):
		return model.ItemKindAll
	case c.Bool("varset"):
		return model.ItemKindVarSet
	case c.Bool("mixin"):
		return model.ItemKindMixin
	case c.Bool("flow"):
		return model.ItemKindFlow
	case c.Bool("endpoint"), c.Seen("method"):
		return model.ItemKindEndpoint
	case spec != nil && len(*spec) > 1:
		return model.ItemKindResource
	}
	return model.ItemKindService
}

func completeServices() cli.CompletionFunc {
	return func(cc *cli.Context) []cli.CompletionItem {
		mo := FromContext(cc).Model()
		names := make([]string, 0, len(mo.Services))
		for _, s := range mo.Services {
			names = append(names, s.Name)
		}
		return cli.ValueCompletion(names...).Complete(cc)
	}
}

func setDescription(c *cli.Context) error {
	servicesData := func() any {
		mo := FromContext(c).Model()

		items := slices.Clone(mo.Services)
		slices.SortFunc(items, func(x, y *model.Service) int {
			return cmp.Compare(x.Name, y.Name)
		})

		return struct {
			Services []*model.Service
		}{
			Services: items,
		}
	}

	return c.SetDescription(
		c.Template("PasticheServices").BindFunc(servicesData),
	)
}

func exactOrUse[T any](useFactory func() bind.ActionBinder[T], paramsopt []T) bind.Binder[T] {
	switch len(paramsopt) {
	case 1:
		return bind.Exact(paramsopt[0])
	case 0:
		return useFactory()
	}
	panic("expected 0 or 1 arg")
}
