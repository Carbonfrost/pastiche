// Copyright 2023, 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package model

import (
	"flag"
	"strings"

	"github.com/Carbonfrost/joe-cli"
)

type ServiceSpec []string

type serviceSpecCounter struct {
	base cli.ArgCounter
}

func newServiceSpecCounter() *serviceSpecCounter {
	return &serviceSpecCounter{
		base: cli.ArgCount(cli.TakeUntilNextFlag),
	}
}

func (s *ServiceSpec) Set(arg string) error {
	*s = append(*s, arg)
	return nil
}

func (s ServiceSpec) ServiceName() string {
	if len(s) == 0 {
		return ""
	}
	return s[0]
}

func (s ServiceSpec) NewCounter() cli.ArgCounter {
	return newServiceSpecCounter()
}

func (s ServiceSpec) String() string {
	return cli.Join(s)
}

func (s ServiceSpec) Path() string {
	if strings.ContainsAny(s[0], "/") {
		return strings.Join(s, ".")
	}
	return strings.Join(s, "/")
}

func (s *serviceSpecCounter) Take(arg string, possibleFlag bool) error {
	err := s.base.Take(arg, possibleFlag)
	if err != nil {
		return err
	}

	// This looks like it is specifying a template variable
	if strings.Contains(arg, "=") {
		return cli.EndOfArguments
	}

	return nil
}

func (*serviceSpecCounter) Done() error {
	return nil
}

var _ flag.Value = (*ServiceSpec)(nil)
