// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package pastiche_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"

	cli "github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/pastiche/pkg/cmd/pastiche"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("Integration", Label("integration"), Ordered, func() {

	Describe("integration", func() {

		Context("when simple static", func() {

			var (
				testServer *httptest.Server
				baseURL    string
			)

			BeforeAll(func() {
				testServer = httptest.NewServer(
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						fmt.Fprint(w, "Hello, world!")
					}),
				)
				baseURL = testServer.URL
			})

			AfterAll(func() {
				testServer.Close()
			})

			DescribeTable("examples",
				func(arguments func() string, expected types.GomegaMatcher) {
					var capture bytes.Buffer
					app := pastiche.NewApp()
					app.Stdout = &capture

					args, _ := cli.Split(arguments())
					err := app.RunContext(context.Background(), args)
					Expect(err).NotTo(HaveOccurred())
					Expect(capture.String()).To(expected)
				},
				Entry(
					"pastiche <base>",
					func() string {
						return fmt.Sprintf("pastiche %v", baseURL)
					},
					Equal("Hello, world!")),
				Entry(
					"pastiche fetch <base>",
					func() string {
						return fmt.Sprintf("pastiche fetch %v", baseURL)
					},
					Equal("Hello, world!")),
			)
		})
	})
})
