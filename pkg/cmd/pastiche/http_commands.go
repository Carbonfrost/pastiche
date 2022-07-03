package pastiche

import (
	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli-http/httpclient"
	"github.com/Carbonfrost/pastiche/pkg/model"
)

func invokeUsingMethod(name string) cli.Action {
	return cli.Pipeline(
		cli.Setup{
			Uses: cli.Pipeline(
				cli.Category("Invoke HTTP client"),
				cli.AddArg(&cli.Arg{
					Name:  "service",
					Value: new(model.ServiceSpec),
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
