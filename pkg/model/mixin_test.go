// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package model_test

import (
	"io"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/Carbonfrost/pastiche/pkg/config"
	"github.com/Carbonfrost/pastiche/pkg/model"
)

var _ = Describe("Mixin", func() {

	var (
		exampleFile = func() *config.File {
			return &config.File{

				// A service whose every configuration layer contributes the
				// value X-Layer so that we can tell which layer wins
				Services: []config.Service{
					{
						Name: "a",
						Vars: map[string]any{"scope": "service"},
						Servers: []config.Server{
							{
								Name:    "default",
								BaseURL: "https://example.com",
								Headers: config.ValuesFromMap(map[string][]string{
									"X-Layer":  {"server"},
									"X-Server": {"server"},
								}),
							},
						},
						Resources: []config.Resource{
							{
								Name: "b",
								URI:  "b",
								Headers: config.ValuesFromMap(map[string][]string{
									"X-Layer": {"resource"},
								}),
								Get: &config.Endpoint{
									Headers: config.ValuesFromMap(map[string][]string{
										"X-Layer":    {"endpoint"},
										"X-Endpoint": {"endpoint"},
									}),
									Body: `{"from":"endpoint"}`,
								},
								Post: &config.Endpoint{},
							},
						},
					},
				},
				Mixins: []config.Mixin{
					{
						Name:   "staging",
						Method: "POST",
						Headers: config.ValuesFromMap(map[string][]string{
							"X-Layer": {"mixin"},
							"X-Mixin": {"mixin"},
						}),
						Query: config.ValuesFromMap(map[string][]string{
							"trace": {"1"},
						}),
						Body: `{"from":"mixin"}`,
						Vars: map[string]any{"scope": "mixin"},
						Auth: &config.Auth{
							Basic: &config.BasicAuth{User: "u", Password: "p"},
						},
					},
					{
						Name: "verbose",
						Query: config.ValuesFromMap(map[string][]string{
							"verbose": {"true"},
						}),
					},
				},
			}
		}

		subject *model.Model
	)

	BeforeEach(func() {
		subject = model.New(exampleFile())
	})

	Describe("Mixin", func() {

		It("obtains the mixin by name", func() {
			mx, ok := subject.Mixin("staging")
			Expect(ok).To(BeTrue())
			Expect(mx.Name).To(Equal("staging"))
		})

		It("reports when the mixin is not defined", func() {
			_, ok := subject.Mixin("nope")
			Expect(ok).To(BeFalse())
		})
	})

	Describe("Resolve", func() {

		It("reports an error when the mixin is not defined", func() {
			_, err := subject.Resolve(strings.Fields("a b"), "default", "", "nope")
			Expect(err).To(MatchError(`mixin not found: "nope"`))
		})

		It("selects the endpoint named by the method of the mixin", func() {
			rr, err := subject.Resolve(strings.Fields("a b"), "default", "", "staging")
			Expect(err).NotTo(HaveOccurred())
			Expect(rr.Endpoint().Method).To(Equal("POST"))
		})

		It("prefers an explicit method over the method of the mixin", func() {
			rr, err := subject.Resolve(strings.Fields("a b"), "default", "GET", "staging")
			Expect(err).NotTo(HaveOccurred())
			Expect(rr.Endpoint().Method).To(Equal("GET"))
		})

		It("obtains the mixins in the order that they were named", func() {
			rr, err := subject.Resolve(strings.Fields("a b"), "default", "GET", "verbose", "staging")
			Expect(err).NotTo(HaveOccurred())
			Expect(names(rr.Mixins())).To(Equal([]string{"verbose", "staging"}))
		})
	})

	Describe("EvalRequest", func() {

		var eval = func(method string, mixins ...string) *model.Request {
			GinkgoHelper()
			rr, err := subject.Resolve(strings.Fields("a b"), "default", method, mixins...)
			Expect(err).NotTo(HaveOccurred())

			merged, err := rr.EvalRequest(nil, nil)
			Expect(err).NotTo(HaveOccurred())
			return merged
		}

		Describe("headers", func() {

			It("applies the mixin as the last layer", func() {
				Expect(eval("GET", "staging").Headers).To(
					HaveKeyWithValue("X-Layer", []string{"mixin"}))
			})

			It("retains the values of the layers which the mixin doesn't name", func() {
				merged := eval("GET", "staging")
				Expect(merged.Headers).To(HaveKeyWithValue("X-Endpoint", []string{"endpoint"}))
				Expect(merged.Headers).To(HaveKeyWithValue("X-Server", []string{"server"}))
				Expect(merged.Headers).To(HaveKeyWithValue("X-Mixin", []string{"mixin"}))
			})

			It("leaves the server as the last layer when no mixin is named", func() {
				Expect(eval("GET").Headers).To(HaveKeyWithValue("X-Layer", []string{"server"}))
			})
		})

		It("merges the query of each mixin into the URL", func() {
			Expect(eval("GET", "staging", "verbose").URL.String()).To(
				Equal("https://example.com/b?trace=1&verbose=true"))
		})

		It("applies the vars of the mixin", func() {
			Expect(eval("GET", "staging").Vars).To(HaveKeyWithValue("scope", "mixin"))
		})

		It("applies the auth of the mixin", func() {
			Expect(eval("GET", "staging").Auth).To(Equal(
				&model.BasicAuth{User: "u", Password: "p"}))
		})

		Describe("body", func() {

			It("prefers the body of the mixin over the one of the endpoint", func() {
				Expect(readAll(eval("GET", "staging").Body)).To(Equal(`{"from":"mixin"}`))
			})

			It("uses the body of the endpoint when no mixin defines one", func() {
				Expect(readAll(eval("GET", "verbose").Body)).To(Equal(`{"from":"endpoint"}`))
			})

			It("prefers the body of the last mixin which defines one", func() {
				Expect(readAll(eval("GET", "staging", "verbose").Body)).To(Equal(`{"from":"mixin"}`))
			})
		})
	})
})

func names(mixins []*model.Mixin) []string {
	res := make([]string, len(mixins))
	for i, m := range mixins {
		res[i] = m.Name
	}
	return res
}

func readAll(r io.Reader) string {
	GinkgoHelper()
	Expect(r).NotTo(BeNil())

	data, err := io.ReadAll(r)
	Expect(err).NotTo(HaveOccurred())
	return string(data)
}
