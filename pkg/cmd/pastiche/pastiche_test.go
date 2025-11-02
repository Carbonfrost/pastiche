// Copyright 2023 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package pastiche_test

import (
	"context"
	"io"

	cli "github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/pastiche/pkg/cmd/pastiche"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("NewApp", func() {

	Describe("integration", func() {

		DescribeTable("errors",
			func(arguments string, errExpected types.GomegaMatcher) {
				app := pastiche.NewApp()
				app.Stdout = io.Discard

				args, _ := cli.Split(arguments)
				err := app.RunContext(context.Background(), args)
				Expect(err).To(errExpected)
			},
			Entry(
				"disallow persistent HTTP flags",
				"pastiche describe service --interface en0",
				MatchError("unknown option interface")), // TODO A better error message should be generated
		)
	})
})
