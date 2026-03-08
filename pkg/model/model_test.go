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
	)
})
