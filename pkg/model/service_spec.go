package model

import (
	"flag"
	"strings"

	"github.com/Carbonfrost/joe-cli"
)

type ServiceSpec []string

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
	return cli.ArgCount(cli.TakeUntilNextFlag)
}

func (s ServiceSpec) String() string {
	return cli.Join(s)
}

func (s ServiceSpec) Path() string {
	return strings.Join(s, "/")
}

var _ flag.Value = (*ServiceSpec)(nil)
