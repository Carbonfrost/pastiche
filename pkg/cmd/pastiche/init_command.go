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
