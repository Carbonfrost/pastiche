// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package model_test

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/Carbonfrost/pastiche/pkg/model"
	"github.com/onsi/gomega/types"
)

var _ = Describe("NewRequest", func() {

	Context("Headers", func() {

		DescribeTable("examples", func(expected types.GomegaMatcher) {
			resource := &model.ResolvedResource{
				Service: &model.Service{},
				Server: &model.Server{
					Headers: newHeader("T", "server"),
				},
				Endpoint: &model.Endpoint{
					Headers: newHeader("S", "endpoint", "E", "endpoint", "T", "X"),
				},
				Lineage: []*model.Resource{
					{
						Headers: newHeader("S", "parent", "U", "lineage2", "W", "lineage2", "T", "X"),
					},
					{
						Headers: newHeader("S", "child", "V", "lineage1", "W", "lineage1", "T", "X"),
					},
				},
			}

			req, err := model.NewRequest(resource, model.WithBaseURL(mustParseURL("https://example.com")))
			Expect(err).NotTo(HaveOccurred())
			Expect(req.Headers).To(expected)
		},
			Entry(
				"reduce by lineage",
				And(
					HaveKeyWithValue("U", []string{"lineage2"}),
					HaveKeyWithValue("V", []string{"lineage1"}),
					HaveKeyWithValue("W", []string{"lineage1"}),
				),
			),
			Entry(
				"narrowest child value",
				HaveKeyWithValue("S", []string{"endpoint"}),
			),
			Entry(
				"server overrides everything",
				HaveKeyWithValue("T", []string{"server"}),
			),
		)
	})

	Context("Query", func() {

		It("merges into the result", func() {
			resource := &model.ResolvedResource{
				Endpoint: &model.Endpoint{
					Query: newHeader("hello", "world", "in", "${var.s}"),
				},
				Lineage: []*model.Resource{
					{
						URITemplate: mustParseURITemplate("/{?x}"),
					},
				},
			}

			result, _ := model.NewRequest(
				resource,
				model.WithBaseURL(mustParseURL("https://localhost:8080")),
				model.WithVars(map[string]any{"s": "to", "x": "y"}),
			)
			Expect(result.URL.String()).To(Equal("https://localhost:8080/?hello=world&in=to&x=y"))
		})

		DescribeTable("examples", func(expected types.GomegaMatcher) {
			resource := &model.ResolvedResource{
				Service: &model.Service{},
				Server: &model.Server{
					Query: newHeader("T", "server"),
				},
				Endpoint: &model.Endpoint{
					Query: newHeader("S", "endpoint", "E", "endpoint", "T", "X"),
				},
				Lineage: []*model.Resource{
					{
						Query: newHeader("S", "parent", "U", "lineage2", "W", "lineage2", "T", "X"),
					},
					{
						Query: newHeader("S", "child", "V", "lineage1", "W", "lineage1", "T", "X"),
					},
				},
			}

			req, err := model.NewRequest(resource, model.WithBaseURL(mustParseURL("https://example.com")))
			Expect(err).NotTo(HaveOccurred())
			Expect(req.URL.Query()).To(expected)
		},
			Entry(
				"reduce by lineage",
				And(
					HaveKeyWithValue("U", []string{"lineage2"}),
					HaveKeyWithValue("V", []string{"lineage1"}),
					HaveKeyWithValue("W", []string{"lineage1"}),
				),
			),
			Entry(
				"narrowest child value",
				HaveKeyWithValue("S", []string{"endpoint"}),
			),
			Entry(
				"server overrides everything",
				HaveKeyWithValue("T", []string{"server"}),
			),
		)
	})

	Context("context expander", func() {

		var exampleVarSet = &model.VarSet{
			Name: "@example/customers",
			Vars: map[string]map[string]any{
				"loyal": {"id": "ABC"},
			},
		}

		It("resolves a dotted path within the selected context varset", func() {
			resource := &model.ResolvedResource{
				Endpoint: &model.Endpoint{
					Query: newHeader("id", "${context.loyal.id}"),
				},
			}

			result, err := model.NewRequest(
				resource,
				model.WithBaseURL(mustParseURL("https://example.com")),
				model.WithContext(exampleVarSet),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.URL.Query()).To(HaveKeyWithValue("id", []string{"ABC"}))
		})

		It("resolves to nothing when there is no context", func() {
			resource := &model.ResolvedResource{
				Endpoint: &model.Endpoint{
					Query: newHeader("id", "${context.loyal.id}"),
				},
			}

			result, err := model.NewRequest(
				resource,
				model.WithBaseURL(mustParseURL("https://example.com")),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.URL.Query()).To(HaveKeyWithValue("id", []string{""}))
		})
	})

	Context("var expander", func() {

		It("resolves a plain name from the effective vars", func() {
			resource := &model.ResolvedResource{
				Endpoint: &model.Endpoint{
					Query: newHeader("id", "${var.id}"),
				},
			}

			result, err := model.NewRequest(
				resource,
				model.WithBaseURL(mustParseURL("https://example.com")),
				model.WithVars(map[string]any{"id": "ABC"}),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.URL.Query()).To(HaveKeyWithValue("id", []string{"ABC"}))
		})

		It("resolves a qualified name by digging any varset in the model", func() {
			resource := &model.ResolvedResource{
				Endpoint: &model.Endpoint{
					Query: newHeader("id", "${var.@example/customers.loyal.id}"),
				},
			}

			mo := &model.Model{
				VarSets: []*model.VarSet{
					{
						Name: "@example/customers",
						Vars: map[string]map[string]any{
							"loyal": {"id": "ABC"},
						},
					},
				},
			}

			result, err := model.NewRequest(
				resource,
				model.WithBaseURL(mustParseURL("https://example.com")),
				model.WithModel(mo),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.URL.Query()).To(HaveKeyWithValue("id", []string{"ABC"}))
		})

		It("resolves to nothing for an unknown qualified varset", func() {
			resource := &model.ResolvedResource{
				Endpoint: &model.Endpoint{
					Query: newHeader("id", "${var.@unknown/varset.loyal.id}"),
				},
			}

			result, err := model.NewRequest(
				resource,
				model.WithBaseURL(mustParseURL("https://example.com")),
				model.WithModel(&model.Model{}),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.URL.Query()).To(HaveKeyWithValue("id", []string{""}))
		})
	})

	Context("Secrets", func() {

		var newResource = func(secrets []*model.Secret) *model.ResolvedResource {
			return &model.ResolvedResource{
				Service: &model.Service{},
				Secrets: secrets,
				Endpoint: &model.Endpoint{
					Headers: newHeader("Authorization", "Bearer ${secret.token}"),
				},
			}
		}

		It("expands file secrets and trims trailing whitespace by default", func() {
			file := filepath.Join(GinkgoT().TempDir(), "token")
			Expect(os.WriteFile(file, []byte("s3cret\n"), 0600)).To(Succeed())

			resource := newResource([]*model.Secret{
				{Name: "token", Provider: &model.FileSecret{Path: file}},
			})

			req, err := model.NewRequest(resource, model.WithBaseURL(mustParseURL("https://example.com")))
			Expect(err).NotTo(HaveOccurred())
			Expect(req.Headers).To(HaveKeyWithValue("Authorization", []string{"Bearer s3cret"}))
		})

		It("preserves trailing whitespace when requested", func() {
			file := filepath.Join(GinkgoT().TempDir(), "token")
			Expect(os.WriteFile(file, []byte("s3cret\n"), 0600)).To(Succeed())

			resource := newResource([]*model.Secret{
				{Name: "token", Provider: &model.FileSecret{Path: file, PreserveTrailingWhitespace: true}},
			})

			req, err := model.NewRequest(resource, model.WithBaseURL(mustParseURL("https://example.com")))
			Expect(err).NotTo(HaveOccurred())
			Expect(req.Headers).To(HaveKeyWithValue("Authorization", []string{"Bearer s3cret\n"}))
		})

		It("expands exec secrets from stdout", func() {
			resource := newResource([]*model.Secret{
				{Name: "token", Provider: &model.ExecSecret{Command: "printf s3cret"}},
			})

			req, err := model.NewRequest(resource, model.WithBaseURL(mustParseURL("https://example.com")))
			Expect(err).NotTo(HaveOccurred())
			Expect(req.Headers).To(HaveKeyWithValue("Authorization", []string{"Bearer s3cret"}))
		})

		It("returns a placeholder when the secret fails to load", func() {
			resource := newResource([]*model.Secret{
				{Name: "token", Provider: &model.FileSecret{Path: "/does/not/exist"}},
			})

			req, err := model.NewRequest(resource, model.WithBaseURL(mustParseURL("https://example.com")))
			Expect(err).NotTo(HaveOccurred())
			Expect(req.Headers).To(HaveKeyWithValue("Authorization", []string{"Bearer <missing secret token>"}))
		})
	})
})

func newHeader(namevalues ...string) model.Values {
	if len(namevalues)%2 != 0 {
		panic(fmt.Errorf("requires even number of arguments, got %d", len(namevalues)))
	}
	v := make(model.Values, 0, len(namevalues)/2)
	for kvp := range slices.Chunk(namevalues, 2) {
		v = append(v, model.Value{Name: kvp[0], Value: kvp[1]})
	}

	return v
}

func mustParseURL(t string) *url.URL {
	u, err := url.Parse(t)
	if err != nil {
		panic(err)
	}
	return u
}
