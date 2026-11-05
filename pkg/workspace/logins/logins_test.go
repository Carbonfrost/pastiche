// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package logins_test

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/Carbonfrost/pastiche/pkg/model"
	"github.com/Carbonfrost/pastiche/pkg/workspace/logins"
)

var _ = Describe("Logins", func() {

	var (
		dir      string
		provider model.LoginProvider
		future   = time.Now().Add(time.Hour).UTC().Truncate(time.Second)
		past     = time.Now().Add(-time.Hour).UTC().Truncate(time.Second)
	)

	writeLogin := func(path string, content string) {
		file := filepath.Join(dir, filepath.FromSlash(path))
		Expect(os.MkdirAll(filepath.Dir(file), 0700)).To(Succeed())
		Expect(os.WriteFile(file, []byte(content), 0600)).To(Succeed())
	}

	exists := func(path string) bool {
		_, err := os.Stat(filepath.Join(dir, filepath.FromSlash(path)))
		return err == nil
	}

	BeforeEach(func() {
		dir = GinkgoT().TempDir()
		provider = logins.New(dir)

		writeLogin("@example-customers.production.json", `{"token": "prod", "expires": "`+future.Format(time.RFC3339)+`"}`)
		writeLogin("@example-customers.json", `{"token": "noserver"}`)
		writeLogin("@example-customers.staging.json", `{"token": "stale", "expires": "`+past.Format(time.RFC3339)+`"}`)
	})

	Describe("GetLogin", func() {

		It("reads the file named by the name and server", func() {
			login, err := provider.GetLogin("@example/customers", "production")
			Expect(err).NotTo(HaveOccurred())
			Expect(login).To(Equal(&model.Login{
				Name:    "@example/customers",
				Server:  "production",
				Token:   "prod",
				Expires: &future,
			}))
		})

		It("reads the file named only by the name when there is no server", func() {
			login, err := provider.GetLogin("@example/customers", "")
			Expect(err).NotTo(HaveOccurred())
			Expect(login.Token).To(Equal("noserver"))
			Expect(login.Expires).To(BeNil())
		})

		It("returns ErrLoginExpired with the login when it has expired", func() {
			login, err := provider.GetLogin("@example/customers", "staging")
			Expect(err).To(MatchError(model.ErrLoginExpired))
			Expect(login.Token).To(Equal("stale"))
		})

		It("returns a not exist error for a missing login", func() {
			_, err := provider.GetLogin("@example/customers", "missing")
			Expect(err).To(MatchError(fs.ErrNotExist))
		})

		It("treats a file which records a different name as missing", func() {
			writeLogin("@example-other.json", `{"name": "@example/other", "token": "other"}`)

			_, err := provider.GetLogin("@example-other", "")
			Expect(err).To(MatchError(fs.ErrNotExist))

			login, err := provider.GetLogin("@example/other", "")
			Expect(err).NotTo(HaveOccurred())
			Expect(login.Token).To(Equal("other"))
		})

		DescribeTable("rejects names which are not service specs",
			func(name, server string) {
				_, err := provider.GetLogin(name, server)
				Expect(err).To(MatchError(ContainSubstring("invalid login name")))
			},
			Entry("empty", "", "production"),
			Entry("parent directory", "..", ""),
			Entry("path traversal", "../secrets", ""),
			Entry("dot in name", "a.b", ""),
			Entry("dot in server", "a", "b.c"),
			Entry("slash in server", "a", "b/c"),
		)
	})

	Describe("FileName", func() {

		It("normalizes slashes to hyphens so that the directory is flat", func() {
			name, err := provider.(*logins.Logins).FileName("@example/customers", "production")
			Expect(err).NotTo(HaveOccurred())
			Expect(name).To(Equal(filepath.Join(dir, "@example-customers.production.json")))
		})
	})

	Describe("NewEntry", func() {

		It("round-trips through JSON", func() {
			login := &model.Login{Name: "@example/customers", Server: "production", Token: "t", Expires: &future}
			data, err := json.Marshal(logins.NewEntry(login))
			Expect(err).NotTo(HaveOccurred())
			writeLogin("@example-customers.production.json", string(data))

			Expect(provider.GetLogin("@example/customers", "production")).To(Equal(login))
		})
	})

	Describe("ClearLogin", func() {

		It("removes the login", func() {
			Expect(provider.ClearLogin("@example/customers", "production")).To(Succeed())
			Expect(exists("@example-customers.production.json")).To(BeFalse())
			Expect(exists("@example-customers.json")).To(BeTrue())
		})

		It("succeeds when the login does not exist", func() {
			Expect(provider.ClearLogin("@example/customers", "missing")).To(Succeed())
		})
	})

	Describe("ClearAllLogins", func() {

		It("removes all logins", func() {
			Expect(provider.ClearAllLogins()).To(Succeed())
			Expect(dir).NotTo(BeADirectory())
		})
	})

	Describe("GC", func() {

		It("removes only the expired logins", func() {
			Expect(provider.GC()).To(Succeed())
			Expect(exists("@example-customers.staging.json")).To(BeFalse())
			Expect(exists("@example-customers.production.json")).To(BeTrue())
			Expect(exists("@example-customers.json")).To(BeTrue())
		})

		It("succeeds when there is no login directory", func() {
			Expect(logins.New(filepath.Join(dir, "missing")).GC()).To(Succeed())
		})
	})
})
