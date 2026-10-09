// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grpcclient

import (
	"context"
	"reflect"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli-http/httpclient"
	"github.com/Carbonfrost/joe-cli/extensions/bind"
)

const (
	requestOptions = "Request options"
)

var (
	tagged           = cli.Data(SourceAnnotation())
	synopsisCategory = cli.SynopsisCategory("grpc-client")
	pkgPath          = reflect.TypeFor[Client]().PkgPath()
)

type Action = cli.Action

func FetchAndPrint() Action {
	return cli.ActionFunc(func(c *cli.Context) error {
		_, err := Do(c)
		return err
	})
}

func ContextValue(c *Client) Action {
	return cli.WithContextValue(servicesKey, c)
}

func FromContext(c context.Context) *Client {
	return c.Value(servicesKey).(*Client)
}

func Do(c *cli.Context) ([]*Response, error) {
	return FromContext(c).Do(c)
}

func FlagsAndArgs() Action {
	return cli.Pipeline(
		cli.AddFlags(
			[]*cli.Flag{
				idFlag(IDPlaintext, SetPlaintext()),
				idFlag(IDDisableReflection, SetDisableReflection()),
				idFlag(IDProtoset, SetProtoset()),
				idFlag(IDHeader, SetHeader()),
			}...,
		),
		cli.AddArgs(
			[]*cli.Arg{
				{Uses: SetAddr()},
				{Uses: SetSymbol()},
			}...,
		),
	)
}

// SourceAnnotation gets the name and value of the annotation added to the Data
// of all flags that are initialized from this package
func SourceAnnotation() (string, string) {
	return "Source", pkgPath
}

func SetPlaintext(s ...bool) Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "plaintext",
			HelpText: "Activate plaintext mode and disable TLS",
			Category: requestOptions,
		},
		bindAction(WithPlaintext, bind.Exact(s...)),
		tagged,
		synopsisCategory,
	)
}

func SetDisableReflection(s ...bool) Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "disable-reflection",
			HelpText: "Disable server gRPC schema reflection",
			Category: requestOptions,
		},
		bindAction(WithDisableReflection, bind.Exact(s...)),
		tagged,
		synopsisCategory,
	)
}

// TODO joe@futures should allow this to be typed as File

func SetProtoset(s ...string) Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "protoset",
			HelpText: "Provide FILE with grpc protoset schema",
			Category: requestOptions,
			Options:  cli.MustExist | cli.EachOccurrence,
		},
		bindAction(WithProtoset, bind.Exact(s...)),
		tagged,
		synopsisCategory,
	)
}

func SetHeader(s ...*httpclient.HeaderValue) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "header",
			Aliases:  []string{"H"},
			HelpText: "Sets header to {NAME} and {VALUE}",
			Category: requestOptions,
			Options:  cli.EachOccurrence,
		},
		bindAction(AddHeader, bind.Exact(s...)),
		tagged,
		synopsisCategory,
	)
}

func SetAddr(s ...string) Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "addr",
			HelpText: "Connect to grpc server {ADDRESS}",
			Category: requestOptions,
		},
		bindAction(WithAddr, bind.Exact(s...)),
		tagged,
	)
}

func SetSymbol(s ...string) Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "symbol",
			HelpText: "Name of {SYMBOL} to invoke on the grpc server",
			Category: requestOptions,
		},
		bindAction(WithSymbol, bind.Exact(s...)),
		tagged,
	)
}

// TODO These shouldn't be needed once joe-cli@future support covariance
func bindAction[T any](fn func(T) Option, t bind.Binder[T]) Action {
	cfn := func(t T) Action {
		return fn(t)
	}
	return bind.Action(cfn, t)
}
