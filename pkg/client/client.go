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
	"github.com/Carbonfrost/joe-cli-http/uritemplates"
	"github.com/Carbonfrost/joe-cli/extensions/bind"
	"github.com/Carbonfrost/joe-cli/extensions/provider"
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
	clientType      Type
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
		httpclient.WithUserAgent(build.DefaultUserAgent()),
		httpclient.WithLocationResolver(
			sr,
		),
		httpclient.WithDownloaderMiddleware(res.filterResponse),
		httpclient.WithDownloaderMiddleware(res.historyLogMiddleware),
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

func (c *Client) filterResponse(_ context.Context, d httpclient.Downloader) httpclient.Downloader {
	var history historyGenerator

	if f, ok := c.filter.(IncludeMetadataFilter); c.includeMetadata || (ok && f.IncludeMetadata()) {
		history = c.historyLog
	}

	return NewFilterDownloader(c.filter, d, history)
}

func (c *Client) historyLogMiddleware(_ context.Context, d httpclient.Downloader) httpclient.Downloader {
	return newHistoryDownloader(d, c.historyLog)
}

// Type gets the client type that was requested
func (c *Client) Type() Type {
	return c.clientType
}

func (c *Client) Pipeline() cli.Action {
	return c.Action
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

		cli.Customize(
			"-cert",
			cli.RemoveAlias("E"), // being used by --param-env
		),

		FilterRegistry,
		FlagsAndArgs(),
		ContextValue(c),
	)
}

// ContextValue provides an action which sets the client into the context.
func ContextValue(c *Client) cli.Action {
	return cli.WithContextValue(servicesKey, c)
}

// FlagsAndArgs provides an action which sets up flags and args used by the client.
// Despite its name, the action contributes no args.
func FlagsAndArgs() cli.Action {
	return cli.Pipeline(
		cli.AddFlags([]*cli.Flag{
			{Uses: ListFilters()},
			{Uses: SetFilter()},
			{Uses: SetType()},
			{Uses: SetIncludeMetadata()},
		}...),
	)
}

func (c *Client) setFilterHelper(v *provider.Value) error {
	args := v.Args.(*map[string]string)

	if _, ok := FilterRegistry.LookupProvider(v.Name); !ok {
		// If the filter name is not in the registry, it might be a named output
		// We'll create a wrapper that will resolve it at runtime
		return c.SetFilter(NewNamedOutputFilter(v.Name))
	}

	// Try to create a filter from the registry
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

func (c *Client) SetType(t Type) error {
	c.clientType = t
	return nil
}

func (c *Client) SetIncludeMetadata(t bool) error {
	c.includeMetadata = t
	return nil
}

func (c *Client) SetVarFromEnvVar(v *uritemplates.Var) error {
	v = VarFromEnv(v)
	return c.locationResolver.AddVar(v.Name, v.Value)
}

// VarFromEnv interprets the value of the URI template variable as an
// environment variable.
func VarFromEnv(v *uritemplates.Var) *uritemplates.Var {
	switch value := v.Value.(type) {
	case map[string]any:
		values := make(map[string]any)
		for k, v := range value {
			values[k] = os.Getenv(fmt.Sprint(v))
		}
		return uritemplates.MapVar(v.Name, values)

	case []any:
		values := make([]any, len(value))
		for i := range value {
			values[i] = os.Getenv(fmt.Sprint(value[i]))
		}
		return uritemplates.ArrayVar(v.Name, values...)
	default:
		return uritemplates.StringVar(v.Name, os.Getenv(fmt.Sprint(v.Value)))
	}
}

func (c *Client) historyLog(ctx context.Context, r *httpclient.Response) (*history, io.Writer) {
	resolver := c.locationResolver.(*serviceResolver)
	req, _ := resolver.resolveRequest(ctx)
	var vars map[string]any
	if req != nil {
		vars = req.Vars // TODO Would be better to separate input vars from compiled
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
func WithDefaultLocationResolver(m *model.Model) Option {
	sr := NewServiceResolver(
		m,
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
