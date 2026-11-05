// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pastiche_test

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"

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

				useWorkspace("context-param", testServer.URL)
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

				useWorkspace("context-expanders", testServer.URL)
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

		Context("when using the login expander", func() {

			var (
				testServer *httptest.Server
				received   *http.Request
			)

			BeforeEach(func() {
				testServer = httptest.NewServer(
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						received = r
						fmt.Fprint(w, "ok")
					}),
				)

				useWorkspace("login", testServer.URL)
			})

			AfterEach(func() {
				testServer.Close()
			})

			It("resolves ${login...} from the workspace logins directory", func() {
				app := pastiche.NewApp()
				app.Stdout = &bytes.Buffer{}

				args, _ := cli.Split("pastiche fetch @example/customers r")
				Expect(app.RunContext(context.Background(), args...)).To(Succeed())
				Expect(received.Header.Get("Authorization")).To(Equal("Bearer t0ken"))
				Expect(received.URL.Query().Get("expiresYear")).To(Equal("2999"))
			})

			It("resolves ${login...} for the server given by --server", func() {
				app := pastiche.NewApp()
				app.Stdout = &bytes.Buffer{}

				args, _ := cli.Split("pastiche fetch @example/customers r -S staging")
				Expect(app.RunContext(context.Background(), args...)).To(Succeed())
				Expect(received.Header.Get("Authorization")).To(Equal("Bearer st4ging"))
			})
		})

		Context("when using auth logout", func() {

			BeforeEach(func() {
				useWorkspace("login", "http://localhost")
			})

			run := func(arguments string) error {
				app := pastiche.NewApp()
				app.Stdout = &bytes.Buffer{}
				app.Stderr = &bytes.Buffer{}

				args, _ := cli.Split(arguments)
				return app.RunContext(context.Background(), args...)
			}

			loginExists := func(server string) bool {
				_, err := os.Stat(filepath.Join(".pastiche", "logins", "@example-customers."+server+".json"))
				return err == nil
			}

			It("removes the login for the default server of the service", func() {
				Expect(run("pastiche auth logout @example/customers")).To(Succeed())
				Expect(loginExists("production")).To(BeFalse())
				Expect(loginExists("staging")).To(BeTrue())
			})

			It("removes the login for the server given by --server", func() {
				Expect(run("pastiche auth logout @example/customers -S staging")).To(Succeed())
				Expect(loginExists("staging")).To(BeFalse())
				Expect(loginExists("production")).To(BeTrue())
			})

			It("succeeds when there is no login", func() {
				Expect(run("pastiche auth logout @example/customers -S missing")).To(Succeed())
			})

			It("requires the service", func() {
				Expect(run("pastiche auth logout")).To(HaveOccurred())
			})
		})
	})
})

const baseURLPlaceholder = "__TEST_SERVER_BASE_URL__"

func useWorkspace(fixture, baseURL string) {
	GinkgoHelper()

	src := filepath.Join("testdata", "valid-workspaces", fixture)
	tmpDir := GinkgoT().TempDir()
	workspaceDir := filepath.Join(tmpDir, ".pastiche")

	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dst := filepath.Join(workspaceDir, rel)
		if d.IsDir() {
			return os.MkdirAll(dst, 0755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		data = []byte(strings.ReplaceAll(string(data), baseURLPlaceholder, baseURL))
		return os.WriteFile(dst, data, 0644)
	})
	Expect(err).NotTo(HaveOccurred())

	origDir, err := os.Getwd()
	Expect(err).NotTo(HaveOccurred())
	DeferCleanup(func() { _ = os.Chdir(origDir) })
	Expect(os.Chdir(tmpDir)).To(Succeed())
}
