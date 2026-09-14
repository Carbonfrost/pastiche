// Copyright 2025, 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config_test

import (
	"encoding/json"

	"github.com/Carbonfrost/pastiche/pkg/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("ServiceSpec", func() {

	Describe("UnmarshalJSON", func() {

		DescribeTable("examples",
			func(in string, expected types.GomegaMatcher) {
				var ss config.ServiceSpec
				err := json.Unmarshal([]byte(in), &ss)
				Expect(err).NotTo(HaveOccurred())
				Expect(ss).To(expected)
			},

			Entry(
				"slice",
				`[ "example", "please" ]`,
				Equal(config.ServiceSpec{"example", "please"}),
			),
			Entry(
				"qualified name slice",
				`[ "@example/api", "please" ]`,
				Equal(config.ServiceSpec{"@example/api", "please"}),
			),
			Entry(
				"string",
				`"@example/api.please"`,
				Equal(config.ServiceSpec{"@example/api", "please"}),
			),
		)
	})
})
