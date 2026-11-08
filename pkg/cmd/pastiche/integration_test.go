// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pastiche_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

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

		Context("when using auth list", func() {

			run := func(arguments string) (string, error) {
				var out bytes.Buffer
				app := pastiche.NewApp()
				app.Stdout = &out

				args, _ := cli.Split(arguments)
				err := app.RunContext(context.Background(), args...)
				return out.String(), err
			}

			It("lists the logins without their tokens", func() {
				useWorkspace("login", "http://localhost")

				out, err := run("pastiche auth list")
				Expect(err).NotTo(HaveOccurred())
				Expect(out).To(Equal(
					"NAME                SERVER      EXPIRES\n" +
						"@example/customers  legacy      2000-01-01T00:00:00Z (expired)\n" +
						"@example/customers  production  2999-01-02T03:04:05Z\n" +
						"@example/customers  staging     never\n",
				))
				Expect(out).NotTo(ContainSubstring("t0ken"))
			})

			It("prints nothing when there are no logins", func() {
				useWorkspace("context-param", "http://localhost")

				out, err := run("pastiche auth list")
				Expect(err).NotTo(HaveOccurred())
				Expect(out).To(BeEmpty())
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
				err := app.RunContext(context.Background(), args...)

				// Flags which exit, such as --expired, report a zero exit code
				var exit cli.ExitCoder
				if errors.As(err, &exit) && exit.ExitCode() == 0 {
					return nil
				}
				return err
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
				Expect(run("pastiche auth logout")).To(MatchError("expected argument for service"))
			})

			It("removes all logins with --all", func() {
				Expect(run("pastiche auth logout --all")).To(Succeed())
				Expect(loginExists("production")).To(BeFalse())
				Expect(loginExists("staging")).To(BeFalse())
				Expect(loginExists("legacy")).To(BeFalse())
			})

			It("displays help without requiring the service", func() {
				Expect(run("pastiche auth logout --help")).To(Succeed())
			})

			It("removes only the expired logins with --expired", func() {
				Expect(run("pastiche auth logout --expired")).To(Succeed())
				Expect(loginExists("legacy")).To(BeFalse())
				Expect(loginExists("production")).To(BeTrue())
				Expect(loginExists("staging")).To(BeTrue())
			})

			It("remove all when both flags are specified", func() {
				Expect(run("pastiche auth logout --expired --all")).To(Succeed())
				Expect(loginExists("legacy")).To(BeFalse())
				Expect(loginExists("production")).To(BeFalse())
				Expect(loginExists("staging")).To(BeFalse())
			})
		})

		Context("when using auth login", func() {

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

			run := func(arguments string, stdin string) error {
				app := pastiche.NewApp()
				app.Stdout = &bytes.Buffer{}
				app.Stdin = strings.NewReader(stdin)

				args, _ := cli.Split(arguments)
				return app.RunContext(context.Background(), args...)
			}

			readLogin := func(server string) map[string]any {
				data, err := os.ReadFile(filepath.Join(".pastiche", "logins", "@example-customers."+server+".json"))
				Expect(err).NotTo(HaveOccurred())

				var result map[string]any
				Expect(json.Unmarshal(data, &result)).To(Succeed())
				return result
			}

			expiresIn := func(login map[string]any) time.Duration {
				t, err := time.Parse(time.RFC3339, login["expires"].(string))
				Expect(err).NotTo(HaveOccurred())
				return time.Until(t)
			}

			It("stores the token from stdin for the default server, which requests then use", func() {
				Expect(run("pastiche auth login --with-token @example/customers", "n3w\n")).To(Succeed())

				login := readLogin("production")
				Expect(login).To(HaveKeyWithValue("token", "n3w"))
				Expect(expiresIn(login)).To(BeNumerically("~", 365*24*time.Hour, time.Minute))

				Expect(run("pastiche fetch @example/customers r", "")).To(Succeed())
				Expect(received.Header.Get("Authorization")).To(Equal("Bearer n3w"))
			})

			It("stores the token for the server named in the service spec with --expires", func() {
				Expect(run("pastiche auth login --with-token --expires=3600s @example/customers.staging", "st\n")).To(Succeed())

				login := readLogin("staging")
				Expect(login).To(HaveKeyWithValue("token", "st"))
				Expect(expiresIn(login)).To(BeNumerically("~", time.Hour, time.Minute))
			})

			DescribeTable("errors",
				func(arguments string, stdin string, expected string) {
					Expect(run(arguments, stdin)).To(MatchError(expected))
				},
				Entry("missing --with-token", "pastiche auth login @example/customers", "t", "--with-token is required"),
				Entry("missing spec", "pastiche auth login --with-token", "t", "expected argument for service"),
				Entry("empty token", "pastiche auth login --with-token @example/customers", " \n", "no token was provided on stdin"),
				Entry("unknown service", "pastiche auth login --with-token @example/nope", "t", `service not found: "@example/nope"`),
				Entry("unknown server", "pastiche auth login --with-token @example/customers.nope", "t", `no server "nope" defined for service "@example/customers"`),
				Entry("non-positive expires", "pastiche auth login --with-token --expires=0s @example/customers", "t", "--expires must be a positive duration: 0s"),
			)
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
