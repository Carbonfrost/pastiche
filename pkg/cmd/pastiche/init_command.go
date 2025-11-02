// Copyright 2023 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package pastiche

import (
	"github.com/Carbonfrost/joe-cli"
)

func InitCommand() cli.Action {
	return cli.Setup{
		Uses: cli.Pipeline(cli.AddFlags([]*cli.Flag{
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
		}...)),
		Action: func(c *cli.Context) error {
			return c.Do(initTemplate(c))
		},
	}
}
