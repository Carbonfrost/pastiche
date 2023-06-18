package httpclient_test

import (
	"context"
	"net/http"

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

	Describe("Resolve", func() {

		DescribeTable("examples", func(s string, expected string) {
			r := phttpclient.NewServiceResolver(exampleModel, specTo(s), serverTo(""))
			loc, err := r.Resolve(context.Background())
			Expect(err).NotTo(HaveOccurred())

			_, u, _ := loc[0].URL(context.Background())
			Expect(u.String()).To(Equal(expected))
		},
			// For compatibility with Wig:
			Entry("localhost", "localhost", "http://localhost"),
			Entry("example.com", "example.com", "http://example.com"),
			Entry("port", ":8080", "http://localhost:8080"),
			Entry("rooted", "/root", "/root"),
			// IP addresses should get treated as URLs
			Entry("IPv4", "192.168.1.19", "http://192.168.1.19"),
			Entry("IPv6", "2001:db8::8a2e:370:7334", "http://2001:db8::8a2e:370:7334"),
		)
	})

})

var _ = Describe("pasticheMiddleware", func() {

	It("returns an error on no endpoint", func() {
		ctx := phttpclient.NewContextWithLocation([]string{"service", "spec"},
			nil,
			nil,
			nil,
			nil, // no endpoint
			nil)

		req, _ := http.NewRequestWithContext(ctx, "GET", "https://example.com", nil)
		mw := phttpclient.NewServiceResolverMiddleware()
		err := mw.Handle(req)

		Expect(err).To(MatchError("no endpoint defined for service/spec"))
	})

	Describe("headers", func() {

		DescribeTable("examples", func(expected types.GomegaMatcher) {
			ctx := phttpclient.NewContextWithLocation(nil,
				&model.Resource{Headers: http.Header{"X-From-Resource": []string{"resource"}}},
				&model.Service{},
				&model.Server{
					Name:    "default",
					Headers: http.Header{"X-From-Server": []string{"server"}},
				},
				&model.Endpoint{Headers: http.Header{"X-From-Endpoint": []string{"endpoint"}}},
				nil)

			req, _ := http.NewRequestWithContext(ctx, "GET", "https://example.com", nil)
			mw := phttpclient.NewServiceResolverMiddleware()
			mw.Handle(req)

			Expect(req.Header).To(expected)
		},
			Entry("copied from resource", HaveKeyWithValue("X-From-Resource", []string{"resource"})),
			Entry("copied from endpoint", HaveKeyWithValue("X-From-Endpoint", []string{"endpoint"})),
			Entry("copied from server", HaveKeyWithValue("X-From-Server", []string{"server"})),
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
