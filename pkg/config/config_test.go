// Copyright 2025 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package config_test

import (
	"os"

	"github.com/Carbonfrost/pastiche/pkg/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("Config", func() {

	Describe("LoadFile", func() {

		DescribeTable("examples",
			func(filename string, expected types.GomegaMatcher) {
				file, err := config.LoadFile(os.DirFS("testdata"), filename)
				Expect(err).NotTo(HaveOccurred())

				Expect(file).To(expected)
			},

			Entry(
				"basic",
				"basic.yml",
				And(
					haveServers(ContainElement(
						config.Server{Name: "production", BaseURL: "https://example.sh/"},
					)),
					haveResources(ContainElement(
						config.Resource{Name: "r", URI: "/api/{name}.json"},
					)),
				),
			),
			Entry(
				"multi",
				"multi.yml",
				haveServices(ContainElements(
					config.Service{Name: "foo"},
					config.Service{Name: "bar"},
					config.Service{Name: "baz"},
				)),
			),
		)

		DescribeTable("errors",
			func(filename string, expected types.GomegaMatcher) {
				_, err := config.LoadFile(os.DirFS("testdata"), filename)
				Expect(err).To(expected)
			},

			Entry(
				"unsupported file extension",
				"error_unsupported.html",
				MatchError(config.ErrUnsupportedFileFormat),
			),
			Entry(
				"both service and service list",
				"error_serviceList.yml",
				MatchError("must contain either service definition or services list, but not both"),
			),
		)

	})

})

func haveServices(m OmegaMatcher) OmegaMatcher {
	return WithTransform(func(cfg any) any {
		return cfg.(*config.File).Services
	}, m)
}

func haveServers(m OmegaMatcher) OmegaMatcher {
	return WithTransform(func(cfg any) any {
		return cfg.(*config.File).Service.Servers
	}, m)
}

func haveResources(m OmegaMatcher) OmegaMatcher {
	return WithTransform(func(cfg any) any {
		return cfg.(*config.File).Service.Resources
	}, m)
}
