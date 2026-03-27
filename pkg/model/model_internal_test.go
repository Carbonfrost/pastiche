// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package model

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("resolveURL", func() {

	DescribeTable("examples", func(base string, prefix []string, expected string) {
		vars := map[string]any{
			"b": "ar",
		}
		u, _ := resolveURL(base, prefix, vars)
		Expect(u.String()).To(Equal(expected))
	},
		Entry(
			"nominal",
			"https://example.com", []string{"a", "{b}"}, "https://example.com/a/ar",
		),
		Entry(
			"qualified prefix",
			"https://example.com", []string{"https://foo.example", "b"}, "https://foo.example/b",
		),
		Entry(
			"qualified prefix vars",
			"https://example.com", []string{"https://foo.example/{b}"}, "https://foo.example/ar",
		),
	)

})

var _ = Describe("reduceAuth", func() {
	DescribeTable("examples", func(x, y, expected Auth) {
		Expect(reduceAuth(x, y)).To(Equal(expected))
	},
		Entry(
			"basic: override",
			&BasicAuth{User: "U", Password: "P"},
			&BasicAuth{User: "V", Password: "Q"},
			&BasicAuth{User: "V", Password: "Q"},
		),
		Entry(
			"basic: merge",
			&BasicAuth{User: "U", Password: "P"},
			&BasicAuth{},
			&BasicAuth{User: "U", Password: "P"},
		),
	)
})

var _ = Describe("reduceValues", func() {

	DescribeTable("examples", func(x, y, expected Values) {
		Expect(reduceValues(x, y)).To(Equal(expected))
	},
		Entry(
			"empty",
			Values{},
			Values{{Name: "A", Value: "A"}},
			Values{{Name: "A", Value: "A"}},
		),
		Entry(
			"nominal",
			Values{{Name: "A", Value: "A"}},
			Values{{Name: "B", Value: "B"}},
			Values{{Name: "A", Value: "A"}, {Name: "B", Value: "B"}},
		),
		Entry(
			"overwrite (replace is the default)",
			Values{{Name: "A", Value: "1"}},
			Values{{Name: "A", Value: "2"}},
			Values{{Name: "A", Value: "2"}},
		),
		Entry(
			"append",
			Values{{Name: "A", Value: "1"}},
			Values{{Name: "A", Value: "2", Merge: MergeAppend}},
			Values{{Name: "A", Values: []string{"1", "2"}, Merge: MergeAppend}},
		),
	)
})

var _ = Describe("reduceParams", func() {

	DescribeTable("examples", func(x, y, expected []*Param) {
		Expect(reduceParams(x, y)).To(Equal(expected))
	},
		Entry(
			"empty",
			[]*Param{},
			[]*Param{{Name: "a", Title: "A"}},
			[]*Param{{Name: "a", Title: "A"}},
		),
		Entry(
			"aggregate by unique names",
			[]*Param{{Name: "a", Title: "A"}},
			[]*Param{{Name: "b", Title: "B"}},
			[]*Param{{Name: "a", Title: "A"}, {Name: "b", Title: "B"}},
		),
		Entry(
			"do not merge metadata for duplicate names",
			[]*Param{{Name: "a", Title: "First", Description: "First desc"}},
			[]*Param{{Name: "a", Title: "Second", Description: "Second desc"}},
			[]*Param{{Name: "a", Title: "First", Description: "First desc"}},
		),
		Entry(
			"keep first param and ignore subsequent duplicates",
			[]*Param{{Name: "a", Title: "A", Tags: []string{"tag1"}}},
			[]*Param{{Name: "a", Title: "B", Tags: []string{"tag2"}}, {Name: "c", Title: "C"}},
			[]*Param{{Name: "a", Title: "A", Tags: []string{"tag1"}}, {Name: "c", Title: "C"}},
		),
		Entry(
			"multiple params with some duplicates",
			[]*Param{{Name: "x", Title: "X"}, {Name: "y", Title: "Y"}},
			[]*Param{{Name: "y", Title: "Y2"}, {Name: "z", Title: "Z"}},
			[]*Param{{Name: "x", Title: "X"}, {Name: "y", Title: "Y"}, {Name: "z", Title: "Z"}},
		),
	)
})
