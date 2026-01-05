// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grpcclient

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli-http/httpclient"
	"github.com/fullstorydev/grpcurl"
	"github.com/jhump/protoreflect/grpcreflect"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

type Response = httpclient.Response

type Client struct {
	cli.Action

	network string
	address string
	symbol  string
	creds   credentials.TransportCredentials

	headers []string

	// interop with httpclient
	locationResolver httpclient.LocationResolver
}

type Option func(*Client)

type contextKey string

const servicesKey contextKey = "grpcclient_services"

func New(opts ...Option) *Client {
	c := &Client{}
	for _, o := range opts {
		o(c)
	}
	c.Action = defaultAction(c)
	return c
}

func defaultAction(c *Client) cli.Action {
	return cli.Pipeline(
		FlagsAndArgs(),
		ContextValue(c),
	)
}

func (c *Client) Do(ctx context.Context) ([]*Response, error) {
	var responses []*Response

	if c.locationResolver != nil {
		locations, err := c.locationResolver.Resolve(ctx)
		if err != nil {
			return nil, err
		}

		for _, l := range locations {
			resp, err := c.doOne(ctx, l)
			if err != nil {
				return nil, err
			}
			responses = append(responses, resp)
		}

	} else {
		resp, err := fetchAndPrintCore(ctx, c, c.address, c.symbol)
		if err != nil {
			return nil, err
		}

		responses = append(responses, resp)
	}

	return responses, nil
}

func (c *Client) doOne(ctx context.Context, l httpclient.Location) (*Response, error) {
	uctx, u, err := l.URL(ctx)
	if err != nil {
		return nil, err
	}

	address := u.Host
	symbol := strings.TrimPrefix(u.Path, "/")
	return fetchAndPrintCore(uctx, c, address, symbol)
}

func WithLocationResolver(value httpclient.LocationResolver) Option {
	return func(c *Client) {
		c.locationResolver = value
	}
}

func WithAddr(value string) Option {
	return func(c *Client) {
		c.address = value
	}
}

func WithSymbol(value string) Option {
	return func(c *Client) {
		c.symbol = value
	}
}

func WithHeader(name, value string) Option {
	return func(c *Client) {
		c.headers = append(c.headers, fmt.Sprintf("%s:%s", name, value))
	}
}

func (o Option) Execute(c context.Context) error {
	o(FromContext(c))
	return nil
}

func fetchAndPrintCore(ctx context.Context, c *Client, target, methodName string) (*Response, error) {
	options := grpcurl.FormatOptions{
		EmitJSONDefaultFields: true,
		IncludeTextSeparator:  true,
		AllowUnknownFields:    true,
	}

	var addlHeaders []string
	addlHeaders = append(addlHeaders, c.headers...)

	// TODO Read document from correct source
	in := os.Stdin

	// Apply reflection
	md := grpcurl.MetadataFromHeaders(addlHeaders)
	refCtx := metadata.NewOutgoingContext(ctx, md)
	var creds credentials.TransportCredentials

	cc, err := grpcurl.BlockingDial(ctx, "", target, creds)
	if err != nil {
		return nil, fmt.Errorf("failed to dial target host %q: %w", target, err)
	}

	refClient := grpcreflect.NewClientAuto(refCtx, cc)
	refClient.AllowMissingFileDescriptors()
	reflSource := grpcurl.DescriptorSourceFromServer(ctx, refClient)

	descSource := reflSource

	rf, formatter, err := grpcurl.RequestParserAndFormatter(
		grpcurl.Format(grpcurl.FormatJSON),
		descSource, in, options,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to construct request parser and formatter: %w", err)
	}

	eventHandler := &grpcurl.DefaultEventHandler{
		Out:            os.Stdout,
		Formatter:      formatter,
		VerbosityLevel: 0,
	}

	// TODO Actually provide response and filter support
	return nil, grpcurl.InvokeRPC(ctx, descSource, cc, methodName, addlHeaders, eventHandler, rf.Next)
}

var _ cli.Action = Option(nil)
