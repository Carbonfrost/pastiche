// Copyright 2023 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package model_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

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

var _ = Describe("ResolvedReference", func() {

	Describe("URL", func() {

		DescribeTable("examples", func(spec []string, vars map[string]any, expected string) {
			subject := model.New(&config.Config{
				Services: []config.Service{
					config.ExampleHTTPBinorg,
				},
			})

			merged, err := subject.Resolve(spec, "", "")
			Expect(err).NotTo(HaveOccurred())

			url, err := merged.URL(nil, vars)
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

		Expect(subject.Services["s"].Resource.Resources[0]).To(match)
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
