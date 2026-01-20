// Copyright 2023 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package model_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/Carbonfrost/joe-cli-http/uritemplates"
	"github.com/Carbonfrost/pastiche/pkg/config"
	"github.com/Carbonfrost/pastiche/pkg/model"
	. "github.com/onsi/gomega/gstruct"
	"github.com/onsi/gomega/types"
)

type resolveResultMatches struct {
	Service, Resource types.GomegaMatcher
}

var _ = Describe("Resolve", func() {

	DescribeTable("examples", func(spec []string, match resolveResultMatches) {
		subject := model.New(&config.Config{
			Services: []config.Service{
				config.ExampleHTTPBinorg,
			},
		})

		merged, err := subject.Resolve(spec, "", "")
		Expect(err).NotTo(HaveOccurred())
		Expect(merged.Service()).To(match.Service)
		Expect(merged.Resource()).To(match.Resource)
	},
		Entry("simple",
			[]string{"httpbin", "get"},
			resolveResultMatches{
				Service:  PointTo(MatchFields(IgnoreExtras, Fields{"Name": Equal("httpbin")})),
				Resource: PointTo(MatchFields(IgnoreExtras, Fields{"Name": Equal("get")})),
			},
		),
		Entry("nested resource",
			[]string{"httpbin", "status", "codes"},
			resolveResultMatches{
				Service:  PointTo(MatchFields(IgnoreExtras, Fields{"Name": Equal("httpbin")})),
				Resource: PointTo(MatchFields(IgnoreExtras, Fields{"Name": Equal("codes")})),
			},
		),
	)
})

var _ = Describe("Header", func() {

	DescribeTable("examples", func(expected types.GomegaMatcher) {
		subject := model.New(&config.Config{
			Services: []config.Service{
				{
					Name: "a",
					Servers: []config.Server{
						{
							Name:    "default",
							Headers: config.Header{"X-From-Server": []string{"server"}},
						},
					},
					Resources: []config.Resource{
						{
							Name:    "b",
							Headers: config.Header{"X-Nested": []string{"b"}},
							Resources: []config.Resource{
								{
									Name:    "c",
									Headers: config.Header{"X-From-Resource": []string{"resource"}},
									Get: &config.Endpoint{
										Headers: config.Header{"X-From-Endpoint": []string{"endpoint"}},
									},
								},
							},
						},
					},
				},
			},
		})

		spec := []string{"a", "b", "c"}
		rr, _ := subject.Resolve(spec, "default", "")
		merged, _ := rr.EvalRequest(nil, nil)
		Expect(merged.Headers()).To(expected)
	},
		Entry("copied from resource", HaveKeyWithValue("X-From-Resource", []string{"resource"})),
		Entry("copied from endpoint", HaveKeyWithValue("X-From-Endpoint", []string{"endpoint"})),
		Entry("copied from server", HaveKeyWithValue("X-From-Server", []string{"server"})),
		Entry("copied from nested", HaveKeyWithValue("X-Nested", []string{"b"})),
	)

	It("expands variables", func() {
		subject := model.New(&config.Config{
			Services: []config.Service{
				{
					Name: "a",
					Servers: []config.Server{
						{
							Name: "default",
							Headers: config.Header{
								"Test": []string{"${var}"},
								"S":    []string{"${varServer}"},
								"R":    []string{"${varResource}"},
							},
							Vars: map[string]any{
								"varServer": "endpoint value from S var set",
							},
						},
					},
					Resources: []config.Resource{
						{
							Name: "b",
							Vars: map[string]any{
								"varResource": "endpoint value from R var set",
							},
						},
					},
				},
			},
		})

		spec := []string{"a", "b"}

		rr, _ := subject.Resolve(spec, "default", "")
		merged, _ := rr.EvalRequest(nil, uritemplates.Vars{
			"var": "endpoint value from var",
		})
		Expect(merged.Headers()).To(HaveKeyWithValue("Test", []string{"endpoint value from var"}))
		Expect(merged.Headers()).To(HaveKeyWithValue("S", []string{"endpoint value from S var set"}))
		Expect(merged.Headers()).To(HaveKeyWithValue("R", []string{"endpoint value from R var set"}))
	})
})

var _ = Describe("ResolvedReference", func() {

	Describe("URL", func() {

		DescribeTable("examples", func(spec []string, vars map[string]any, expected string) {
			subject := model.New(&config.Config{
				Services: []config.Service{
					config.ExampleHTTPBinorg,
					{
						Name: "a",
						Servers: []config.Server{
							{
								Vars: map[string]any{
									"baseURL": "http://server.example",
								},
							},
						},
						Resources: []config.Resource{
							{
								Name: "get",
								URI:  "{+baseURL}/get",
							},
						},
					},
					{
						Name: "b",
						Servers: []config.Server{
							{
								BaseURL: "{+baseURL}",
								Vars: map[string]any{
									"baseURL": "http://var.example:8080",
								},
							},
						},
						Resources: []config.Resource{
							{
								Name: "make",
								URI:  "make",
							},
						},
					},
				},
			})

			rr, err := subject.Resolve(spec, "", "")
			Expect(err).NotTo(HaveOccurred())

			merged, _ := rr.EvalRequest(nil, vars)
			url, err := merged.URL()
			Expect(err).NotTo(HaveOccurred())

			Expect(url.String()).To(Equal(expected))
		},
			Entry("simple",
				[]string{"httpbin", "get"},
				nil,
				"https://httpbin.org/get",
			),
			Entry("nested resource",
				[]string{"httpbin", "status", "codes"},
				map[string]any{"codes": 200},
				"https://httpbin.org/status/200",
			),
			Entry("using vars",
				[]string{"a", "get"},
				nil,
				"http://server.example/get",
			),
			Entry("baseURL also using vars",
				[]string{"b", "make"},
				nil,
				"http://var.example:8080/make",
			),
		)

	})
})

var _ = Describe("New", func() {

	DescribeTable("resource binding", func(r config.Resource, match types.GomegaMatcher) {
		subject := model.New(&config.Config{
			Services: []config.Service{
				{
					Name:      "s",
					Resources: []config.Resource{r},
				},
			},
		})

		s, _ := subject.Service("s")
		Expect(s.Resource.Resources[0]).To(match)
	},
		Entry("assume GET when no endpoint is defined",
			config.Resource{
				Name: "/",
			},
			PointTo(MatchFields(IgnoreExtras,
				Fields{
					"Endpoints": MatchElementsWithIndex(IndexIdentity, IgnoreExtras,
						Elements{
							"0": PointTo(MatchFields(IgnoreExtras, Fields{"Method": Equal("GET")})),
						}),
				})),
		),
	)
})
