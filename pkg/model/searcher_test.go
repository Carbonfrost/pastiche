// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package model_test

import (
	"fmt"

	"github.com/Carbonfrost/pastiche/pkg/config"
	"github.com/Carbonfrost/pastiche/pkg/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func searchModel() *model.Model {
	public := config.Metadata{Tags: []string{"public"}}

	return model.New(&config.File{
		Services: []config.Service{
			{
				Name: "alpha",
				Tags: []string{"public", "v1"},
				Resources: []config.Resource{
					{
						Name:     "widgets",
						Metadata: public,
						Get:      &config.Endpoint{Name: "getWidgets", Metadata: public},
						Post:     &config.Endpoint{Name: "postWidgets"},
						Resources: []config.Resource{
							{
								Name: "parts",
								Get:  &config.Endpoint{Name: "getParts"},
							},
						},
					},
					{
						Name:   "gadgets",
						Delete: &config.Endpoint{Name: "deleteGadgets"},
					},
				},
			},
			{
				Name:     "beta",
				Metadata: public,
			},
		},
		VarSets: []config.VarSet{
			{Name: "creds", Metadata: public},
			{Name: "other"},
		},
		Flows: []config.Flow{
			{Name: "login", Metadata: public},
			{Name: "logout"},
		},
	})
}

func spec(names ...string) *model.ServiceSpec {
	s := model.ServiceSpec(names)
	return &s
}

func itemNames(s model.Searcher) []string {
	res, err := s.Results()
	Expect(err).NotTo(HaveOccurred())

	names := []string{}
	for item := range res {
		switch it := item.(type) {
		case *model.Service:
			names = append(names, "service:"+it.Name)
		case *model.VarSet:
			names = append(names, "varSet:"+it.Name)
		case *model.Flow:
			names = append(names, "flow:"+it.Name)
		case *model.Resource:
			names = append(names, "resource:"+it.Name)
		case *model.Endpoint:
			names = append(names, "endpoint:"+it.Name)
		default:
			names = append(names, fmt.Sprintf("%T", item))
		}
	}
	return names
}

