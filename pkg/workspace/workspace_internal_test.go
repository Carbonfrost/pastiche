// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package workspace

import (
	"bytes"

	cli "github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/pastiche/pkg/model"
	g "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/yaml"
)

var (
	DescribeTable = g.DescribeTable
	Entry         = g.Entry
	It            = g.It
)

var _ = g.Describe("describeResults", func() {

	DescribeTable("add", func(item model.Item, expected string) {
		var results describeResults
		Expect(results.add(item)).To(Succeed())
		results.Schema = results.fileSchema()

		data, err := yaml.Marshal(&results)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(data)).To(Equal(expected))
	},
		Entry("service",
			&model.Service{
				Name:     "demo",
				Resource: &model.Resource{},
			},
			`$schema: pastiche:file
services:
- $schema: pastiche:service
  name: demo
  resources:
  - $schema: pastiche:resource
`),
		Entry("var set",
			&model.VarSet{Name: "creds"},
			`$schema: pastiche:file
varSets:
- $schema: pastiche:varSet
  name: creds
`),
		Entry("mixin",
			&model.Mixin{Name: "staging", Method: "POST"},
			`$schema: pastiche:file
mixins:
- $schema: pastiche:mixin
  method: POST
  name: staging
`),
		Entry("flow",
			&model.Flow{Name: "login"},
			`$schema: pastiche:file
flows:
- $schema: pastiche:flow
  name: login
`),

		// Resources and endpoints can't be written as a configuration file,
		// hence no schema is claimed for the results
		Entry("resource",
			&model.Resource{Name: "widgets"},
			`resources:
- $schema: pastiche:resource
  name: widgets
`),
		Entry("endpoint",
			&model.Endpoint{Name: "listWidgets", Method: "GET"},
			`endpoints:
- $schema: pastiche:endpoint
  method: GET
  name: listWidgets
`),
	)

	It("reports an error for an item which can't be described", func() {
		var results describeResults
		Expect(results.add(model.Link{})).To(MatchError("cannot describe model.Link"))
	})

	g.Describe("empty", func() {
		It("is true when no items were added", func() {
			var results describeResults
			Expect(results.empty()).To(BeTrue())
		})

		It("is false when an item was added", func() {
			var results describeResults
			Expect(results.add(&model.Endpoint{Method: "GET"})).To(Succeed())
			Expect(results.empty()).To(BeFalse())
		})
	})

	g.Describe("items", func() {
		It("names each item with its kind, sorted by name", func() {
			results := describe(
				&model.Service{Name: "beta", Resource: &model.Resource{}},
				&model.Resource{Name: "alpha"},
				&model.Flow{Name: "login"},
				&model.VarSet{Name: "creds"},
			)

			Expect(results.items()).To(Equal([]describeItem{
				{"alpha", model.ItemKindResource},
				{"beta", model.ItemKindService},
				{"creds", model.ItemKindVarSet},
				{"login", model.ItemKindFlow},
			}))
		})

		It("names an unnamed endpoint by its request method", func() {
			results := describe(&model.Endpoint{Method: "GET"})

			Expect(results.items()).To(Equal([]describeItem{
				{"GET", model.ItemKindEndpoint},
			}))
		})
	})
})

var _ = g.Describe("listItems", func() {

	DescribeTable("stylize", func(colorCapable bool, expected string) {
		results := describe(
			service("@pastiche/api"),
			service("@pastiche/meta"),
			service("@quark/api"),
			service("@quark/meta"),
			service("solo"),
		)

		var buf bytes.Buffer
		out := cli.NewWriter(&buf)
		out.SetColorCapable(colorCapable)

		Expect(listItems(out, results)).To(Succeed())
		Expect(buf.String()).To(Equal(expected))
	},
		// Only the first name in each run of names which share a package
		// prefix is stylized
		Entry("in color", true,
			"\x1b[36m@pastiche/\x1b[0mapi\tservice\n"+
				"@pastiche/meta\tservice\n"+
				"\x1b[36m@quark/\x1b[0mapi\tservice\n"+
				"@quark/meta\tservice\n"+
				"solo\tservice\n"),
		Entry("without color", false,
			"@pastiche/api\tservice\n"+
				"@pastiche/meta\tservice\n"+
				"@quark/api\tservice\n"+
				"@quark/meta\tservice\n"+
				"solo\tservice\n"),
	)
})

var _ = g.Describe("treeItems", func() {

	It("nests the resources of a service and endpoints of a resource", func() {
		results := describe(&model.Service{
			Name: "demo",
			Resource: &model.Resource{
				Endpoints: []*model.Endpoint{{Method: "GET"}},
				Resources: []*model.Resource{
					{
						Name: "widgets",
						Endpoints: []*model.Endpoint{
							{Method: "GET"},
							{Method: "POST", Name: "createWidget"},
						},
						Resources: []*model.Resource{
							{Name: "parts", Endpoints: []*model.Endpoint{{Method: "GET"}}},
						},
					},
					{
						Name:      "gadgets",
						Endpoints: []*model.Endpoint{{Method: "DELETE"}},
					},
				},
			},
		})
		Expect(render(results, false)).To(Equal(
			`demo
├── GET
├── gadgets    DELETE
└── widgets    GET • POST (createWidget)
    └── parts    GET
`))
	})

	// The children of a root don't interrupt the run of package prefixes
	// which the roots themselves form
	It("stylizes the package prefix of the roots", func() {
		results := describe(
			serviceWithEndpoint("@pastiche/api"),
			serviceWithEndpoint("@pastiche/meta"),
		)

		Expect(render(results, true)).To(Equal(
			"\x1b[36m@pastiche/\x1b[0mapi\n└── GET\n@pastiche/meta\n└── GET\n"))
	})
})

func render(results *describeResults, colorCapable bool) string {
	var buf bytes.Buffer
	out := cli.NewWriter(&buf)
	out.SetColorCapable(colorCapable)

	Expect(treeItems(out, results)).To(Succeed())
	return buf.String()
}

func serviceWithEndpoint(name string) *model.Service {
	return &model.Service{
		Name:     name,
		Resource: &model.Resource{Endpoints: []*model.Endpoint{{Method: "GET"}}},
	}
}

func describe(items ...model.Item) *describeResults {
	var results describeResults
	for _, item := range items {
		Expect(results.add(item)).To(Succeed())
	}
	return &results
}

func service(name string) *model.Service {
	return &model.Service{Name: name, Resource: &model.Resource{}}
}
