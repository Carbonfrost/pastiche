// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package logs_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Carbonfrost/pastiche/pkg/model/history"
	"github.com/Carbonfrost/pastiche/pkg/workspace/logs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
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

func logEntry(ts time.Time, spec []string, url, method string, statusCode int, bodyField map[string]any) string {
	rec := map[string]any{
		"ts":   ts,
		"spec": spec,
		"url":  url,
		"request": map[string]any{
			"method": method,
		},
		"response": map[string]any{
			"status":     "200 OK",
			"statusCode": statusCode,
			"body":       bodyField,
		},
	}
	b, _ := json.Marshal(rec)
	return string(b)
}

var _ = Describe("Read", func() {

	BeforeEach(func() {
		tmpDir := GinkgoT().TempDir()
		origDir, err := os.Getwd()
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { _ = os.Chdir(origDir) })
		Expect(os.Chdir(tmpDir)).To(Succeed())

		os.MkdirAll("logs", 0755)
	})

	writeLogFile := func(filename string, lines ...string) {
		path := filepath.Join("logs", filename)
		content := strings.Join(lines, "\n") + "\n"
		ExpectWithOffset(1, os.WriteFile(path, []byte(content), 0644)).To(Succeed())
	}

	collectAll := func() (entries []*history.LogEntry, errs []error) {
		for entry, err := range logs.New("logs").Read() {
			if err != nil {
				errs = append(errs, err)
			}
			if entry != nil {
				entries = append(entries, entry)
			}
		}
		return
	}

	t := func(year int, month time.Month, day, hour int) time.Time {
		return time.Date(year, month, day, hour, 0, 0, 0, time.UTC)
	}

	textBody := map[string]any{"text": ""}
	jsonBody := func(raw string) map[string]any {
		return map[string]any{"json": json.RawMessage(raw)}
	}

	Context("with an empty log directory", func() {

		It("yields no entries", func() {
			entries, errs := collectAll()
			Expect(errs).To(BeEmpty())
			Expect(entries).To(BeEmpty())
		})
	})

	Context("with a single log file containing one entry", func() {

		BeforeEach(func() {
			writeLogFile("requests.2026-01-15.json",
				logEntry(t(2026, 1, 15, 10), []string{"myservice", "users"}, "https://api.example.com/users", "GET", 200, textBody),
			)
		})

		It("yields the entry", func() {
			entries, errs := collectAll()
			Expect(errs).To(BeEmpty())
			Expect(entries).To(HaveLen(1))
		})

		It("maps spec, URL, method, status code, and timestamp", func() {
			entries, _ := collectAll()
			entry := entries[0]

			Expect(entry.Spec).To(Equal([]string{"myservice", "users"}))
			Expect(entry.URL).To(Equal("https://api.example.com/users"))
			Expect(entry.Request.Method).To(Equal("GET"))
			Expect(entry.Response.StatusCode).To(Equal(200))
			Expect(entry.Timestamp).To(Equal(t(2026, 1, 15, 10)))
		})
	})

	Context("with optional fields present", func() {

		BeforeEach(func() {
			baseURL := "https://api.example.com"
			rec := map[string]any{
				"ts":      t(2026, 1, 15, 10),
				"spec":    []string{"myservice"},
				"url":     "https://api.example.com/v1/users",
				"server":  "prod",
				"baseUrl": baseURL,
				"vars":    map[string]any{"user": "alice"},
				"request": map[string]any{
					"method": "POST",
					"headers": map[string][]string{
						"Content-Type": {"application/json"},
					},
				},
				"response": map[string]any{
					"status":     "201 Created",
					"statusCode": 201,
					"headers": map[string][]string{
						"Location": {"/users/123"},
					},
					"body": textBody,
				},
			}
			b, _ := json.Marshal(rec)
			writeLogFile("requests.2026-01-15.json", string(b))
		})

		It("maps server, baseUrl, vars, request headers, and response headers", func() {
			entries, errs := collectAll()
			Expect(errs).To(BeEmpty())
			Expect(entries).To(HaveLen(1))
			entry := entries[0]

			Expect(entry.Server).To(Equal("prod"))
			Expect(entry.BaseURL).To(PointTo(Equal("https://api.example.com")))
			Expect(entry.Vars).To(HaveKeyWithValue("user", "alice"))
			Expect(entry.Request.Headers).To(HaveKey("Content-Type"))
			Expect(entry.Response.Status).To(Equal("201 Created"))
			Expect(entry.Response.Headers).To(HaveKey("Location"))
		})
	})

	DescribeTable("body content",
		func(bodyField map[string]any, expectedContent string) {
			rec := map[string]any{
				"ts":   t(2026, 1, 15, 10),
				"spec": []string{"svc"},
				"url":  "https://api.example.com",
				"request": map[string]any{
					"method": "GET",
				},
				"response": map[string]any{
					"status":     "200 OK",
					"statusCode": 200,
					"body":       bodyField,
				},
			}
			b, _ := json.Marshal(rec)
			writeLogFile("requests.2026-01-15.json", string(b))

			entries, errs := collectAll()
			Expect(errs).To(BeEmpty())
			Expect(entries).To(HaveLen(1))
			Expect(entries[0].Response.Body.String()).To(Equal(expectedContent))
		},
		Entry("json body", jsonBody(`{"count":3}`), `{"count":3}`),
		Entry("text body", map[string]any{"text": "plain text response"}, "plain text response"),
	)

	Context("with a single log file containing multiple entries", func() {

		BeforeEach(func() {
			writeLogFile("requests.2026-01-15.json",
				logEntry(t(2026, 1, 15, 9), []string{"svc"}, "https://api.example.com/a", "GET", 200, textBody),
				logEntry(t(2026, 1, 15, 10), []string{"svc"}, "https://api.example.com/b", "POST", 201, textBody),
				logEntry(t(2026, 1, 15, 11), []string{"svc"}, "https://api.example.com/c", "DELETE", 204, textBody),
			)
		})

		It("yields entries newest first within the file", func() {
			entries, errs := collectAll()
			Expect(errs).To(BeEmpty())
			Expect(entries).To(HaveLen(3))
			Expect(entries[0].URL).To(Equal("https://api.example.com/c"))
			Expect(entries[1].URL).To(Equal("https://api.example.com/b"))
			Expect(entries[2].URL).To(Equal("https://api.example.com/a"))
		})
	})

	Context("with multiple log files on different dates", func() {

		BeforeEach(func() {
			writeLogFile("requests.2026-01-13.json",
				logEntry(t(2026, 1, 13, 10), []string{"svc"}, "https://api.example.com/a", "GET", 200, textBody),
			)
			writeLogFile("requests.2026-01-14.json",
				logEntry(t(2026, 1, 14, 10), []string{"svc"}, "https://api.example.com/b", "GET", 200, textBody),
			)
			writeLogFile("requests.2026-01-15.json",
				logEntry(t(2026, 1, 15, 10), []string{"svc"}, "https://api.example.com/c", "GET", 200, textBody),
			)
		})

		It("yields entries from the newest file first", func() {
			entries, errs := collectAll()
			Expect(errs).To(BeEmpty())
			Expect(entries).To(HaveLen(3))
			Expect(entries[0].URL).To(Equal("https://api.example.com/c")) // Jan 15
			Expect(entries[1].URL).To(Equal("https://api.example.com/b")) // Jan 14
			Expect(entries[2].URL).To(Equal("https://api.example.com/a")) // Jan 13
		})
	})

	Context("with blank lines in the log file", func() {

		BeforeEach(func() {
			line := logEntry(t(2026, 1, 15, 10), []string{"svc"}, "https://api.example.com", "GET", 200, textBody)
			content := "\n" + line + "\n\n"
			path := filepath.Join("logs", "requests.2026-01-15.json")
			Expect(os.WriteFile(path, []byte(content), 0644)).To(Succeed())
		})

		It("skips blank lines and yields only the valid entry", func() {
			entries, errs := collectAll()
			Expect(errs).To(BeEmpty())
			Expect(entries).To(HaveLen(1))
		})
	})

	Context("with a malformed JSON line among valid entries", func() {

		BeforeEach(func() {
			writeLogFile("requests.2026-01-15.json",
				logEntry(t(2026, 1, 15, 9), []string{"svc"}, "https://api.example.com/a", "GET", 200, textBody),
				`not valid json {{{`,
				logEntry(t(2026, 1, 15, 11), []string{"svc"}, "https://api.example.com/b", "GET", 200, textBody),
			)
		})

		It("surfaces an error for the malformed line", func() {
			_, errs := collectAll()
			Expect(errs).To(HaveLen(1))
		})

		It("still yields the valid entries", func() {
			entries, _ := collectAll()
			Expect(entries).To(HaveLen(2))
		})
	})

	Context("with non-log files in the log directory", func() {

		BeforeEach(func() {
			writeLogFile("requests.2026-01-15.json",
				logEntry(t(2026, 1, 15, 10), []string{"svc"}, "https://api.example.com", "GET", 200, textBody),
			)
			path := filepath.Join("logs", "other-file.txt")
			Expect(os.WriteFile(path, []byte("not a log file"), 0644)).To(Succeed())
		})

		It("ignores files that do not match the log naming pattern", func() {
			entries, errs := collectAll()
			Expect(errs).To(BeEmpty())
			Expect(entries).To(HaveLen(1))
		})
	})
})
