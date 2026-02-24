// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package workspace

import (
	"github.com/Carbonfrost/joe-cli"
)

// Log provides the action to access logs
func Log() cli.Action {
	return cli.Pipeline(
		cli.Prototype{
			Name:     "log",
			Aliases:  []string{"logs"},
			HelpText: "Access request logs for the workspace",
			Setup: cli.Setup{
				Uses: cli.AddFlags([]*cli.Flag{
					{Uses: ClearLogs()},
				}...),
			},
		},
	)
}

func ClearLogs() cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "clear",
			HelpText: "Remove all request logs in the workspace",
			Options:  cli.Exits,
			Value:    new(bool),
		},
		cli.At(cli.ActionTiming, cli.ActionOf(ClearLogDir)),
	)
}
