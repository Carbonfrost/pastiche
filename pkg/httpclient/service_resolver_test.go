package httpclient_test

import (
	"context"

	"github.com/Carbonfrost/joe-cli-http/httpclient"
	"github.com/Carbonfrost/joe-cli-http/uritemplates"
	phttpclient "github.com/Carbonfrost/pastiche/pkg/httpclient"
	"github.com/Carbonfrost/pastiche/pkg/model"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("ServiceResolver", func() {

	var (
		exampleModel = &model.Model{
			Services: map[string]*model.Service{
				"hasNoServers": {
					Name:    "hasNoServers",
					Servers: []*model.Server{},
					Resource: &model.Resource{
						URITemplate: mustParseURITemplate("https://example.com"),
					},
				},
			},
		}

		specTo = func(names ...string) func(context.Context) *model.ServiceSpec {
			return func(context.Context) *model.ServiceSpec {
				ss := model.ServiceSpec(names)
				return &ss
			}
		}

		serverTo = func(name string) func(context.Context) string {
			return func(context.Context) string {
				return name
			}
		}
	)

	Describe("NewServiceResolver", func() {

		DescribeTable("errors",
			func(subject httpclient.LocationResolver, errExpected types.GomegaMatcher) {
				_, err := subject.Resolve(context.TODO())
				Expect(err).To(errExpected)
			},
			Entry(
				"default server requested but has no servers",
				phttpclient.NewServiceResolver(exampleModel, specTo("hasNoServers"), serverTo("")),
				MatchError(`no servers defined for service "hasNoServers"`)),
		)
	})
})

func mustParseURITemplate(t string) *uritemplates.URITemplate {
	u, err := uritemplates.Parse(t)
	if err != nil {
		panic(err)
	}
	return u
}
