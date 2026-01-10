// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package model_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/Carbonfrost/pastiche/pkg/model"
	"github.com/onsi/gomega/types"
)

var _ = Describe("Validate", func() {

	DescribeTable("examples", func(m *model.Model, expected types.GomegaMatcher) {
		err := model.Validate(m)
		Expect(err).To(expected)
	},
		Entry("invalid service name",
			&model.Model{
				Services: map[string]*model.Service{
					"": {
						Name: "invalid name",
					},
				},
			},
			MatchError(ContainSubstring("A name must start with a letter")),
		),
		Entry("invalid package service name",
			&model.Model{
				Services: map[string]*model.Service{
					"": {
						Name: "@hello/three/parts",
					},
				},
			},
			MatchError(ContainSubstring("A package-scoped name must have")),
		),
		Entry("invalid package service name",
			&model.Model{
				Services: map[string]*model.Service{
					"": {
						Name: "@hello/invalid name",
					},
				},
			},
			MatchError(ContainSubstring("A package-scoped name must have")),
		),
		Entry("invalid resource name",
			&model.Model{
				Services: map[string]*model.Service{
					"": {
						Name: "@httpbin/name",
						Resource: &model.Resource{
							Name: "invalid resource",
						},
					},
				},
			},
			MatchError(ContainSubstring("A name must start with a letter")),
		),
	)
})
