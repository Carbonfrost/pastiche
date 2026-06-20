// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package model_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/Carbonfrost/joe-cli-http/uritemplates"
	"github.com/Carbonfrost/pastiche/pkg/model"
	"github.com/onsi/gomega/types"
)

var _ = Describe("Validate", func() {

	DescribeTable("examples", func(m *model.Model) {
		err := model.Validate(m)
		Expect(err).NotTo(HaveOccurred())
	},
		Entry("valid",
			&model.Model{
				Services: []*model.Service{
					{
						Name: "valid",
					},
					{
						Name: "@also/valid",
						Resource: &model.Resource{
							Name: "valid",
						},
					},
				},
			},
		),
	)

	DescribeTable("errors", func(m *model.Model, expected types.GomegaMatcher) {
		err := model.Validate(m)
		Expect(err).To(expected)
	},
		Entry("invalid service name",
			&model.Model{
				Services: []*model.Service{
					{
						Name: "invalid name",
					},
				},
			},
			MatchError(ContainSubstring("A name must start with a letter")),
		),
		Entry("invalid package service name",
			&model.Model{
				Services: []*model.Service{
					{
						Name: "@hello/three/parts",
					},
				},
			},
			MatchError(ContainSubstring("A package-scoped name must have")),
		),
		Entry("invalid package service name",
			&model.Model{
				Services: []*model.Service{
					{
						Name: "@hello/invalid name",
					},
				},
			},
			MatchError(ContainSubstring("A package-scoped name must have")),
		),
		Entry("invalid resource name",
			&model.Model{
				Services: []*model.Service{
					{
						Name: "@httpbin/name",
						Resource: &model.Resource{
							Name: "invalid resource",
						},
					},
				},
			},
			MatchError(ContainSubstring("A name must start with a letter")),
		),
		Entry("invalid qualified resource name",
			&model.Model{
				Services: []*model.Service{
					{
						Name: "@httpbin/name",
						Resource: &model.Resource{
							Name: "invalid.resource",
						},
					},
				},
			},
			MatchError(ContainSubstring("A name must start with a letter")),
		),

		Entry("unexpected template expression in service",
			&model.Model{
				Services: []*model.Service{
					{
						Name: "s",
						Resource: &model.Resource{
							URITemplate: mustParseURITemplate("https://local.example/${var.invalid}"),
						},
					},
				},
			},
			MatchError(ContainSubstring("URL cannot contain template expressions ${...}")),
		),

		Entry("unexpected template expression in server",
			&model.Model{
				Services: []*model.Service{
					{
						Name: "s",
						Servers: []*model.Server{
							{
								BaseURL: "https://local.example/${var.invalid}",
							},
						},
					},
				},
			},
			MatchError(ContainSubstring("URL cannot contain template expressions ${...}")),
		),
	)
})

func mustParseURITemplate(t string) *uritemplates.URITemplate {
	u, err := uritemplates.Parse(t)
	if err != nil {
		panic(err)
	}
	return u
}
