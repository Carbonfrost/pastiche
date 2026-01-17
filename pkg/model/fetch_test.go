// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package model_test

import (
	"github.com/Carbonfrost/pastiche/pkg/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

const fetchExpr = `fetch("https://api.example.com/graphql", {
    "body": "body",
    "cache": "default",
    "credentials": "include",
    "headers": {
        "Accept": "*/*",
        "Accept-Language": "en-US,en;q=0.9",
        "Authorization": "Bearer XXX",
        "Content-Type": "application/json"
    },
    "method": "POST",
    "mode": "cors",
    "redirect": "follow",
    "referrer": "https://api.example.com/",
    "referrerPolicy": "strict-origin-when-cross-origin"
})
`

var _ = Describe("ParseJSFetchCall", func() {

	It("examples", func() {
		call, err := model.ParseJSFetchCall(fetchExpr)
		Expect(err).NotTo(HaveOccurred())

		Expect(call.URL).To(Equal("https://api.example.com/graphql"))
		Expect(call.Options).To(
			MatchFields(IgnoreUnexportedExtras,
				Fields{
					"Method": Equal("POST"),
					"Headers": Equal(map[string]string{
						"Accept":          "*/*",
						"Accept-Language": "en-US,en;q=0.9",
						"Authorization":   "Bearer XXX",
						"Content-Type":    "application/json",
					}),
					"Body": Equal("body"),
				}),
		)
	})

})
