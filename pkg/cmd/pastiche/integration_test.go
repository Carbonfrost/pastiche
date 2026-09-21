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
	"net/url"
	"os"
	"path/filepath"

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
					err := app.RunContext(context.Background(), args...)
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

		Context("when using --context-param", func() {

			var (
				testServer    *httptest.Server
				receivedQuery string
			)

			BeforeEach(func() {
				testServer = httptest.NewServer(
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						receivedQuery = r.URL.Query().Get("id")
						fmt.Fprint(w, "ok")
					}),
				)

				tmpDir := GinkgoT().TempDir()
				origDir, err := os.Getwd()
				Expect(err).NotTo(HaveOccurred())
				DeferCleanup(func() { _ = os.Chdir(origDir) })

				Expect(os.MkdirAll(filepath.Join(tmpDir, ".pastiche"), 0755)).To(Succeed())
				config := fmt.Sprintf(`
name: "@example/customers"
servers:
  - name: production
    baseUrl: %s
resources:
  - name: r
    query:
      id: "${id}"
varSets:
  - name: "@example/customers"
    vars:
      loyal:
        id: "ABC"
`, testServer.URL)
				Expect(os.WriteFile(filepath.Join(tmpDir, ".pastiche", "customers.yml"), []byte(config), 0644)).To(Succeed())

				Expect(os.Chdir(tmpDir)).To(Succeed())
			})

			AfterEach(func() {
				testServer.Close()
			})

			run := func(arguments string) error {
				var capture bytes.Buffer
				app := pastiche.NewApp()
				app.Stdout = &capture

				args, _ := cli.Split(arguments)
				return app.RunContext(context.Background(), args...)
			}

			It("resolves the value using the full path", func() {
				err := run("pastiche fetch @example/customers r -Kid=loyal.id")
				Expect(err).NotTo(HaveOccurred())
				Expect(receivedQuery).To(Equal("ABC"))
			})

			It("resolves the value using the shorthand path naming only the group", func() {
				err := run("pastiche fetch @example/customers r -Kid=loyal")
				Expect(err).NotTo(HaveOccurred())
				Expect(receivedQuery).To(Equal("ABC"))
			})

			It("implies the context from the service name when --context is not given", func() {
				err := run("pastiche fetch @example/customers r -Kid=loyal.id")
				Expect(err).NotTo(HaveOccurred())
				Expect(receivedQuery).To(Equal("ABC"))
			})

			It("uses the varset named by --context", func() {
				err := run("pastiche fetch @example/customers r -c @example/customers -Kid=loyal.id")
				Expect(err).NotTo(HaveOccurred())
				Expect(receivedQuery).To(Equal("ABC"))
			})

			It("lets an explicit --param win over the context-resolved value", func() {
				err := run("pastiche fetch @example/customers r -Kid=loyal.id -Tid=explicit")
				Expect(err).NotTo(HaveOccurred())
				Expect(receivedQuery).To(Equal("explicit"))
			})
		})

		Context("when using the context and var expanders", func() {

			var (
				testServer  *httptest.Server
				receivedURL *url.URL
			)

			BeforeEach(func() {
				testServer = httptest.NewServer(
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						receivedURL = r.URL
						fmt.Fprint(w, "ok")
					}),
				)

				tmpDir := GinkgoT().TempDir()
				origDir, err := os.Getwd()
				Expect(err).NotTo(HaveOccurred())
				DeferCleanup(func() { _ = os.Chdir(origDir) })

				Expect(os.MkdirAll(filepath.Join(tmpDir, ".pastiche"), 0755)).To(Succeed())
				config := fmt.Sprintf(`
name: "@example/customers"
servers:
  - name: production
    baseUrl: %s
resources:
  - name: r
    query:
      contextId: "${context.loyal.id}"
      qualifiedId: "${var.@example/customers.loyal.id}"
varSets:
  - name: "@example/customers"
    vars:
      loyal:
        id: "ABC"
  - name: "@example/other"
    vars:
      loyal:
        id: "XYZ"
`, testServer.URL)
				Expect(os.WriteFile(filepath.Join(tmpDir, ".pastiche", "customers.yml"), []byte(config), 0644)).To(Succeed())

				Expect(os.Chdir(tmpDir)).To(Succeed())
			})

			AfterEach(func() {
				testServer.Close()
			})

			run := func(arguments string) error {
				var capture bytes.Buffer
				app := pastiche.NewApp()
				app.Stdout = &capture

				args, _ := cli.Split(arguments)
				return app.RunContext(context.Background(), args...)
			}

			It("resolves ${context...} from the implied context varset", func() {
				err := run("pastiche fetch @example/customers r")
				Expect(err).NotTo(HaveOccurred())
				Expect(receivedURL.Query().Get("contextId")).To(Equal("ABC"))
			})

			It("resolves ${var.@name...} by qualified varset name, regardless of context", func() {
				err := run("pastiche fetch @example/customers r")
				Expect(err).NotTo(HaveOccurred())
				Expect(receivedURL.Query().Get("qualifiedId")).To(Equal("ABC"))
			})

			It("resolves ${context...} from the varset named by --context", func() {
				err := run("pastiche fetch @example/customers r -c @example/other")
				Expect(err).NotTo(HaveOccurred())
				Expect(receivedURL.Query().Get("contextId")).To(Equal("XYZ"))
			})
		})
	})
})
