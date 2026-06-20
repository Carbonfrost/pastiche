// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package workspace

import (
	cli "github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/bind"
)

func Init() cli.Action {
	return cli.Pipeline(
		cli.Prototype{
			HelpText: "Initialize the current directory with a new service definition",
			Uses: cli.AddFlags([]*cli.Flag{
				{
					Name:     "name",
					HelpText: "Name of the service",
				},
				{
					Name:     "title",
					HelpText: "Title of the service",
				},
				{
					Name:     "description",
					HelpText: "Short description of the service",
				},
			}...),
		},
		cli.At(cli.ActionTiming, NewInitServiceCommand()),
	)
}

func Env() cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "env",
			HelpText: "Display information about the Pastiche environment",
			Options:  cli.Exits,
			Value:    new(bool),
		},
		bind.Call(nilError((*Workspace).PrintEnv), bind.FromContext(FromContext)),
	)
}

// Log provides the action to access logs
func Log() cli.Action {
	return cli.Pipeline(
		cli.Prototype{
			Name:     "log",
			Aliases:  []string{"logs"},
			HelpText: "Access request logs for the workspace",
			Uses: cli.AddFlags([]*cli.Flag{
				{Uses: ClearLogs()},
			}...),
		},
	)
}

// ClearLogs removes logs from the workspace
func ClearLogs() cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "clear",
			HelpText: "Remove all request logs in the workspace",
			Options:  cli.Exits,
			Value:    new(bool),
		},
		bind.Call((*Workspace).ClearLogDir, bind.FromContext(FromContext)),
	)
}

func nilError(fn func(*Workspace)) func(*Workspace) error {
	return func(w *Workspace) error {
		fn(w)
		return nil
	}
}
