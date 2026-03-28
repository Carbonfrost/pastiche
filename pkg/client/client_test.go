// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client_test

import (
	"os"

	"github.com/Carbonfrost/joe-cli-http/uritemplates"
	"github.com/Carbonfrost/pastiche/pkg/client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("VarFromEnvVar", Ordered, func() {

	BeforeAll(func() {
		os.Setenv("USER", "s")
		os.Setenv("PASSWORD", "hunter2")
	})

	AfterAll(func() {
		os.Setenv("USER", "")
		os.Setenv("PASSWORD", "")
	})

	DescribeTable("examples", func(v *uritemplates.Var, expected *uritemplates.Var) {
		actual := client.VarFromEnv(v)
		Expect(actual).To(Equal(expected))
	},
		Entry(
			"nominal",
			uritemplates.StringVar("x", "PASSWORD"),
			uritemplates.StringVar("x", "hunter2"),
		),
		Entry(
			"array",
			uritemplates.ArrayVar("x", "USER", "PASSWORD"),
			uritemplates.ArrayVar("x", "s", "hunter2"),
		),
		Entry(
			"map",
			uritemplates.MapVar("x", map[string]any{"p": "PASSWORD", "m": "MISSING_ENV_VAR"}),
			uritemplates.MapVar("x", map[string]any{"p": "hunter2", "m": ""}),
		),
	)

})
