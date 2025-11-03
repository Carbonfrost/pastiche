// Copyright 2025 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package httpclient

import (
	"context"
	"fmt"

	cli "github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli-http/httpclient"
	"github.com/Carbonfrost/joe-cli/extensions/provider"
	"github.com/Carbonfrost/pastiche/pkg/config"
	"github.com/Carbonfrost/pastiche/pkg/internal/build"
	"github.com/Carbonfrost/pastiche/pkg/model"
)

type Client struct {
	*httpclient.Client

	filter Filter
}

const (
	pasticheURL = "https://github.com/Carbonfrost/pastiche"

	servicesKey contextKey = "pastiche.client"
)

func New() *Client {
	cfg, _ := config.Load()
	client := httpclient.New(
		httpclient.WithDefaultUserAgent(defaultUserAgent()),
		httpclient.WithLocationResolver(
			NewServiceResolver(
				model.New(cfg),
				lateBinding[*model.ServiceSpec]("service"),
				lateBinding[string]("server"),
			),
		),
	)
	res := &Client{
		Client: client,
	}
	client.UseDownloadMiddleware(res.filterResponse)
	return res
}

func (c *Client) filterResponse(d httpclient.Downloader) httpclient.Downloader {
	if c.filter == nil {
		return d
	}
	return NewFilterDownloader(c.filter, d)
}

// FromContext obtains the client stored in the context
func FromContext(c context.Context) *Client {
	return c.Value(servicesKey).(*Client)
}

func (c *Client) Execute(ctx context.Context) error {
	return cli.Do(
		ctx,
		cli.Pipeline(
			c.Client,
			cli.RemoveArg(0), // Remove URL contributed by http client
			FilterRegistry,
			FlagsAndArgs(),
			ContextValue(c),
		),
	)
}

func ContextValue(c *Client) cli.Action {
	return cli.ContextValue(servicesKey, c)
}

func FlagsAndArgs() cli.Action {
	return cli.Pipeline(
		cli.AddFlags([]*cli.Flag{
			{Uses: ListFilters()},
			{Uses: SetFilter()},
		}...),
	)
}

func (c *Client) setFilterHelper(v *provider.Value) error {
	args := v.Args.(*map[string]string)
	f, err := FilterRegistry.New(v.Name, *args)
	if err != nil {
		return err
	}
	return c.SetFilter(f.(Filter))
}

func (c *Client) SetFilter(f Filter) error {
	c.filter = f
	return nil
}

func withBinding[V any](binder func(*Client, V) error, args []V) cli.Action {
	switch len(args) {
	case 0:
		return cli.BindContext(FromContext, binder)
	case 1:
		return cli.BindContext(FromContext, binder, args[0])
	default:
		panic("expected 0 or 1 arg")
	}
}

func lateBinding[V any](name string) func(context.Context) V {
	return func(c context.Context) V {
		ss := c.(*cli.Context).Value(name).(V)
		return ss
	}
}

func defaultUserAgent() string {
	version := build.Version
	if len(version) == 0 {
		version = "development"
	}
	return fmt.Sprintf("Go-http-client/1.1 (pastiche/%s, +%s)", version, pasticheURL)
}
