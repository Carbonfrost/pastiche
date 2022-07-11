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
