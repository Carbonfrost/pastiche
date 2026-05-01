// Copyright 2023, 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package model

import (
	"flag"
	"fmt"
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
	return strings.Join(s, ".")
}

// ParseServiceSpec parses a service spec
func ParseServiceSpec(text string) (ServiceSpec, error) {
	names := strings.Split(text, ".")
	for index, name := range names {
		var err error
		if index == 0 {
			err = checkQName(name)
		} else {
			err = checkName(name)
		}
		if err != nil {
			return nil, fmt.Errorf("invalid name %q: %w", name, err)
		}
	}
	return ServiceSpec(names), nil
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
