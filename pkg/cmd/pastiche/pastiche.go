package pastiche

import (
	"context"
	"os"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli-http/httpclient"
	"github.com/Carbonfrost/joe-cli/extensions/color"
	"github.com/Carbonfrost/pastiche/pkg/config"
	phttpclient "github.com/Carbonfrost/pastiche/pkg/httpclient"
	"github.com/Carbonfrost/pastiche/pkg/model"
)

func Run() {
	NewApp().Run(os.Args)
}

func NewApp() *cli.App {
	cfg, _ := config.Load()
	return &cli.App{
		Name:     "pastiche",
		HelpText: "Make requests to HTTP APIs using their OpenAPI schemas and definitions.",
		Comment:  "Smart OpenAPI client",
		Options:  cli.Sorted,
		Uses: cli.Pipeline(
			&color.Options{},
			httpclient.New(
				httpclient.WithLocationResolver(
					phttpclient.NewServiceResolver(cfg, func(c context.Context) *model.ServiceSpec {
						ss := c.(*cli.Context).Value("service").(*model.ServiceSpec)
						return ss
					}),
				),
			),
			cli.RemoveArg("url"), // Remove URL contributed by http client
			cli.ImplicitCommand("get"),
		),
		Before: cli.Pipeline(
			suppressHTTPClientHelpByDefault(),
		),
		Commands: []*cli.Command{
			{
				Name:     "init",
				HelpText: "Initialize the current directory with a new service definition",
				Uses:     InitCommand(),
			},
			{
				Name:     "describe",
				HelpText: "Describe resources within Pastiche workspace",
				Subcommands: []*cli.Command{
					{
						Name:    "service",
						Aliases: []string{"services", "svc"},
						Uses:    DescribeServiceCommand(),
					},
				},
			},
			{Name: "get", Uses: invokeUsingMethod("GET")},
			{Name: "head", Uses: invokeUsingMethod("HEAD")},
			{Name: "post", Uses: invokeUsingMethod("POST")},
			{Name: "put", Uses: invokeUsingMethod("PUT")},
			{Name: "patch", Uses: invokeUsingMethod("PATCH")},
			{Name: "delete", Uses: invokeUsingMethod("DELETE")},
		},
		Flags: []*cli.Flag{
			{
				Name:     "chdir",
				HelpText: "Change directory into the specified working {DIRECTORY}",
				Value:    new(cli.File),
				Options:  cli.WorkingDirectory | cli.NonPersistent,
			},
			{
				Name:        "all",
				HelpText:    "Facilitates displaying help text that is suppressed by default (used with --help)",
				Value:       cli.Bool(),
				Options:     cli.NonPersistent,
				Description: "'pastiche --help --all' displays information about HTTP client options and other advanced features",
			},
		},
	}
}

func suppressHTTPClientHelpByDefault() cli.ActionFunc {
	// There are quite a number of options to display for the HTTP client, so
	// hide these until they are requested explicitly
	return func(c *cli.Context) error {
		if c.Seen("all") {
			return nil
		}
		n, v := httpclient.SourceAnnotation()
		return c.Walk(func(cmd *cli.Context) error {
			for _, f := range cmd.Command().Flags {

				if f.Data[n] == v {
					f.SetHidden()
				}
			}
			return nil
		})
	}
}
