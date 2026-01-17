// Copyright 2025, 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"slices"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli-http/httpclient"
	"github.com/Carbonfrost/joe-cli-http/uritemplates"
	"github.com/Carbonfrost/joe-cli/extensions/bind"
	"github.com/Carbonfrost/joe-cli/extensions/exec"
	"github.com/Carbonfrost/pastiche/pkg/config"
	"github.com/Carbonfrost/pastiche/pkg/grpcclient"
	"github.com/Carbonfrost/pastiche/pkg/model"
	"sigs.k8s.io/yaml"
)

// SetClientType provides an action which sets the client type
func SetClientType(v ...ClientType) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "client",
			Aliases:  []string{"g"},
			HelpText: "Specify the client that will be used",
		},
		withBinding((*Client).SetClientType, v),
	)
}

// Do provides the default action of the client which is to invoke it and print the results.
// This can be assigned to the Uses pipeline to set up necessary prerequisites. The actual action
// is [FetchAndPrint]
func Do() cli.Action {
	return invokeUsingMethod()
}

// Open reveals a particular file or link in the editor or web browser
func Open() cli.Action {
	return cli.Pipeline(
		cli.Prototype{
			Description: "View a given service configuration or link",
		},
		cli.AddArgs([]*cli.Arg{
			{
				Name:       "service",
				Value:      new(model.ServiceSpec),
				Completion: completeServices(),
				// TODO Dubious - but perhaps Description should be displayed
			},
		}...),
		cli.AddFlags([]*cli.Flag{
			{
				Name:     "rel",
				Aliases:  []string{"r"},
				HelpText: "Follow the given link by RELATIONSHIP",
				Value:    new(string),
			},
		}...),
		bind.Action2(openSpec, bind.Value[*model.ServiceSpec]("service"), bind.String("rel")),
	)
}

// Import provides the action for importing a definition
func Import() cli.Action {
	return cli.Pipeline(
		cli.Prototype{
			Description: "Import a service configuration from another format",
		},
		cli.AddArgs([]*cli.Arg{
			{
				Name:       "service",
				Value:      new(model.ServiceSpec),
				Completion: completeServices(),
				// TODO Dubious - but perhaps Description should be displayed
			},
		}...),
		cli.AddFlags([]*cli.Flag{
			{
				Name:     "name",
				HelpText: "Set the name of the resource",
			},
			{
				Name:     "title",
				HelpText: "Set the title of the resource",
			},
			{
				Name:     "description",
				HelpText: "Set the description of the resource",
			},
		}...),
		cli.At(cli.ActionTiming, cli.ActionFunc(importSpec)),
	)
}

// FetchAndPrint invokes the client and prints the results.
func FetchAndPrint() cli.Action {
	return cli.ActionOf(func(ctx context.Context) error {
		c := FromContext(ctx)

		clientType := c.Type()
		// TODO This should delegate to respective location methods rather than
		// have them resolve this again within. This also assumes only one location
		// is ever returned

		if clientType == ClientTypeUnspecified {
			locations, err := c.locationResolver.Resolve(ctx)
			if err != nil {
				return err
			}
			if p, ok := locations[0].(Location); ok {
				clientType = fromClientType(p.Resolved().Client())
			}
		}

		if clientType == ClientTypeGRPC {
			return cli.Do(ctx, cli.Pipeline(httpClientInterop, grpcclient.FetchAndPrint()))
		}

		return cli.Do(ctx, httpclient.FetchAndPrint())
	})
}

func fromClientType(c model.Client) ClientType {
	if _, ok := c.(*model.GRPCClient); ok {
		return ClientTypeGRPC
	}
	if _, ok := c.(*model.HTTPClient); ok {
		return ClientTypeHTTP
	}
	return ClientTypeUnspecified
}

func openSpec(ss *model.ServiceSpec, rel string) cli.Action {
	return cli.ActionFunc(func(c *cli.Context) error {
		cfg, _ := config.Load()

		mo := model.New(cfg)

		merged, err := mo.Resolve(*ss, "", "")
		if err != nil {
			return err
		}
		request, err := merged.EvalRequest(nil, nil)
		if err != nil {
			return err
		}
		if rel == "" {
			panic("not implemented: file-based resolution")
		}
		for _, l := range request.Links() {
			if l.Rel == rel {
				return exec.Open(l.HRef)
			}
		}
		return fmt.Errorf("unknown rel %s in %q", rel, ss.Path())
	})
}

