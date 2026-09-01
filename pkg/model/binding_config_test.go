// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package model_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	"github.com/Carbonfrost/pastiche/pkg/config"
	"github.com/Carbonfrost/pastiche/pkg/model"
)

var _ = Describe("ToConfig", func() {

	It("converts to a config value", func() {
		subject := model.New(config.BuiltinFiles()...)

		Expect(func() {
			model.ToConfig(subject)
		}).NotTo(Panic())

	})

	DescribeTable("examples", func(item model.Item, expected any) {
		Expect(model.ToConfig(item)).To(Equal(expected))
	},
		Entry("var set",
			&model.VarSet{
				Name:        "creds",
				Comment:     "TODO Move these out of the workspace",
				Title:       "Credentials",
				Description: "Credentials used by the demo",
				Tags:        []string{"private"},
				Vars: map[string]map[string]any{
					"default": {"token": "abc"},
				},
			},
			config.VarSet{
				Schema:      config.SchemaVarSet,
				Name:        "creds",
				Comment:     "TODO Move these out of the workspace",
				Title:       "Credentials",
				Description: "Credentials used by the demo",
				Tags:        []string{"private"},
				Links:       []config.Link{},
				Vars: map[string]map[string]any{
					"default": {"token": "abc"},
				},
			}),
		Entry("flow",
			&model.Flow{
				Name:  "login",
				Title: "Log in",
				Steps: []*model.Step{
					{
						Name:     "obtain token",
						Method:   "POST",
						StepType: &model.SpecStep{Spec: "demo.tokens"},
					},
					{
						Name:     "call out",
						StepType: &model.URLStep{URL: "https://example.com"},
					},
				},
				Vars: map[string]any{"user": "root"},
			},
			config.Flow{
				Schema: config.SchemaFlow,
				Name:   "login",
				Title:  "Log in",
				Links:  []config.Link{},
				Steps: []config.Step{
					{
						Name:   "obtain token",
						Method: "POST",
						Spec:   "demo.tokens",
						Links:  []config.Link{},
					},
					{
						Name:  "call out",
						URL:   "https://example.com",
						Links: []config.Link{},
					},
				},
				Vars: map[string]any{"user": "root"},
			}),
		Entry("endpoint",
			&model.Endpoint{
				Name:   "listWidgets",
				Method: "GET",
				Tags:   []string{"public"},
				Query: model.Values{
					{Name: "limit", Value: "10"},
				},
			},
			&config.Endpoint{
				Schema: config.SchemaEndpoint,
				Name:   "listWidgets",
				Tags:   []string{"public"},
				Links:  []config.Link{},
				Query: config.Values{
					{Name: "limit", Value: "10"},
				},
			}),
	)

	It("converts var sets and flows within a model", func() {
		subject := &model.Model{
			VarSets: []*model.VarSet{{Name: "creds"}},
			Flows:   []*model.Flow{{Name: "login"}},
		}

		Expect(model.ToConfigFile(subject)).To(PointTo(MatchFields(IgnoreExtras, Fields{
			"VarSets": ConsistOf(HaveField("Name", "creds")),
			"Flows":   ConsistOf(HaveField("Name", "login")),
		})))
	})
})
