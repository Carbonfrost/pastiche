package pastiche

import (
	"bytes"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli-http/httpclient"
	"github.com/Carbonfrost/pastiche/pkg/config"
	"github.com/Carbonfrost/pastiche/pkg/model"
)

func invokeUsingMethod() cli.Action {
	return cli.Pipeline(
		cli.Setup{
			Uses: cli.Pipeline(
				cli.Category("Invoke HTTP client"),
				cli.AddArg(&cli.Arg{
					Name:       "service",
					Value:      new(model.ServiceSpec),
					Completion: completeServices(),
					Uses: func(c *cli.Context) {
						c.Arg().Description = renderServices(c)
					},
				}),
				cli.AddFlags([]*cli.Flag{
					{
						Name:       "server",
						Aliases:    []string{"S"},
						HelpText:   "Use the specified server for the request",
						Value:      new(string),
						Completion: completeServer(),
					},
				}...),
			),
			Action: cli.Pipeline(
				httpclient.FetchAndPrint(),
			),
		})
}

func completeServices() cli.CompletionFunc {
	return func(cc *cli.CompletionContext) []cli.CompletionItem {
		cfg, _ := config.Load()
		names := make([]string, 0, len(cfg.Services))
		for _, s := range cfg.Services {
			names = append(names, s.Name)
		}
		return cli.CompletionValues(names...).Complete(cc)
	}
}

func completeServer() cli.CompletionFunc {
	return func(cc *cli.CompletionContext) []cli.CompletionItem {
		if v, ok := cc.Context.Value("service").(*model.ServiceSpec); ok {
			cfg, _ := config.Load()
			mo := model.New(cfg)
			service, _, err := mo.Resolve(*v)
			if err != nil {
				return nil
			}

			names := make([]string, 0, len(service.Servers))
			for _, s := range service.Servers {
				names = append(names, s.Name)
			}
			return cli.CompletionValues(names...).Complete(cc)
		}
		return nil
	}
}

func renderServices(c *cli.Context) string {
	cfg, _ := config.Load()
	mo := model.New(cfg)
	items := []*model.Service{}
	for _, v := range mo.Services {
		items = append(items, v)
	}

	data := struct {
		Services []*model.Service
	}{
		Services: items,
	}

	var buf bytes.Buffer
	c.Template("PasticheServices").Execute(&buf, data)
	return buf.String()
}
