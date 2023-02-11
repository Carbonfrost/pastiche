package pastiche

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/pastiche/pkg/config"
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
			name := c.String("name")
			if name == "" {
				name = "service"
				wd, err := os.Getwd()
				if err == nil {
					name = filepath.Base(wd)
				}
			}

			svc := &config.Service{
				Title:       c.String("title"),
				Name:        name,
				Description: c.String("description"),
				Servers: []*config.Server{
					{
						Name:    "default",
						BaseURL: "http://localhost:8000/",
					},
				},
				Resource: &config.Resource{
					Name: "/",
					Resources: []*config.Resource{
						{
							Name: "get",
							URI:  "/",
							Get:  &config.Endpoint{},
						},
					},
				},
			}

			_ = os.Mkdir(".pastiche", 0755)
			f, err := os.Create(".pastiche/" + name + ".json")
			if err != nil {
				return err
			}

			b, err := json.MarshalIndent(svc, "", "  ")
			if err != nil {
				return err
			}

			_, err = f.Write(b)
			return err
		},
	}
}
