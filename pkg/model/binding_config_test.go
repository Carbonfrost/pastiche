// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package model_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/Carbonfrost/pastiche/pkg/config"
	"github.com/Carbonfrost/pastiche/pkg/model"
)

var _ = Describe("ToConfig", func() {

	It("converts to a config value", func() {
		subject := model.New(&config.Config{
			Services: []config.Service{
				config.ExampleHTTPBinorg(),
			},
		})

		Expect(func() {
			model.ToConfig(subject)
		}).NotTo(Panic())

	})
})
