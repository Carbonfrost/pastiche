// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package router

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("filePathToPattern", func() {

	DescribeTable("examples", func(path, expected string) {
		Expect(filePathToPattern(path)).To(Equal(expected))
	},
		Entry("nominal", "/x", "/x"),
		Entry("template at root", "/index.html", "/{$}"),
		Entry("template in dir", "/about/index.html", "/about/{$}"),
		Entry("file at root", "/main.js", "/main.js"),
		Entry("file in dir", "/assets/main.js", "/assets/main.js"),
		Entry("pattern", "/blog/[slug]/index.html", "/blog/{slug}/{$}"),
		Entry("catch all pattern", "/blog/[...slug]/index.html", "/blog/{slug...}"),
	)
})
