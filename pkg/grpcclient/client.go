// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grpcclient

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli-http/httpclient"
	"github.com/Carbonfrost/pastiche/pkg/internal/build"
	"github.com/Carbonfrost/pastiche/pkg/model"
	"github.com/fullstorydev/grpcurl"
	"github.com/jhump/protoreflect/grpcreflect"
	"google.golang.org/grpc"
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

	protoset          []string
	disableReflection bool
	plaintext         bool

	body              io.ReadCloser
	headers           []string
	reflectionHeaders []string // TODO Allow setting reflection headers

	// interop with httpclient
	locationResolver httpclient.LocationResolver
}

type modelLocation interface {
	Resolved() model.ResolvedResource
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
		// TODO Read document from correct source
		c.body = os.Stdin

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

	// TODO Like httpclient, it would be better to communicate with the underlying
	// layer via Middleware or another convention rather than depend upon model.

	// TODO Would be better to apply opts to the specific invocation than globally
	// to the client
	if m, ok := l.(modelLocation); ok {
		c.copyOpts(m.Resolved().Client())

		// TODO Use variables from the location resolver
		request, err := m.Resolved().EvalRequest(nil, nil)
		if err != nil {
			return nil, err
		}
		c.headers = formatHeaders(request.Headers())
		c.body = request.Body()
	}

	address := u.Host
	symbol := strings.TrimPrefix(u.Path, "/")
	return fetchAndPrintCore(uctx, c, address, symbol)
}

func (c *Client) copyOpts(clientOpts model.Client) {
	opts, _ := clientOpts.(*model.GRPCClient)
	if opts == nil {
		return
	}

	c.disableReflection = opts.DisableReflection
	c.plaintext = opts.Plaintext
	if opts.ProtoSet != "" {
		c.protoset = []string{opts.ProtoSet}
	}
}

func formatHeaders(m map[string][]string) []string {
	var res []string
	for k, v := range m {
		res = append(res, fmt.Sprintf("%s: %s", k, strings.Join(v, ",")))
	}
	return res
}

func WithLocationResolver(value httpclient.LocationResolver) Option {
	return func(c *Client) {
		c.locationResolver = value
	}
}

func WithDisableReflection(value bool) Option {
	return func(c *Client) {
		c.disableReflection = value
	}
}

func WithProtoset(value string) Option {
	return func(c *Client) {
		c.protoset = append(c.protoset, value)
	}
}

func WithPlaintext(value bool) Option {
	return func(c *Client) {
		c.plaintext = value
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

func (c *Client) dial(ctx context.Context, target string) (*grpc.ClientConn, error) {
	// TODO Should support customized dial and connect timeouts
	dialTime := 10 * time.Second
	ctx, cancel := context.WithTimeout(ctx, dialTime)
	defer cancel()

	var opts []grpc.DialOption
	var creds credentials.TransportCredentials

	if c.plaintext {
		// TODO Should support configuring authority

	} else {
		tlsConf := clientTLSConfig(ctx)
		creds = credentials.NewTLS(tlsConf)
	}

	opts = append(opts, grpc.WithUserAgent(build.DefaultUserAgent()))

	cc, err := grpcurl.BlockingDial(ctx, "", target, creds, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to dial target host %q: %w", target, err)
	}
	return cc, nil
}

func (c *Client) descSource(ctx context.Context, target string) (grpcurl.DescriptorSource, error) {
	var fileSource grpcurl.DescriptorSource
	if len(c.protoset) > 0 {
		var err error
		fileSource, err = grpcurl.DescriptorSourceFromProtoSets(c.protoset...)
		if err != nil {
			return nil, fmt.Errorf("failed to process proto descriptor sets: %w", err)
		}
	}

	if !c.disableReflection {
		md := grpcurl.MetadataFromHeaders(append(c.headers, c.reflectionHeaders...))
		refCtx := metadata.NewOutgoingContext(ctx, md)

		cc, err := c.dial(ctx, target)
		if err != nil {
			return nil, fmt.Errorf("failed to reflect service: %w", err)
		}

		refClient := grpcreflect.NewClientAuto(refCtx, cc)
		refClient.AllowMissingFileDescriptors()
		reflSource := grpcurl.DescriptorSourceFromServer(ctx, refClient)
		if fileSource != nil {
			panic("not implemented: composite reflection and file source")

		} else {
			return reflSource, nil
		}
	} else {
		return fileSource, nil
	}
}

func fetchAndPrintCore(ctx context.Context, c *Client, target, methodName string) (*Response, error) {
	options := grpcurl.FormatOptions{
		EmitJSONDefaultFields: true,
		IncludeTextSeparator:  true,
		AllowUnknownFields:    true,
	}

	var addlHeaders []string
	addlHeaders = append(addlHeaders, c.headers...)

	in := c.body

	cc, err := c.dial(ctx, target)
	if err != nil {
		return nil, fmt.Errorf("failed to dial target host %q: %w", target, err)
	}

	descSource, err := c.descSource(ctx, target)
	if err != nil {
		return nil, err
	}

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

func clientTLSConfig(c context.Context) *tls.Config {
	// TODO Requires an update to joe-cli-http@future to use the TLS-specific package
	return httpclient.FromContext(c).TLSConfig()
}

var _ cli.Action = Option(nil)
