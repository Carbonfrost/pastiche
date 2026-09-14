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

var _ = g.Describe("describeResults", func() {

	g.DescribeTable("add", func(item model.Item, expected string) {
		var results describeResults
		Expect(results.add(item)).To(Succeed())
		results.Schema = results.fileSchema()

		data, err := yaml.Marshal(&results)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(data)).To(Equal(expected))
	},
		g.Entry("service",
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
		g.Entry("var set",
			&model.VarSet{Name: "creds"},
			`$schema: pastiche:file
varSets:
- $schema: pastiche:varSet
  name: creds
`),
		g.Entry("mixin",
			&model.Mixin{Name: "staging", Method: "POST"},
			`$schema: pastiche:file
mixins:
- $schema: pastiche:mixin
  method: POST
  name: staging
`),
		g.Entry("flow",
			&model.Flow{Name: "login"},
			`$schema: pastiche:file
flows:
- $schema: pastiche:flow
  name: login
`),

		// Resources and endpoints can't be written as a configuration file,
		// hence no schema is claimed for the results
		g.Entry("resource",
			&model.Resource{Name: "widgets"},
			`resources:
- $schema: pastiche:resource
  name: widgets
`),
		g.Entry("endpoint",
			&model.Endpoint{Name: "listWidgets", Method: "GET"},
			`endpoints:
- $schema: pastiche:endpoint
  method: GET
  name: listWidgets
`),
	)

	g.It("reports an error for an item which can't be described", func() {
		var results describeResults
		Expect(results.add(model.Link{})).To(MatchError("cannot describe model.Link"))
	})

	g.Describe("empty", func() {
		g.It("is true when no items were added", func() {
			var results describeResults
			Expect(results.empty()).To(BeTrue())
		})

		g.It("is false when an item was added", func() {
			var results describeResults
			Expect(results.add(&model.Endpoint{Method: "GET"})).To(Succeed())
			Expect(results.empty()).To(BeFalse())
		})
	})

	g.Describe("items", func() {
		g.It("names each item with its kind, sorted by name", func() {
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

		g.It("names an unnamed endpoint by its request method", func() {
			results := describe(&model.Endpoint{Method: "GET"})

			Expect(results.items()).To(Equal([]describeItem{
				{"GET", model.ItemKindEndpoint},
			}))
		})
	})
})

var _ = g.Describe("listItems", func() {

	g.DescribeTable("stylize", func(colorCapable bool, expected string) {
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
		g.Entry("in color", true,
			"\x1b[36m@pastiche/\x1b[0mapi\tservice\n"+
				"@pastiche/meta\tservice\n"+
				"\x1b[36m@quark/\x1b[0mapi\tservice\n"+
				"@quark/meta\tservice\n"+
				"solo\tservice\n"),
		g.Entry("without color", false,
			"@pastiche/api\tservice\n"+
				"@pastiche/meta\tservice\n"+
				"@quark/api\tservice\n"+
				"@quark/meta\tservice\n"+
				"solo\tservice\n"),
	)
})

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
