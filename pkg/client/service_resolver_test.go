// Copyright 2023, 2025 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package client_test

import (
	"context"
	"io"
	"net/http"
	"net/url"

	"github.com/Carbonfrost/joe-cli-http/httpclient"
	"github.com/Carbonfrost/joe-cli-http/uritemplates"
	phttpclient "github.com/Carbonfrost/pastiche/pkg/client"
	"github.com/Carbonfrost/pastiche/pkg/model"
	"github.com/Carbonfrost/pastiche/pkg/model/modelfakes"

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
				"@example/test": {
					Servers: []*model.Server{
						{
							BaseURL: "https://foo.example/",
						},
					},
					Resource: &model.Resource{
						URITemplate: mustParseURITemplate(""),
						Endpoints: []*model.Endpoint{
							{},
						},
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

		stringTo = func(name string) func(context.Context) string {
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
				phttpclient.NewServiceResolver(exampleModel, specTo("hasNoServers"), stringTo(""), stringTo("")),
				MatchError(`no servers defined for service "hasNoServers"`)),
		)
	})

	Describe("Resolve", func() {

		DescribeTable("examples", func(s string, expected string) {
			r := phttpclient.NewServiceResolver(exampleModel, specTo(s), stringTo(""), stringTo(""))
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
			// Supporting grpc
			// TODO Requires update from joe-cli-http@futures to support correctly
			XEntry("unix", "unix:///tmp/tmp.srKIC1Mk2e", "unix:///tmp/tmp.srKIC1Mk2e"),

			Entry("allow @ in service names", "@example/test", "https://foo.example/"),
		)
	})

})

var _ = Describe("pasticheLocation", func() {

	It("returns an error on no endpoint", func() {
		ctx := phttpclient.NewLocation(
			nil,
			nil,
			nil,
			nil, // no endpoint
			testRequest{},
			nil)

		req, _ := http.NewRequest("GET", "https://example.com", nil)
		mw := ctx.Middleware
		err := mw.Handle(req, nil)

		Expect(err).To(MatchError("no endpoint defined for service/spec"))
	})

	Describe("headers", func() {

		It("copies headers in Middleware", func() {
			ctx := phttpclient.NewLocationVars(
				nil,
				&modelfakes.FakeResolvedResource{
					EndpointStub: func() *model.Endpoint {
						return &model.Endpoint{}
					},
					EvalRequestStub: func(*url.URL, map[string]any) (model.Request, error) {
						return testRequest{
							headers: map[string][]string{
								"X-Header": {"Value"},
							},
						}, nil
					},
				},
			)

			req, _ := http.NewRequest("GET", "https://example.com", nil)
			mw := ctx.Middleware
			_ = mw.Handle(req, nil)

			Expect(req.Header).To(Equal(http.Header{
				"X-Header": {"Value"},
			}))
		})
	})
})

func mustParseURITemplate(t string) *uritemplates.URITemplate {
	u, err := uritemplates.Parse(t)
	if err != nil {
		panic(err)
	}
	return u
}

type testRequest struct {
	headers map[string][]string
}

func (t testRequest) URL() (*url.URL, error) {
	return nil, nil
}
func (t testRequest) Body() io.ReadCloser {
	return nil
}
func (t testRequest) Header() http.Header {
	return t.headers
}
func (t testRequest) Vars() map[string]any {
	return nil
}
