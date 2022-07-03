package pastiche

import (
	"fmt"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/pastiche/pkg/config"
	"github.com/Carbonfrost/pastiche/pkg/model"
)

func DescribeServiceCommand() cli.Action {
	return cli.Setup{
		Uses: cli.Pipeline(
			cli.AddArgs([]*cli.Arg{
				{
					Name:  "name",
					Value: cli.List(),
				},
			}...)),
		Action: func(c *cli.Context) error {
			name := c.List("name")
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			if len(name) == 0 {
				for _, s := range cfg.Services {
					displayService(s)
				}
				return nil
			}

			for _, nom := range name {
				s, ok := cfg.Service(nom)
				if !ok {
					return fmt.Errorf("service not found: %q", nom)
				}
				displayService(s)
			}
			return nil
		},
	}
}

func displayService(s *model.Service) {
	fmt.Printf("%#v\n", s)
}
