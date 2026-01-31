// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package funcs_test

import (
	"bytes"
	"errors"
	"text/template"

	"github.com/Carbonfrost/pastiche/pkg/template/funcs"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("TermFuncs", func() {

	Describe("style and color printers", func() {

		var render = func(tpl string) (string, error) {
			t := template.Must(template.New("<example>").Parse(tpl))
			data := map[string]any{
				"term": &funcs.TermFuncs{true},

				"Data":       " string ",
				"Int":        420,
				"Items":      []string{"A", "B"},
				"EmptyItems": []string{},
			}

			var results bytes.Buffer
			err := t.Execute(&results, data)
			return results.String(), err
		}

		DescribeTable("examples", func(tpl string, expected types.GomegaMatcher) {
			output, err := render(tpl)
			Expect(err).NotTo(HaveOccurred())
			Expect(output).To(expected)
		},
			Entry("pipe func", "{{ .Data | .term.Bold }}", Equal("\x1b[1m string \x1b[22m")),
			Entry("direct func color", "{{ .term.Red }} {{ .Int }} {{ .term.ResetColor }}", Equal("\x1b[31m 420 \x1b[39m")),
			Entry("direct func style", "{{ .term.Underline }} {{ .Int }} {{ .term.Reset }}", Equal("\x1b[4m 420 \x1b[0m")),
			Entry("empty string", `{{ "" | .term.Underline }}`, Equal("")),
			Entry("Color pipe func", `{{ .Data | .term.Color "Red" }}`, Equal("\x1b[31m string \x1b[39m")),
			Entry("Color direct func", `{{ .term.Color "Red" }} {{ .Int }} {{ .term.ResetColor }}`, Equal("\x1b[31m 420 \x1b[39m")),
			XEntry("Background pipe func", `{{ .Data | .term.Background "Red" }}`, Equal("\x1b[41m string \x1b[49m")),
			XEntry("Background direct func", `{{ .term.Background "Red" }} {{ .Int }} {{ .term.ResetColor }}`, Equal("\x1b[41m 420 \x1b[39m")),
			Entry("Style pipe func", `{{ .Data | .term.Style "Bold" }}`, Equal("\x1b[1m string \x1b[22m")),
			Entry("Style direct func", `{{ .term.Style "Bold" }} {{ .Int }} {{ .term.ResetColor }}`, Equal("\x1b[1m 420 \x1b[39m")),
			Entry("Multiple styles", `{{ .Data | .term.Style "Bold Underline" }}`, Equal("\x1b[1m\x1b[4m string \x1b[24m\x1b[22m")),
			Entry("No styles", `{{ .Data | .term.Style "" }}`, Equal(" string ")),
			Entry("empty style", `{{ .term.Style "" }} Style`, Equal(" Style")),
		)

		DescribeTable("errors", func(tpl string, expected types.GomegaMatcher) {
			_, err := render(tpl)
			Expect(err).To(HaveOccurred())
			Expect(errors.Unwrap(errors.Unwrap(err))).To(expected)
		},
			Entry("invalid color", `{{ .term.Color "unknown" }}`, MatchError("unknown or unsupported color \"unknown\"")),
			XEntry("invalid background", `{{ .term.Background "unknown" }}`, MatchError("unknown or unsupported color \"unknown\"")),
			Entry("invalid style", `{{ .term.Style "unknown" }}`, MatchError("not valid style: \"unknown\"")),
			Entry("invalid styles", `{{ .term.Style "Bold Superscript" }} Style`, MatchError("not valid style: \"Bold Superscript\"")),
		)
	})

})
