// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package model_test

import (
	"fmt"
	"net/http"
	"net/url"
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/Carbonfrost/pastiche/pkg/model"
	"github.com/Carbonfrost/pastiche/pkg/model/modelfakes"
	"github.com/onsi/gomega/types"
)

var _ = Describe("NewRequest", func() {

	Context("Headers", func() {

		DescribeTable("examples", func(expected types.GomegaMatcher) {
			resource := new(modelfakes.FakeResolvedResource)
			resource.ServiceReturns(&model.Service{})
			resource.ServerReturns(&model.Server{
				Headers: newHeader("T", "server"),
			})
			resource.EndpointReturns(&model.Endpoint{
				Headers: newHeader("S", "endpoint", "E", "endpoint", "T", "X"),
			})
			resource.LineageReturns([]*model.Resource{
				{
					Headers: newHeader("S", "parent", "U", "lineage2", "W", "lineage2", "T", "X"),
				},
				{
					Headers: newHeader("S", "child", "V", "lineage1", "W", "lineage1", "T", "X"),
				},
			})

			req, err := model.NewRequest(resource, model.WithBaseURL(mustParseURL("https://example.com")))
			Expect(err).NotTo(HaveOccurred())
			Expect(req.Headers).To(expected)
		},
			Entry(
				"reduce by lineage",
				And(
					HaveKeyWithValue("U", []string{"lineage2"}),
					HaveKeyWithValue("V", []string{"lineage1"}),
					HaveKeyWithValue("W", []string{"lineage1"}),
				),
			),
			Entry(
				"narrowest child value",
				HaveKeyWithValue("S", []string{"endpoint"}),
			),
			Entry(
				"server overrides everything",
				HaveKeyWithValue("T", []string{"server"}),
			),
		)
	})

	Context("Query", func() {

		It("merges into the result", func() {
			resource := new(modelfakes.FakeResolvedResource)
			resource.EndpointReturns(&model.Endpoint{
				Query: newHeader("hello", "world", "in", "${var.s}"),
			})
			resource.LineageReturns([]*model.Resource{
				{
					URITemplate: mustParseURITemplate("/{?x}"),
				},
			})

			result, _ := model.NewRequest(
				resource,
				model.WithBaseURL(mustParseURL("https://localhost:8080")),
				model.WithVars(map[string]any{"s": "to", "x": "y"}),
			)
			Expect(result.URL.String()).To(Equal("https://localhost:8080/?hello=world&in=to&x=y"))
		})

		DescribeTable("examples", func(expected types.GomegaMatcher) {
			resource := new(modelfakes.FakeResolvedResource)
			resource.ServiceReturns(&model.Service{})
			resource.ServerReturns(&model.Server{
				Query: newHeader("T", "server"),
			})
			resource.EndpointReturns(&model.Endpoint{
				Query: newHeader("S", "endpoint", "E", "endpoint", "T", "X"),
			})
			resource.LineageReturns([]*model.Resource{
				{
					Query: newHeader("S", "parent", "U", "lineage2", "W", "lineage2", "T", "X"),
				},
				{
					Query: newHeader("S", "child", "V", "lineage1", "W", "lineage1", "T", "X"),
				},
			})

			req, err := model.NewRequest(resource, model.WithBaseURL(mustParseURL("https://example.com")))
			Expect(err).NotTo(HaveOccurred())
			Expect(req.URL.Query()).To(expected)
		},
			Entry(
				"reduce by lineage",
				And(
					HaveKeyWithValue("U", []string{"lineage2"}),
					HaveKeyWithValue("V", []string{"lineage1"}),
					HaveKeyWithValue("W", []string{"lineage1"}),
				),
			),
			Entry(
				"narrowest child value",
				HaveKeyWithValue("S", []string{"endpoint"}),
			),
			Entry(
				"server overrides everything",
				HaveKeyWithValue("T", []string{"server"}),
			),
		)
	})
})

func newHeader(namevalues ...string) http.Header {
	if len(namevalues)%2 != 0 {
		panic(fmt.Errorf("requires even number of arguments, got %d", len(namevalues)))
	}
	m := make(map[string][]string, len(namevalues)/2)
	for kvp := range slices.Chunk(namevalues, 2) {
		key := kvp[0]
		m[key] = []string{kvp[1]}
	}

	return m
}

func mustParseURL(t string) *url.URL {
	u, err := url.Parse(t)
	if err != nil {
		panic(err)
	}
	return u
}
