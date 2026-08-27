// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package history

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/Carbonfrost/joe-cli-http/httpclient"
	"github.com/Carbonfrost/pastiche/pkg/model"
)

// LogEntry represents an entry in the history log, capturing the
// request and response details for a service invocation.
type LogEntry struct {
	Timestamp time.Time
	Spec      []string
	URL       string
	Server    string
	BaseURL   *string
	Vars      map[string]any
	Request   Request
	Response  Response
}

// Request captures the HTTP request details for a history log entry.
type Request struct {
	Method  string
	Headers map[string][]string
}

// Response captures the HTTP response details for a history log entry.
type Response struct {
	Headers    map[string][]string
	Status     string
	StatusCode int
	Body       *bytes.Buffer
}

// NewHistoryLogEntry creates a new HistoryLogEntry from model types.
// It returns the entry and an io.Writer for capturing the response body as it streams in.
func NewLogEntry(spec []string, server string, baseURL *url.URL, req *model.Request, resp *httpclient.Response) (*LogEntry, io.Writer) {
	var vars map[string]any
	if req != nil {
		vars = req.Vars
	}

	var responseBody bytes.Buffer
	entry := &LogEntry{
		Timestamp: time.Now(),
		URL:       fmt.Sprint(resp.Request.URL),
		Spec:      spec,
		Server:    server,
		Response: Response{
			Headers:    resp.Header,
			Status:     resp.Status,
			StatusCode: resp.StatusCode,
			Body:       &responseBody,
		},
		Request: Request{
			Headers: resp.Request.Header,
			Method:  resp.Request.Method,
		},
		Vars:    vars,
		BaseURL: historyBaseURL(baseURL),
	}
	return entry, &responseBody
}

func historyBaseURL(u *url.URL) *string {
	if u == nil {
		return nil
	}
	s := u.String()
	return &s
}
