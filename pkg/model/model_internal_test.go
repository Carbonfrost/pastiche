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
