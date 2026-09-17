// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package model_test

import (
	"fmt"
	"reflect"

	"github.com/Carbonfrost/pastiche/pkg/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("metadata", func() {

	DescribeTableSubtree("metadata", func(v any) {
		for _, field := range []string{
			"Name",
			"Comment",
			"Title",
			"Description",
			"Tags",
			"Links",
		} {
			It(fmt.Sprintf("contains field %s", field), func() {
				Expect(reflect.ValueOf(v).Elem().FieldByName(field).IsValid()).To(BeTrue())
			})
		}
	},
		Entry("service", new(model.Service)),
		Entry("server", new(model.Server)),
		Entry("resource", new(model.Resource)),
		Entry("endpoint", new(model.Endpoint)),
		Entry("varSet", new(model.VarSet)),
		Entry("mixin", new(model.Mixin)),
		Entry("flow", new(model.Flow)),
	)
})

var _ = Describe("VarSet", func() {

	Describe("Resolve", func() {

		var varSet = &model.VarSet{
			Name: "@example/customers",
			Vars: map[string]map[string]any{
				"loyal": {
					"id":   "ABC",
					"tier": "gold",
				},
				"new": {
					"id": "XYZ",
				},
			},
		}

		DescribeTable("examples",
			func(target, path string, expected any) {
				v, ok := varSet.Resolve(target, path)
				Expect(ok).To(BeTrue())
				Expect(v).To(Equal(expected))
			},
			Entry("full path", "id", "loyal.id", "ABC"),
			Entry("full path, other group", "id", "new.id", "XYZ"),
			Entry("full path, other property", "tier", "loyal.tier", "gold"),
			Entry("group only, implies target name", "id", "loyal", "ABC"),
			Entry("group only, implies target name (tier)", "tier", "loyal", "gold"),
		)

		DescribeTable("errors",
			func(target, path string) {
				_, ok := varSet.Resolve(target, path)
				Expect(ok).To(BeFalse())
			},
			Entry("unknown group", "id", "unknown.id"),
			Entry("unknown property", "unknown", "loyal.unknown"),
			Entry("group only, target name not present", "unknown", "loyal"),
		)
	})
})
