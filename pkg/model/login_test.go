// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package model_test

import (
	"fmt"
	"io/fs"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/Carbonfrost/pastiche/pkg/internal/modelfakes"
	"github.com/Carbonfrost/pastiche/pkg/model"
)

var _ = Describe("WithLogins", func() {

	expires := time.Date(2030, time.March, 4, 5, 6, 7, 0, time.UTC)
	var (
		requestedName   string
		requestedServer string
	)

	logins := func(err error) model.LoginProvider {
		return fakeLogins(func(name, server string) (*model.Login, error) {
			requestedName, requestedServer = name, server
			if name != "@example/customers" {
				return nil, fs.ErrNotExist
			}
			return &model.Login{Name: name, Server: server, Token: "t0ken", Expires: &expires}, err
		})
	}

	newRequest := func(query string, err error) *model.Request {
		resource := &model.ResolvedResource{
			Server: &model.Server{Name: "production"},
			Endpoint: &model.Endpoint{
				Query: newHeader("q", query),
			},
		}
		req, e := model.NewRequest(
			resource,
			model.WithBaseURL("https://example.com"),
			model.WithLogins(logins(err)),
		)
		Expect(e).NotTo(HaveOccurred())
		return req
	}

	DescribeTable("examples", func(query string, expected string) {
		req := newRequest(query, nil)
		Expect(req.URL.Query().Get("q")).To(Equal(expected))
	},
		Entry("token", "${login.@example/customers}", "t0ken"),
		Entry("expires", "${login.@example/customers.expires}", "2030-03-04T05:06:07Z"),
		Entry("expires time expansion", "${login.@example/customers.expires.year}", "2030"),
		Entry("expires nested time expansion", "${login.@example/customers.expires.date}", "2030-03-04"),
		Entry("expires timestamp", "${login.@example/customers.expires.unix}", fmt.Sprint(expires.Unix())),
		Entry("unknown property", "${login.@example/customers.other}", ""),
		Entry("unknown login", "${login.@example/other}", ""),
	)

	It("looks up the login using the server of the request", func() {
		newRequest("${login.@example/customers}", nil)
		Expect(requestedName).To(Equal("@example/customers"))
		Expect(requestedServer).To(Equal("production"))
	})

	It("returns a placeholder for the token when the login fails to load", func() {
		req := newRequest("${login.@example/customers}", model.ErrLoginExpired)
		Expect(req.URL.Query().Get("q")).To(Equal("<missing login @example/customers>"))
	})

	It("still provides the expiration time of an expired login", func() {
		req := newRequest("${login.@example/customers.expires.year}", model.ErrLoginExpired)
		Expect(req.URL.Query().Get("q")).To(Equal("2030"))
	})
})

func fakeLogins(fn func(name, server string) (*model.Login, error)) model.LoginProvider {
	return &modelfakes.FakeLoginProvider{
		GetLoginStub: fn,
	}
}
