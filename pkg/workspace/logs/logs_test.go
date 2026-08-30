// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package logs_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/Carbonfrost/pastiche/pkg/model/history"
	"github.com/Carbonfrost/pastiche/pkg/workspace/logs"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("Log", func() {

	var (
		newEntry = func(ts time.Time, body string) *logs.Entry {
			baseURL := "https://example.com"
			return logs.NewEntry(&history.LogEntry{
				Timestamp: ts,
				Spec:      []string{"example", "resource"},
				URL:       "https://example.com/resource",
				Server:    "prod",
				BaseURL:   &baseURL,
				Vars:      map[string]any{"id": "1"},
				Request: history.Request{
					Method:  "GET",
					Headers: map[string][]string{"Accept": {"application/json"}},
				},
				Response: history.Response{
					Headers:    map[string][]string{"Content-Type": {"application/json"}},
					Status:     "200 OK",
					StatusCode: 200,
					Body:       bytes.NewBufferString(body),
				},
			})
		}
	)

	Describe("FileName", func() {

		It("names the file for the day of the entry", func() {
			l := logs.New("/pastiche/logs")
			ts := time.Date(2026, time.August, 28, 13, 15, 0, 0, time.UTC)

			Expect(l.FileName(ts)).To(Equal(filepath.Join("/pastiche/logs", "requests.2026-08-28.json")))
		})
	})

	Describe("Append", func() {

		var (
			dir string
			ts  = time.Date(2026, time.August, 28, 13, 15, 0, 0, time.UTC)
		)

		BeforeEach(func() {
			dir = GinkgoT().TempDir()
		})

		It("writes the entry to the file for the day of the entry", func() {
			l := logs.New(dir)
			Expect(l.Append(newEntry(ts, `{"hello":"world"}`))).To(Succeed())

			data, err := os.ReadFile(filepath.Join(dir, "requests.2026-08-28.json"))
			Expect(err).NotTo(HaveOccurred())

			var actual map[string]any
			Expect(json.Unmarshal(data, &actual)).To(Succeed())
			Expect(actual).To(HaveKeyWithValue("url", "https://example.com/resource"))
			Expect(actual).To(HaveKeyWithValue("server", "prod"))
			Expect(actual).To(HaveKeyWithValue("baseUrl", "https://example.com"))
			Expect(actual).To(HaveKeyWithValue("spec", ConsistOf("example", "resource")))
			Expect(actual).To(HaveKeyWithValue("vars", HaveKeyWithValue("id", "1")))
			Expect(actual).To(HaveKeyWithValue("request", HaveKeyWithValue("method", "GET")))
			Expect(actual).To(HaveKeyWithValue("response", HaveKeyWithValue("statusCode", float64(200))))
		})

		It("appends each entry on its own line", func() {
			l := logs.New(dir)
			Expect(l.Append(newEntry(ts, `{"hello":"world"}`))).To(Succeed())
			Expect(l.Append(newEntry(ts, `{"hello":"world"}`))).To(Succeed())

			data, err := os.ReadFile(filepath.Join(dir, "requests.2026-08-28.json"))
			Expect(err).NotTo(HaveOccurred())
			Expect(bytes.Count(data, []byte("\n"))).To(Equal(2))
		})

		It("uses a separate file per day", func() {
			l := logs.New(dir)
			Expect(l.Append(newEntry(ts, `{}`))).To(Succeed())
			Expect(l.Append(newEntry(ts.AddDate(0, 0, 1), `{}`))).To(Succeed())

			names, err := filepath.Glob(filepath.Join(dir, "*.json"))
			Expect(err).NotTo(HaveOccurred())
			Expect(names).To(ConsistOf(
				filepath.Join(dir, "requests.2026-08-28.json"),
				filepath.Join(dir, "requests.2026-08-29.json"),
			))
		})

		It("returns an error when the log directory does not exist", func() {
			l := logs.New(filepath.Join(dir, "nope"))
			Expect(l.Append(newEntry(ts, `{}`))).NotTo(Succeed())
		})
	})
})

var _ = Describe("Entry", func() {

	DescribeTable("response body representation",
		func(body string, expected types.GomegaMatcher) {
			e := logs.NewEntry(&history.LogEntry{
				Response: history.Response{
					Body: bytes.NewBufferString(body),
				},
			})

			data, err := json.Marshal(e)
			Expect(err).NotTo(HaveOccurred())

			var actual map[string]any
			Expect(json.Unmarshal(data, &actual)).To(Succeed())
			Expect(actual["response"]).To(HaveKeyWithValue("body", expected))
		},
		Entry(
			"encodes valid JSON verbatim",
			`{"hello":"world"}`,
			HaveKeyWithValue("json", HaveKeyWithValue("hello", "world")),
		),
		Entry(
			"encodes other content as text",
			"hello, world",
			HaveKeyWithValue("text", "hello, world"),
		),
	)
})
