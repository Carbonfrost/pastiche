package pastiche

import (
	"os"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli-http/httpclient"
	"github.com/Carbonfrost/joe-cli/extensions/color"
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
			&httpclient.Options{},
			cli.RemoveArg(-1), // Remove URL contributed by http clietnm
		),
		Before: cli.Pipeline(
			suppressHTTPClientHelpByDefault(),
		),
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
