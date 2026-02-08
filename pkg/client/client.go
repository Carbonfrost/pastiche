// Copyright 2025, 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"time"

	cli "github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli-http/httpclient"
	"github.com/Carbonfrost/joe-cli/extensions/bind"
	"github.com/Carbonfrost/joe-cli/extensions/provider"
	"github.com/Carbonfrost/pastiche/pkg/config"
	"github.com/Carbonfrost/pastiche/pkg/grpcclient"
	"github.com/Carbonfrost/pastiche/pkg/internal/build"
	"github.com/Carbonfrost/pastiche/pkg/model"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

// Client provides the Pastiche client, which can invoke underlying HTTP or gRPC
// services
type Client struct {
	cli.Action

	http            *httpclient.Client
	grpc            *grpcclient.Client
	clientType      ClientType
	filter          Filter
	includeMetadata bool

	locationResolver httpclient.LocationResolver
}

// Option identifies a client option
type Option func(*Client)

const (
	servicesKey contextKey = "pastiche.client"
)

// New initializes a new client with the given set of options.
func New(opts ...Option) *Client {
	res := &Client{}
	res.Apply(opts...)

	sr := res.locationResolver
	client := httpclient.New(
		httpclient.WithDefaultUserAgent(build.DefaultUserAgent()),
		httpclient.WithLocationResolver(
			sr,
		),
		func(c *httpclient.Client) {
			c.UseDownloadMiddleware(res.filterResponse)
		},
		func(c *httpclient.Client) {
			c.UseDownloadMiddleware(func(downloader httpclient.Downloader) httpclient.Downloader {
				return newHistoryDownloader(downloader, res.historyLog)
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
	return res
}

func (c *Client) Apply(opts ...Option) {
	for _, o := range opts {
		o(c)
	}
}

func (c *Client) filterResponse(d httpclient.Downloader) httpclient.Downloader {
	var history historyGenerator
	if c.includeMetadata {
		history = c.historyLog
	}
	return NewFilterDownloader(c.filter, d, history)
}

// Type gets the client type that was requested
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

// ContextValue provides an action which sets the client into the context.
func ContextValue(c *Client) cli.Action {
	return cli.ContextValue(servicesKey, c)
}

// FlagsAndArgs provides an action which sets up flags and args used by the client.
// Despite its name, the action contributes no args.
func FlagsAndArgs() cli.Action {
	return cli.Pipeline(
		cli.AddFlags([]*cli.Flag{
			{Uses: ListFilters()},
			{Uses: SetFilter()},
			{Uses: SetClientType()},
			{Uses: SetIncludeMetadata()},
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

func (c *Client) SetIncludeMetadata(t bool) error {
	c.includeMetadata = t
	return nil
}

func (c *Client) historyLog(ctx context.Context, r *httpclient.Response) (*history, io.Writer) {
	resolver := c.locationResolver.(*serviceResolver)
	req, _ := resolver.resolveRequest(ctx)
	var vars map[string]any
	if req != nil {
		vars = req.Vars() // TODO Would be better to separate input vars from compiled
	}
	var responseBody bytes.Buffer
	return &history{
		Timestamp: time.Now(), // TODO To be persnickety, should be the exact request timing
		URL:       fmt.Sprint(r.Request.URL),
		Spec:      *resolver.root(ctx),
		Server:    resolver.server(ctx),
		Response: historyResponse{
			Headers:    r.Header,
			Status:     r.Status,
			StatusCode: r.StatusCode,
			Body:       &historyResponseBody{&responseBody},
		},
		Request: historyRequest{
			Headers: r.Request.Header,
			Method:  r.Request.Method,
		},
		Vars:    vars,
		BaseURL: sprintURL(resolver.base),
	}, &responseBody
}

func (o Option) Execute(c context.Context) error {
	o(FromContext(c))
	return nil
}

// WithLocationResolver sets the location resolver used by the client
func WithLocationResolver(value httpclient.LocationResolver) Option {
	return func(c *Client) {
		c.locationResolver = value
	}
}

// WithDefaultLocationResolver provides a client option which sets up the
// default location resolver, which uses the CLI arguments and flags named
// "service", "server", and "method"
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
	return bind.Call2(binder, bind.FromContext(FromContext), bind.Exact(args...))
}

func lateBinding[V any](name string) func(context.Context) V {
	return func(c context.Context) V {
		ss := c.(*cli.Context).Value(name).(V)
		return ss
	}
}
