package model

import (
	"flag"
	"strings"

	"github.com/Carbonfrost/joe-cli"
)

type ServiceSpec []string

type serviceCounter struct {
	count int
}

// Pastiche will take either the path to a service or will ignore built-in
// or reserved command names
var (
	commandNames = map[string]bool{
		// In-use commands
		"help":     true,
		"init":     true,
		"version":  true,
		"describe": true,

		// Reserved for future use
		"env": true,

		// HTTP Method names
		"connect": true,
		"delete":  true,
		"get":     true,
		"head":    true,
		"options": true,
		"patch":   true,
		"post":    true,
		"put":     true,
	}
)

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
	return &serviceCounter{}
}

func (s ServiceSpec) String() string {
	return cli.Join(s)
}

func (s ServiceSpec) Path() string {
	return strings.Join(s, "/")
}

func (o *serviceCounter) Take(arg string, possibleFlag bool) error {
	if possibleFlag && strings.HasPrefix(arg, "-") {
		return cli.EndOfArguments
	}

	// If the arg looks like a command name, then do not take it
	if commandNames[arg] {
		return cli.EndOfArguments
	}
	o.count++
	return nil
}

func (*serviceCounter) Done() error {
	return nil
}

var _ flag.Value = (*ServiceSpec)(nil)