func importSpec(c *cli.Context) error {
	in, err := io.ReadAll(c.Stdin)
	if err != nil {
		return err
	}

	call, err := model.ParseJSFetchCall(string(in))
	if err != nil {
		return err
	}

	endpoint := call.ToEndpoint()
	endpoint.Name = c.String("name")
	endpoint.Description = c.String("description")
	endpoint.Title = c.String("title")

	ss := *c.Value("service").(*model.ServiceSpec)
	mo := &model.Model{
		Services: []*model.Service{
			{
				Name:     ss[0],
				Resource: &model.Resource{},
			},
		},
	}

	current := mo.Services[0].Resource
	for _, s := range ss[1:] {
		newChild := &model.Resource{
			Name: s,
		}
		current.Resources = append(current.Resources, newChild)
		current = newChild
	}
	current.Endpoints = append(current.Endpoints, endpoint)

	// TODO Actually persist in the config location specified
	data, err := yaml.Marshal(model.ToConfig(mo))
	if err != nil {
		return err
	}
	fmt.Println(string(data))

	return nil
}

func invokeUsingMethod() cli.Action {
	return cli.Pipeline(
		cli.Prototype{
			Description: "Make a request using the given service",
		},
		cli.Setup{
			Uses: cli.Pipeline(
				cli.AddArgs([]*cli.Arg{
					{
						Name:       "service",
						Value:      new(model.ServiceSpec),
						Completion: completeServices(),
						Uses:       setDescription,
					},
					{
						Name:       "args",
						Value:      new(cli.NameValue),
						Options:    cli.EachOccurrence,
						Completion: completeServiceArgs(),
						NArg:       cli.TakeUntilNextFlag,
						Action: func(c *cli.Context) error {
							it := c.NameValue("")
							httpclient.FromContext(c).LocationResolver.AddVar(uritemplates.StringVar(it.Name, it.Value))
							return nil
						},
					},
				}...,
				),
				cli.AddFlags([]*cli.Flag{
					{
						Name:       "server",
						Aliases:    []string{"S"},
						HelpText:   "Use the specified server for the request",
						Value:      new(string),
						Completion: completeServer(),
					},
				}...),
			),
			Action: cli.Pipeline(
				FetchAndPrint(),
			),
		})
}

func completeServices() cli.CompletionFunc {
	return func(cc *cli.Context) []cli.CompletionItem {
		cfg, _ := config.Load()
		names := make([]string, 0, len(cfg.Services))
		for _, s := range cfg.Services {
			names = append(names, s.Name)
		}
		return cli.CompletionValues(names...).Complete(cc)
	}
}

func completeServiceArgs() cli.CompletionFunc {
	return func(cc *cli.Context) []cli.CompletionItem {
		_, resource, ok := tryContextResolve(cc)
		if !ok {
			return nil
		}

		names := resource.URITemplate.Names()
		return cli.CompletionValues(names...).Complete(cc)
	}
}

func completeServer() cli.CompletionFunc {
	return func(cc *cli.Context) []cli.CompletionItem {
		service, _, ok := tryContextResolve(cc)
		if !ok {
			return nil
		}

		names := make([]string, 0, len(service.Servers))
		for _, s := range service.Servers {
			names = append(names, s.Name)
		}
		return cli.CompletionValues(names...).Complete(cc)
	}
}

func tryContextResolve(c *cli.Context) (service *model.Service, res *model.Resource, ok bool) {
	method := c.String("method")
	server := c.String("server")
	v, found := c.Value("service").(*model.ServiceSpec)
	if !found {
		return
	}
	cfg, err := config.Load()
	if err != nil {
		return
	}

	mo := model.New(cfg)
	merged, err := mo.Resolve(*v, server, method)
	service = merged.Service()
	res = merged.Resource()

	if err != nil {
		return
	}
	ok = true
	return
}

func setDescription(c *cli.Context) error {
	servicesData := func() any {
		cfg, _ := config.Load()
		mo := model.New(cfg)

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

// TODO Allow grpcclient to have standalone support for headers
var httpClientInteropFlags = map[string]func(any) grpcclient.Option{
	"header": func(in any) grpcclient.Option {
		header := in.(*httpclient.HeaderValue)
		return grpcclient.WithHeader(header.Name, header.Value)
	},
}

// httpClientInterop scans the context for values that were set using
// the httpclient and copies them to compatible grpcclient options
func httpClientInterop(c *cli.Context) error {
	src, tag := httpclient.SourceAnnotation()
	client := grpcclient.FromContext(c)

	for flag, optFn := range httpClientInteropFlags {
		// Ensure that flag is defined and is from httpclient package
		f, ok := c.LookupFlag(flag)
		if !ok {
			continue
		}

		if v, ok := f.LookupData(src); ok && tag == v {

			// TODO Only supports last value in binding lookup
			actual := c.BindingLookup().Value(flag)
			opt := optFn(actual)
			opt(client)
		}
	}

	return nil
}
