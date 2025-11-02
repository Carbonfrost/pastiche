// Copyright 2025 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package pastiche

import (
	"bytes"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli-http/httpclient"
	"github.com/Carbonfrost/joe-cli-http/uritemplates"
	"github.com/Carbonfrost/pastiche/pkg/config"
	"github.com/Carbonfrost/pastiche/pkg/model"
)

func invokeUsingMethod() cli.Action {
	return cli.Pipeline(
		cli.Setup{
			Uses: cli.Pipeline(
				cli.Category("Invoke HTTP client"),
				cli.AddArgs([]*cli.Arg{
					{
						Name:       "service",
						Value:      new(model.ServiceSpec),
						Completion: completeServices(),
						Uses: func(c *cli.Context) {
							c.Arg().Description = renderServices(c)
						},
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
				httpclient.FetchAndPrint(),
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

func renderServices(c *cli.Context) string {
	cfg, _ := config.Load()
	mo := model.New(cfg)
	items := []*model.Service{}
	for _, v := range mo.Services {
		items = append(items, v)
	}

	data := struct {
		Services []*model.Service
	}{
		Services: items,
	}

	var buf bytes.Buffer
	c.Template("PasticheServices").Execute(&buf, data)
	return buf.String()
}
