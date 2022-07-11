package pastiche

import (
	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli-http/httpclient"
	"github.com/Carbonfrost/pastiche/pkg/config"
	"github.com/Carbonfrost/pastiche/pkg/model"
)

func invokeUsingMethod(name string) cli.Action {
	return cli.Pipeline(
		cli.Setup{
			Uses: cli.Pipeline(
				cli.Category("Invoke HTTP client"),
				cli.AddArg(&cli.Arg{
					Name:       "service",
					Value:      new(model.ServiceSpec),
					Completion: completeServices(),
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
				func(c *cli.Context) {
					httpclient.FromContext(c).SetMethod(name)
				},
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
			svc, _, err := cfg.Resolve(*v)
			if err != nil {
				return nil
			}

			names := make([]string, 0, len(svc.Servers))
			for _, s := range svc.Servers {
				names = append(names, s.Name)
			}
			return cli.CompletionValues(names...).Complete(cc)
		}
		return nil
	}
}
