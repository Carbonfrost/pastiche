// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package logs provides the JSON representation of history log entries and
// writes them to the log files within the workspace.
package logs

import (
	"bufio"
	"bytes"
	"cmp"
	"encoding/json"
	"fmt"
	"iter"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/Carbonfrost/pastiche/pkg/model/history"
)

type (
	// Entry provides the JSON representation of a history log entry.
	Entry struct {
		Timestamp time.Time      `json:"ts"`
		Spec      []string       `json:"spec"`
		URL       string         `json:"url"`
		Server    string         `json:"server,omitempty"`
		Response  Response       `json:"response"`
		Request   Request        `json:"request"`
		Vars      map[string]any `json:"vars,omitempty"`
		BaseURL   *string        `json:"baseUrl"`
	}

	// Response provides the JSON representation of the response within
	// a history log entry.
	Response struct {
		Headers    map[string][]string `json:"headers,omitempty"`
		Status     string              `json:"status"`
		StatusCode int                 `json:"statusCode"`
		Body       *ResponseBody       `json:"body"`
	}

	// ResponseBody provides the JSON representation of the response body,
	// which is encoded either as JSON or as text depending upon its contents.
	ResponseBody struct {
		buffer *bytes.Buffer
	}

	// Request provides the JSON representation of the request within
	// a history log entry.
	Request struct {
		Method  string              `json:"method"`
		Headers map[string][]string `json:"headers,omitempty"`
	}
)

// Log appends history log entries to the log files contained in a directory.
type Log struct {
	dir string
}

// New creates a log which writes its entries to the given directory.
func New(dir string) *Log {
	return &Log{dir: dir}
}

// NewEntry creates the JSON representation of the given history log entry.
func NewEntry(e *history.LogEntry) *Entry {
	return &Entry{
		Timestamp: e.Timestamp,
		Spec:      e.Spec,
		URL:       e.URL,
		Server:    e.Server,
		Response: Response{
			Headers:    e.Response.Headers,
			Status:     e.Response.Status,
			StatusCode: e.Response.StatusCode,
			Body:       &ResponseBody{e.Response.Body},
		},
		Request: Request{
			Method:  e.Request.Method,
			Headers: e.Request.Headers,
		},
		Vars:    e.Vars,
		BaseURL: e.BaseURL,
	}
}

// Append writes the entry to the log file which corresponds to the day on
// which the entry occurred.
func (l *Log) Append(e *Entry) error {
	f, err := os.OpenFile(l.FileName(e.Timestamp), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	logLines, err := json.Marshal(e)
	if err != nil {
		return err
	}

	if _, err := f.Write(logLines); err != nil {
		return err
	}

	_, _ = f.Write([]byte("\n"))
	return nil
}

// FileName obtains the name of the log file which contains the entries
// for the given time.
func (l *Log) FileName(t time.Time) string {
	return filepath.Join(l.dir, fmt.Sprintf("requests.%s.json", t.Format("2006-01-02")))
}

func (b ResponseBody) MarshalJSON() ([]byte, error) {
	if json.Valid(b.buffer.Bytes()) {
		return json.Marshal(map[string]any{
			"json": json.RawMessage(b.buffer.Bytes()),
		})
	}

	return json.Marshal(map[string]any{
		"text": b.buffer.String(),
	})
}

func (b *ResponseBody) UnmarshalJSON(data []byte) error {
	var wrapper map[string]json.RawMessage
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return err
	}
	if raw, ok := wrapper["json"]; ok {
		b.buffer = bytes.NewBuffer(raw)
		return nil
	}
	if text, ok := wrapper["text"]; ok {
		var s string
		if err := json.Unmarshal(text, &s); err != nil {
			return err
		}
		b.buffer = bytes.NewBufferString(s)
		return nil
	}
	return nil
}

func (e Entry) toEntry() *history.LogEntry {
	return &history.LogEntry{
		Timestamp: e.Timestamp,
		Spec:      e.Spec,
		URL:       e.URL,
		Server:    e.Server,
		BaseURL:   e.BaseURL,
		Vars:      e.Vars,
		Request: history.Request{
			Method:  e.Request.Method,
			Headers: e.Request.Headers,
		},
		Response: history.Response{
			Headers:    e.Response.Headers,
			Status:     e.Response.Status,
			StatusCode: e.Response.StatusCode,
			Body:       e.Response.Body.buffer,
		},
	}
}

// Clear clears the logs
func (l *Log) Clear() error {
	stat, err := os.Stat(l.dir)
	if os.IsNotExist(err) {
		return nil
	}
	if stat.IsDir() {
		return os.RemoveAll(l.dir)
	}

	return os.MkdirAll(l.dir, 0o755)
}

// Read returns an iterator over history log entries in reverse chronological order.
// Within each day's log file, entries are yielded most-recent first.
func (l *Log) Read() iter.Seq2[*history.LogEntry, error] {
	return func(yield func(*history.LogEntry, error) bool) {
		matches, err := filepath.Glob(filepath.Join(l.dir, "requests.*.json"))
		if err != nil {
			yield(nil, err)
			return
		}

		// Reverse chronological (and lexicographical - one and the same) order
		slices.SortFunc(matches, func(a, b string) int {
			return cmp.Compare(filepath.Base(b), filepath.Base(a))
		})

		for _, path := range matches {
			if !readLogFile(path, yield) {
				return
			}
		}
	}
}

func readLogFile(path string, yield func(*history.LogEntry, error) bool) bool {
	f, err := os.Open(path)
	if err != nil {
		return yield(nil, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 10*1024*1024)

	var lines [][]byte
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(strings.TrimSpace(string(line))) == 0 {
			continue
		}
		lines = append(lines, append([]byte(nil), line...))
	}
	if err := scanner.Err(); err != nil {
		return yield(nil, err)
	}

	for i := len(lines) - 1; i >= 0; i-- {
		var rec Entry
		if err := json.Unmarshal(lines[i], &rec); err != nil {
			if !yield(nil, err) {
				return false
			}
			continue
		}
		if !yield(rec.toEntry(), nil) {
			return false
		}
	}
	return true
}

var _ json.Marshaler = (*ResponseBody)(nil)
