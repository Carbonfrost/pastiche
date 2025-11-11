// Copyright 2025 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package model

import (
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("newTemplateContent", func() {

	Describe("string body", func() {

		DescribeTable("examples", func(tpl, expected string) {
			c := newTemplateContent(tpl, map[string]any{"found": "found", "v": "var"})

			c.Set("v", "form")

			rendered, _ := io.ReadAll(c.Read())
			Expect(string(rendered)).To(Equal(expected))
		},
			Entry("var fallback literal", `{{ var "notfound" "fallback" }}`, "fallback"),
			Entry("var fallback var", `{{ var "notfound" "found" "fallback" }}`, "found"),
			Entry("form wins over var", `{{ var "v" }}`, `form`),
		)

		DescribeTable("errors", func(tpl, expected string) {
			c := newTemplateContent(tpl, map[string]any{"found": "found", "v": "var"})

			_, err := io.ReadAll(c.Read())
			Expect(err).To(MatchError(ContainSubstring(expected)))
		},
			Entry("bad template", "{{ missing_func }}", `function "missing_func" not defined`),
			Entry("no variable names", "{{ var }}", "var/n requires at least one var name"),
			Entry("missing variable value", `{{ var "notfound" }}`, `var not found: "notfound"`),
		)

	})

	Describe("other body", func() {

		It("expands JSON representation", func() {
			data := map[string]any{
				"query": "GraphQL query",
				"variables": map[string]any{
					"token": "${var.sessionToken}",
					"p":     "${form.v}",
				},
			}

			c := newTemplateContent(data, map[string]any{"sessionToken": "ABC"})
			c.Set("v", "form")

			rendered, _ := io.ReadAll(c.Read())
			Expect(string(rendered)).To(MatchJSON(`{
			"query": "GraphQL query",
			"variables": {
				"token": "ABC",
				"p": "form"
			}
		}`))
		})

	})
})

var _ = Describe("bodyToBytes", func() {

	DescribeTable("examples", func(body any, expected string) {
		c := bodyToBytes(body)
		Expect(string(c)).To(Equal(expected))
	},
		Entry("string", "str", "str"),
		Entry("string literal", `"str"`, `"str"`),
		Entry("map", map[string]any{"a": 3, "b": 1}, `{"a":3,"b":1}`),
		Entry("list", []any{1, "b"}, `[1,"b"]`),
	)

})

var _ = Describe("expandObject", func() {

	DescribeTable("examples", func(body any, expected any) {
		c := expandObject(body, func(s string) string {
			if s == "var.value" {
				return "value"
			}
			return ""
		})
		Expect(c).To(Equal(expected))
	},
		Entry("slice",
			[]any{"d", "${var.value}"},
			[]any{"d", "value"},
		),
		Entry("slice recursion",
			[]any{"d", []any{"${var.value}"}},
			[]any{"d", []any{"value"}},
		),
		Entry("map",
			map[string]any{"d": "${var.value}"},
			map[string]any{"d": "value"},
		),
		Entry("map recursion",
			map[string]any{"d": map[string]any{"e": "${var.value}"}},
			map[string]any{"d": map[string]any{"e": "value"}},
		),
	)

})
