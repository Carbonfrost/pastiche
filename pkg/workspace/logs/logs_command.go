package logs

import (
	cli "github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/bind"
)

type Action = cli.Action

// Clear removes logs from the workspace
func Clear(l *Log) Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "clear",
			HelpText: "Remove all request logs in the workspace",
			Options:  cli.Exits,
			Value:    new(bool),
		},
		// TODO joe-cli@futures removes this extraneous generics
		bind.Call0[cli.Action](l.Clear),
	)
}
