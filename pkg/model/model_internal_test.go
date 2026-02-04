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
