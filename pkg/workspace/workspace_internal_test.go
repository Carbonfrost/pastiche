// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package workspace

import (
	g "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/yaml"

	"github.com/Carbonfrost/pastiche/pkg/model"
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
})
