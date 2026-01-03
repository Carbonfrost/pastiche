// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grpcclient

import (
	"context"
	"reflect"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/bind"
)

const (
	requestOptions = "Request options"
)

var (
	tagged  = cli.Data(SourceAnnotation())
	pkgPath = reflect.TypeFor[Client]().PkgPath()
)

func FetchAndPrint() cli.Action {
	return cli.ActionFunc(func(c *cli.Context) error {
		_, err := Do(c)
		return err
	})
}

func ContextValue(c *Client) cli.Action {
	return cli.ContextValue(servicesKey, c)
}

func FromContext(c context.Context) *Client {
	return c.Value(servicesKey).(*Client)
}

func Do(c *cli.Context) ([]*Response, error) {
	return FromContext(c).Do(c)
}

func FlagsAndArgs() cli.Action {
	return cli.Pipeline(
		cli.AddFlags(
			[]*cli.Flag{}...,
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

func SetAddr(s ...string) cli.Action {
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

func SetSymbol(s ...string) cli.Action {
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
func bindAction[T any](fn func(T) Option, t bind.Binder[T]) cli.Action {
	cfn := func(t T) cli.Action {
		return fn(t)
	}
	return bind.Action(cfn, t)
}
