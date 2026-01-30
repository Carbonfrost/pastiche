// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// FetchOptions represents common fetch options
type FetchOptions struct {
	Method  string            `json:"method,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"`
}

// FetchCall represents the parsed fetch call
type FetchCall struct {
	URL     string
	Options FetchOptions
}

func ParseJSFetchCall(s string) (*FetchCall, error) {
	return parseFetchExpression(s)
}

func (f FetchCall) ToEndpoint() *Endpoint {
	var body any
	var decoded any

	body = f.Options.Body
	headers := http.Header{}

	// Try interpreting body as JSON.
	if err := json.Unmarshal([]byte(f.Options.Body), &decoded); err == nil {
		body = decoded
	}
	for k, v := range f.Options.Headers {
		headers.Set(k, v)
	}

	return &Endpoint{
		Method:  f.Options.Method,
		Body:    body,
		Headers: headers,
	}
}

// parseFetchExpression parses a very simple fetch("url", { ... }) expression
func parseFetchExpression(input string) (*FetchCall, error) {
	input = strings.TrimSpace(input)

	// Basic fetch(...) shape check
	if !strings.HasPrefix(input, "fetch(") || !strings.HasSuffix(input, ")") {
		return nil, errors.New("expression is not a fetch(...) call")
	}

	// Extract arguments inside fetch(...)
	args := strings.TrimSuffix(strings.TrimPrefix(input, "fetch("), ")")
	urlPart, optionsPart, _ := strings.Cut(args, ",")

	urlPart = strings.TrimSpace(urlPart)
	url, err := strconv.Unquote(urlPart)
	if err != nil || url == urlPart {
		return nil, errors.New("fetch URL must be a quoted string")
	}

	// Parse options object
	optionsPart = strings.TrimSpace(optionsPart)
	if !strings.HasPrefix(optionsPart, "{") || !strings.HasSuffix(optionsPart, "}") {
		return nil, errors.New("fetch options must be an object literal")
	}

	var opts FetchOptions
	if err := json.Unmarshal([]byte(optionsPart), &opts); err != nil {
		return nil, fmt.Errorf("invalid options JSON: %w", err)
	}

	return &FetchCall{
		URL:     url,
		Options: opts,
	}, nil
}
