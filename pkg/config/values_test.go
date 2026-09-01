// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config_test

import (
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/Carbonfrost/pastiche/pkg/config"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("Values", func() {

	Describe("UnmarshalJSON", func() {

		DescribeTable("examples",
			func(jsonString string, expected types.GomegaMatcher) {
				v := config.Values{}
				err := json.Unmarshal([]byte(jsonString), &v)
				Expect(err).NotTo(HaveOccurred())
				Expect(v).To(expected)
			},

			Entry(
				"canonical list with single value",
				`[{"name":"user_id","value":1}]`,
				Equal(config.Values{{Name: "user_id", Value: "1"}})),

			Entry(
				"canonical list with explicit values",
				`[{"name":"user_id","values":[1,2]}]`,
				Equal(config.Values{{Name: "user_id", Values: []string{"1", "2"}}})),

			Entry(
				"canonical list with optional and merge",
				`[{"name":"scope","value":"read","optional":true,"merge":"APPEND"}]`,
				Equal(config.Values{{
					Name:     "scope",
					Value:    "read",
					Optional: true,
					Merge:    config.MergeAppend,
				}})),

			Entry(
				"abbreviated map single value",
				`{"user_id":1}`,
				Equal(config.Values{{Name: "user_id", Value: "1"}})),

			Entry(
				"abbreviated map value array",
				`{"scope":["read","write"]}`,
				Equal(config.Values{{Name: "scope", Values: []string{"read", "write"}}})),

			Entry(
				"abbreviated map is sorted by name",
				`{"b":"2","a":"1"}`,
				Equal(config.Values{
					{Name: "a", Value: "1"},
					{Name: "b", Value: "2"},
				})),
		)

		DescribeTable("errors",
			func(jsonString string, errExpected types.GomegaMatcher) {
				v := config.Values{}
				err := json.Unmarshal([]byte(jsonString), &v)
				Expect(err).To(errExpected)
			},
			Entry(
				"unknown merge mode",
				`[{"name":"x","merge":"NOPE"}]`,
				MatchError(`unknown merge mode: "NOPE"`)),
			Entry(
				"unexpected value type",
				`{"x":{"nested":true}}`,
				MatchError("unexpected type in config.Values: map[string]interface {}")),
		)
	})

	Describe("ToHeader", func() {
		DescribeTable("examples",
			func(input config.Values, expected http.Header) {
				Expect(input.ToHeader()).To(Equal(expected))
			},

			Entry(
				"single value",
				config.Values{{Name: "X-Trace", Value: "abc"}},
				http.Header{"X-Trace": {"abc"}}),

			Entry(
				"explicit values",
				config.Values{{Name: "X-Tag", Values: []string{"a", "b"}}},
				http.Header{"X-Tag": {"a", "b"}}),

			Entry(
				"optional empty is omitted",
				config.Values{{Name: "X-Skip", Optional: true}},
				http.Header{}),

			Entry(
				"non-optional empty is included",
				config.Values{{Name: "X-Keep"}},
				http.Header{"X-Keep": nil}),
		)
	})

	Describe("ToURLValues", func() {
		It("converts into url.Values", func() {
			v := config.Values{
				{Name: "user_id", Value: "1"},
				{Name: "scope", Values: []string{"read", "write"}},
				{Name: "empty", Optional: true},
			}
			Expect(v.ToURLValues()).To(Equal(url.Values{
				"user_id": {"1"},
				"scope":   {"read", "write"},
			}))
		})
	})

	Describe("Reduce", func() {
		DescribeTable("examples",
			func(base, next, expected config.Values) {
				Expect(base.Reduce(next)).To(Equal(expected))
			},

			Entry(
				"replace by default",
				config.Values{{Name: "scope", Value: "read"}},
				config.Values{{Name: "scope", Value: "write"}},
				config.Values{{Name: "scope", Value: "write"}}),

			Entry(
				"append accumulates values",
				config.Values{{Name: "scope", Value: "read"}},
				config.Values{{Name: "scope", Value: "write", Merge: config.MergeAppend}},
				config.Values{{Name: "scope", Values: []string{"read", "write"}, Merge: config.MergeAppend}}),

			Entry(
				"new name is added",
				config.Values{{Name: "a", Value: "1"}},
				config.Values{{Name: "b", Value: "2"}},
				config.Values{{Name: "a", Value: "1"}, {Name: "b", Value: "2"}}),
		)

		It("does not mutate the receiver", func() {
			base := config.Values{{Name: "scope", Value: "read"}}
			base.Reduce(config.Values{{Name: "scope", Value: "write", Merge: config.MergeAppend}})
			Expect(base).To(Equal(config.Values{{Name: "scope", Value: "read"}}))
		})
	})

	Describe("MarshalJSON", func() {
		DescribeTable("examples",
			func(v config.Values, expected string) {
				data, err := json.Marshal(v)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(data)).To(Equal(expected))
			},

			Entry(
				"canonical list form when merge rules are non-default",
				config.Values{{Name: "scope", Values: []string{"read", "write"}, Merge: config.MergeAppend}},
				`[{"name":"scope","values":["read","write"],"merge":"APPEND"}]`),

			Entry(
				"canonical list form when a value is optional",
				config.Values{{Name: "scope", Value: "read", Optional: true}},
				`[{"name":"scope","value":"read","optional":true}]`),

			Entry(
				"canonical list form when a name is duplicated",
				config.Values{{Name: "scope", Value: "read"}, {Name: "scope", Value: "write"}},
				`[{"name":"scope","value":"read"},{"name":"scope","value":"write"}]`),

			Entry(
				"canonical list form when a name is missing",
				config.Values{{Value: "read"}},
				`[{"value":"read"}]`),

			Entry(
				"canonical list form when both value and values are set",
				config.Values{{Name: "scope", Value: "read", Values: []string{"write"}}},
				`[{"name":"scope","value":"read","values":["write"]}]`),

			Entry(
				"abbreviated map of strings when every value is singular",
				config.Values{{Name: "user_id", Value: "1"}, {Name: "scope", Value: "read"}},
				`{"scope":"read","user_id":"1"}`),

			Entry(
				"abbreviated map of lists when any value has several values",
				config.Values{{Name: "user_id", Value: "1"}, {Name: "scope", Values: []string{"read", "write"}}},
				`{"scope":["read","write"],"user_id":["1"]}`),

			Entry(
				"abbreviated empty map",
				config.Values{},
				`{}`),
		)

		DescribeTable("round trips",
			func(v config.Values) {
				data, err := json.Marshal(v)
				Expect(err).NotTo(HaveOccurred())

				var actual config.Values
				Expect(json.Unmarshal(data, &actual)).To(Succeed())
				Expect(actual.ToURLValues()).To(Equal(v.ToURLValues()))
			},

			Entry(
				"singular values",
				config.Values{{Name: "user_id", Value: "1"}, {Name: "scope", Value: "read"}}),

			Entry(
				"mixed values",
				config.Values{{Name: "user_id", Value: "1"}, {Name: "scope", Values: []string{"read", "write"}}}),

			Entry(
				"non-default merge rules",
				config.Values{{Name: "scope", Values: []string{"read"}, Merge: config.MergeAppend}}),

			Entry(
				"optional values",
				config.Values{{Name: "scope", Optional: true}}),
		)
	})
})
