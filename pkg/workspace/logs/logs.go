// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package logs provides the JSON representation of history log entries and
// writes them to the log files within the workspace.
package logs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Carbonfrost/pastiche/pkg/model/history"
	"github.com/Carbonfrost/pastiche/pkg/workspace"
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

// FromContext creates a log which writes its entries to the log directory
// of the workspace in the context.
func FromContext(ctx context.Context) *Log {
	return New(workspace.FromContext(ctx).LogDir())
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

var _ json.Marshaler = (*ResponseBody)(nil)