var _ = Describe("Search", func() {

	DescribeTable("results", func(criteria *model.SearchCriteria, expected []string) {
		Expect(itemNames(searchModel().Search(criteria))).To(Equal(expected))
	},
		Entry("nil criteria implies all services",
			nil,
			[]string{"service:alpha", "service:beta"},
		),
		Entry("all services",
			&model.SearchCriteria{},
			[]string{"service:alpha", "service:beta"},
		),
		Entry("service by name",
			&model.SearchCriteria{Spec: spec("alpha")},
			[]string{"service:alpha"},
		),
		Entry("services filtered by tag",
			&model.SearchCriteria{IncludeTags: []string{"v1"}},
			[]string{"service:alpha"},
		),
		Entry("services must have all tags",
			&model.SearchCriteria{IncludeTags: []string{"public", "v1"}},
			[]string{"service:alpha"},
		),
		Entry("empty spec is the same as no spec",
			&model.SearchCriteria{Spec: spec()},
			[]string{"service:alpha", "service:beta"},
		),

		Entry("all var sets",
			&model.SearchCriteria{Kind: model.ItemKindVarSet},
			[]string{"varSet:creds", "varSet:other"},
		),
		Entry("var set by name",
			&model.SearchCriteria{Kind: model.ItemKindVarSet, Spec: spec("other")},
			[]string{"varSet:other"},
		),
		Entry("var sets filtered by tag",
			&model.SearchCriteria{Kind: model.ItemKindVarSet, IncludeTags: []string{"public"}},
			[]string{"varSet:creds"},
		),

		Entry("all flows",
			&model.SearchCriteria{Kind: model.ItemKindFlow},
			[]string{"flow:login", "flow:logout"},
		),
		Entry("flows filtered by tag",
			&model.SearchCriteria{Kind: model.ItemKindFlow, IncludeTags: []string{"public"}},
			[]string{"flow:login"},
		),

		Entry("all resources",
			&model.SearchCriteria{Kind: model.ItemKindResource},
			[]string{"resource:widgets", "resource:parts", "resource:gadgets"},
		),
		Entry("resources within a service",
			&model.SearchCriteria{Kind: model.ItemKindResource, Spec: spec("alpha")},
			[]string{"resource:widgets", "resource:parts", "resource:gadgets"},
		),
		Entry("resource by qualified name",
			&model.SearchCriteria{Kind: model.ItemKindResource, Spec: spec("alpha", "widgets")},
			[]string{"resource:widgets"},
		),
		Entry("nested resource by qualified name",
			&model.SearchCriteria{Kind: model.ItemKindResource, Spec: spec("alpha", "widgets", "parts")},
			[]string{"resource:parts"},
		),
		Entry("resources filtered by tag",
			&model.SearchCriteria{Kind: model.ItemKindResource, IncludeTags: []string{"public"}},
			[]string{"resource:widgets"},
		),

		Entry("all endpoints",
			&model.SearchCriteria{Kind: model.ItemKindEndpoint, Spec: spec("alpha")},
			[]string{"endpoint:", "endpoint:getWidgets", "endpoint:postWidgets", "endpoint:getParts", "endpoint:deleteGadgets"},
		),
		Entry("endpoints filtered by method",
			&model.SearchCriteria{Kind: model.ItemKindEndpoint, Spec: spec("alpha"), Method: "post"},
			[]string{"endpoint:postWidgets"},
		),
		Entry("endpoints of the resource named by a qualified name",
			&model.SearchCriteria{Kind: model.ItemKindEndpoint, Spec: spec("alpha", "widgets")},
			[]string{"endpoint:getWidgets", "endpoint:postWidgets"},
		),
		Entry("endpoints filtered by tag",
			&model.SearchCriteria{Kind: model.ItemKindEndpoint, Spec: spec("alpha"), IncludeTags: []string{"public"}},
			[]string{"endpoint:getWidgets"},
		),

		Entry("all items",
			&model.SearchCriteria{Kind: model.ItemKindAll},
			[]string{
				"service:alpha", "service:beta",
				"varSet:creds", "varSet:other",
				"flow:login", "flow:logout",
				"resource:widgets", "resource:parts", "resource:gadgets",
				"endpoint:", "endpoint:getWidgets", "endpoint:postWidgets", "endpoint:getParts",
				"endpoint:deleteGadgets", "endpoint:",
			},
		),
		Entry("all items ignores searchers which the spec can't satisfy",
			&model.SearchCriteria{Kind: model.ItemKindAll, Spec: spec("alpha", "widgets")},
			[]string{
				"resource:widgets",
				"endpoint:getWidgets", "endpoint:postWidgets",
			},
		),
	)

	DescribeTable("errors", func(criteria *model.SearchCriteria, expected string) {
		_, err := searchModel().Search(criteria).Results()
		Expect(err).To(MatchError(expected))
	},
		Entry("unknown service",
			&model.SearchCriteria{Spec: spec("unknown")},
			`service not found: "unknown"`,
		),
		Entry("qualified name for a service",
			&model.SearchCriteria{Spec: spec("alpha", "widgets")},
			`service cannot be named by a qualified name: "alpha.widgets"`,
		),
		Entry("unknown var set",
			&model.SearchCriteria{Kind: model.ItemKindVarSet, Spec: spec("unknown")},
			`var set not found: "unknown"`,
		),
		Entry("qualified name for a var set",
			&model.SearchCriteria{Kind: model.ItemKindVarSet, Spec: spec("creds", "extra")},
			`var set cannot be named by a qualified name: "creds.extra"`,
		),
		Entry("unknown flow",
			&model.SearchCriteria{Kind: model.ItemKindFlow, Spec: spec("unknown")},
			`flow not found: "unknown"`,
		),
		Entry("unknown resource",
			&model.SearchCriteria{Kind: model.ItemKindResource, Spec: spec("alpha", "unknown")},
			`resource not found: "alpha.unknown"`,
		),
		Entry("unknown nested resource",
			&model.SearchCriteria{Kind: model.ItemKindResource, Spec: spec("alpha", "widgets", "unknown")},
			`resource not found: "alpha.widgets.unknown"`,
		),
		Entry("unknown service when searching endpoints",
			&model.SearchCriteria{Kind: model.ItemKindEndpoint, Spec: spec("unknown")},
			`service not found: "unknown"`,
		),
		Entry("all items when nothing can satisfy the spec",
			&model.SearchCriteria{Kind: model.ItemKindAll, Spec: spec("unknown")},
			`service not found: "unknown"`,
		),
	)

})
