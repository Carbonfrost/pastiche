package pastiche

import (
	"os"
	"path/filepath"

	"github.com/Carbonfrost/joe-cli"
	. "github.com/Carbonfrost/joe-cli/extensions/template"
	"github.com/Carbonfrost/pastiche/pkg/config"
)

func initTemplate(c *cli.Context) *Root {
	svc := service(c)
	return New(
		Dir(".pastiche",
			Vars{
				"ServiceName": svc.Name,
			},
			File("{{ .ServiceName }}.json", Contents(svc)),
		),
	)
}

func serviceName(c *cli.Context) string {
	name := c.String("name")
	if name == "" {
		name = "service"
		wd, err := os.Getwd()
		if err == nil {
			return filepath.Base(wd)
		}
	}
	return name
}

func service(c *cli.Context) *config.Service {
	return &config.Service{
		Title:       c.String("title"),
		Name:        serviceName(c),
		Description: c.String("description"),
		Servers: []config.Server{
			{
				Name:    "default",
				BaseURL: "http://localhost:8000/",
			},
		},
		Resources: []config.Resource{
			{
				Name: "get",
				URI:  "/",
				Get:  &config.Endpoint{},
			},
		},
	}
}
