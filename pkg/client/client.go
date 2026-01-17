// Copyright 2025, 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"context"
	"os"

	cli "github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli-http/httpclient"
	"github.com/Carbonfrost/joe-cli/extensions/provider"
	"github.com/Carbonfrost/pastiche/pkg/config"
	"github.com/Carbonfrost/pastiche/pkg/grpcclient"
	"github.com/Carbonfrost/pastiche/pkg/internal/build"
	"github.com/Carbonfrost/pastiche/pkg/model"
)

type Client struct {
	cli.Action

	http       *httpclient.Client
	grpc       *grpcclient.Client
	clientType ClientType
	filter     Filter

	locationResolver httpclient.LocationResolver
}

type Option func(*Client)

const (
	servicesKey contextKey = "pastiche.client"
)

func New(opts ...Option) *Client {
	res := &Client{}
	for _, o := range opts {
		o(res)
	}

	sr := res.locationResolver
	client := httpclient.New(
		httpclient.WithDefaultUserAgent(build.DefaultUserAgent()),
		httpclient.WithLocationResolver(
			sr,
		),
		func(c *httpclient.Client) {
			c.UseDownloadMiddleware(func(downloader httpclient.Downloader) httpclient.Downloader {
				return newHistoryDownloader(downloader, sr.(*serviceResolver))
			})
		},
	)

	res.http = client
	res.grpc = grpcclient.New(
		grpcclient.WithLocationResolver(
			sr,
		),
	)
	res.Action = defaultAction(res)
	client.UseDownloadMiddleware(res.filterResponse)
	return res
}

func (c *Client) filterResponse(d httpclient.Downloader) httpclient.Downloader {
	if c.filter == nil {
		return d
	}
	return NewFilterDownloader(c.filter, d)
}

func (c *Client) Type() ClientType {
	return c.clientType
}

// FromContext obtains the client stored in the context
func FromContext(c context.Context) *Client {
	return c.Value(servicesKey).(*Client)
}

func defaultAction(c *Client) cli.Action {
	return cli.Pipeline(
		c.http,
		cli.RemoveArg(0), // Remove URL contributed by http client

		c.grpc,
		cli.RemoveArg(0), // Remove address and symbol contributed by client
		cli.RemoveArg(0),

		// TODO This will be available from joe-cli-http@futures
		cli.Customize(
			"-param",
			cli.ValueTransform(cli.TransformOptionalFileReference(cli.NewSysFS(cli.DirFS("."), os.Stdin, os.Stdout))),
		),

		FilterRegistry,
		FlagsAndArgs(),
		ContextValue(c),
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
			{Uses: SetClientType()},
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

func (c *Client) SetClientType(t ClientType) error {
	c.clientType = t
	return nil
}

func (o Option) Execute(c context.Context) error {
	o(FromContext(c))
	return nil
}

func WithLocationResolver(value httpclient.LocationResolver) Option {
	return func(c *Client) {
		c.locationResolver = value
	}
}

func WithDefaultLocationResolver() Option {
	cfg, _ := config.Load()
	sr := NewServiceResolver(
		model.New(cfg),
		lateBinding[*model.ServiceSpec]("service"),
		lateBinding[string]("server"),
		lateBinding[string]("method"),
	)
	return WithLocationResolver(sr)
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
