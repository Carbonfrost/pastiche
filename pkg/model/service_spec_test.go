// Copyright 2023, 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package model_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/pastiche/pkg/model"
	"github.com/onsi/gomega/types"
)

var _ = Describe("ServiceSpecCounter", func() {

	Describe("Take", func() {

		DescribeTable("examples", func(spec []string, shouldTake int, expectErr types.GomegaMatcher) {
			subject := new(model.ServiceSpec).NewCounter()

			for index, s := range spec {
				err := subject.Take(s, false)
				if index < shouldTake {
					Expect(err).NotTo(HaveOccurred())
					continue
				}

				Expect(err).To(expectErr)
			}
		},
			Entry("nominal",
				[]string{"homebrew", "formula"},
				2,
				MatchError(cli.EndOfArguments),
			),
			Entry("stop on template variables",
				[]string{"homebrew", "formula", "formula=wget"},
				2,
				MatchError(cli.EndOfArguments),
			),
		)
	})
})

var _ = Describe("ServiceSpec", func() {

	Describe("Path", func() {

		DescribeTable("examples", func(spec []string, expected string) {
			subject := model.ServiceSpec(spec)
			Expect(subject.Path()).To(Equal(expected))
		},
			Entry("nominal",
				[]string{"homebrew", "formula"},
				"homebrew/formula",
			),
			Entry("split with dots",
				[]string{"@homebrew/formula", "keg"},
				"@homebrew/formula.keg",
			),
		)
	})
})
