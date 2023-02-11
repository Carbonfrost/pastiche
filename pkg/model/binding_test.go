package model_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/Carbonfrost/pastiche/pkg/config"
	"github.com/Carbonfrost/pastiche/pkg/model"
	. "github.com/onsi/gomega/gstruct"
	"github.com/onsi/gomega/types"
)

type resolveResultMatches struct {
	Service, Resource types.GomegaMatcher
}

var _ = Describe("Resolve", func() {

	DescribeTable("examples", func(spec []string, match resolveResultMatches) {
		subject := model.New(&config.Config{
			Services: []*config.Service{
				config.ExampleHTTPBinorg,
			},
		})

		service, resource, err := subject.Resolve(spec)
		Expect(err).NotTo(HaveOccurred())
		Expect(service).To(match.Service)
		Expect(resource).To(match.Resource)
	},
		Entry("simple",
			[]string{"httpbin", "get"},
			resolveResultMatches{
				Service:  PointTo(MatchFields(IgnoreExtras, Fields{"Name": Equal("httpbin")})),
				Resource: PointTo(MatchFields(IgnoreExtras, Fields{"Name": Equal("get")})),
			},
		),
		Entry("nested resource",
			[]string{"httpbin", "status", "codes"},
			resolveResultMatches{
				Service:  PointTo(MatchFields(IgnoreExtras, Fields{"Name": Equal("httpbin")})),
				Resource: PointTo(MatchFields(IgnoreExtras, Fields{"Name": Equal("codes")})),
			},
		),
	)
})
