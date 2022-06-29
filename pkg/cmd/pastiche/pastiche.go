package pastiche

import (
	"net/url"
	"os"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli-http/httpclient"
	"github.com/Carbonfrost/joe-cli/extensions/color"
	"github.com/Carbonfrost/pastiche/pkg/config"
	"github.com/Carbonfrost/pastiche/pkg/model"
)

func Run() {
	NewApp().Run(os.Args)
}

func NewApp() *cli.App {
	return &cli.App{
		Name:     "pastiche",
		HelpText: "Make requests to HTTP APIs using their OpenAPI schemas and definitions.",
		Comment:  "Smart OpenAPI client",
		Options:  cli.Sorted,
		Uses: cli.Pipeline(
			&color.Options{},
			httpclient.New(),
			cli.RemoveArg(-1), // Remove URL contributed by http client
			cli.AddArg(&cli.Arg{
				Name:  "service",
				Value: new(model.ServiceSpec),
			}),
		),
		Action: cli.Pipeline(
			selectAPIResourceFromSpec(),
			httpclient.FetchAndPrint(),
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

// selectAPIResourceFromSpec uses the service spec to determine which API resource
// is being selected and configures the HTTP client so that when it processes it, it will
// invoke the API request
func selectAPIResourceFromSpec() cli.ActionFunc {
	return func(c *cli.Context) error {
		ss := c.Value("service").(*model.ServiceSpec)
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		service, resource, err := cfg.Resolve(*ss)
		if err != nil {
			return err
		}

		var baseURL *url.URL
		if len(service.Servers) > 0 {
			baseURL, _ = url.Parse(service.Servers[0].BaseURL)
		}

		client := httpclient.FromContext(c)
		return resource.ApplyToClient(client, baseURL)
	}
}
