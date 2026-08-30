// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package model_test

import (
	"net/http"
	"net/url"

	"github.com/Carbonfrost/pastiche/pkg/model"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Values", func() {
	Describe("ToHeader", func() {
		It("converts into net/http.Header and omits optional empties", func() {
			v := model.Values{
				{Name: "X-Trace", Value: "abc"},
				{Name: "X-Tag", Values: []string{"a", "b"}},
				{Name: "X-Skip", Optional: true},
			}
			Expect(v.ToHeader()).To(Equal(http.Header{
				"X-Trace": {"abc"},
				"X-Tag":   {"a", "b"},
			}))
		})
	})

	Describe("ToURLValues", func() {
		It("converts into url.Values", func() {
			v := model.Values{{Name: "user_id", Value: "1"}}
			Expect(v.ToURLValues()).To(Equal(url.Values{"user_id": {"1"}}))
		})
	})

	Describe("Reduce", func() {
		It("replaces by default", func() {
			base := model.Values{{Name: "scope", Value: "read"}}
			next := model.Values{{Name: "scope", Value: "write"}}
			Expect(base.Reduce(next)).To(Equal(model.Values{{Name: "scope", Value: "write"}}))
		})

		It("appends when the incoming value uses MergeAppend", func() {
			base := model.Values{{Name: "scope", Value: "read"}}
			next := model.Values{{Name: "scope", Value: "write", Merge: model.MergeAppend}}
			Expect(base.Reduce(next)).To(Equal(model.Values{
				{Name: "scope", Values: []string{"read", "write"}, Merge: model.MergeAppend},
			}))
		})

		It("adds values with new names", func() {
			base := model.Values{{Name: "a", Value: "1"}}
			next := model.Values{{Name: "b", Value: "2"}}
			Expect(base.Reduce(next)).To(Equal(model.Values{
				{Name: "a", Value: "1"},
				{Name: "b", Value: "2"},
			}))
		})
	})
})
