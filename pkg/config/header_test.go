package config_test

import (
	"encoding/json"

	"github.com/Carbonfrost/pastiche/pkg/config"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("Header", func() {
	Describe("UnmarshalJSON", func() {
		DescribeTable("examples",
			func(jsonString string, expected types.GomegaMatcher) {
				h := config.Header{}
				_ = json.Unmarshal([]byte(jsonString), &h)
				Expect(h).To(expected)
			},

			Entry(
				"nominal",
				`{"Referer":["https://example.com"]}`,
				Equal(config.Header{"Referer": {"https://example.com"}})),

			Entry(
				"basic key values",
				`{"Referer":"https://example.com"}`,
				Equal(config.Header{"Referer": {"https://example.com"}})),
		)

		DescribeTable("errors",
			func(jsonString string, errExpected types.GomegaMatcher) {
				h := config.Header{}
				err := json.Unmarshal([]byte(jsonString), &h)
				Expect(err).To(errExpected)
			},
			Entry(
				"wrong type", `89`,
				BeAssignableToTypeOf(&json.UnmarshalTypeError{})),
			Entry(
				"error in values",
				`{"Link":3}`,
				MatchError("unexpected type in header: float64")),
		)
	})
})
