// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
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

	// Split on first comma only (url, options)
	parts := splitOnFirstComma(args)
	if len(parts) != 2 {
		return nil, errors.New("fetch call must have exactly two arguments")
	}

	// Parse URL
	urlPart := strings.TrimSpace(parts[0])
	if !isQuotedString(urlPart) {
		return nil, errors.New("fetch URL must be a quoted string")
	}
	url, _ := strconv.Unquote(urlPart)

	// Parse options object
	optionsPart := strings.TrimSpace(parts[1])
	if !strings.HasPrefix(optionsPart, "{") || !strings.HasSuffix(optionsPart, "}") {
		return nil, errors.New("fetch options must be an object literal")
	}

	// Detect unquoted keys (error if any found)
	if hasUnquotedKeys(optionsPart) {
		return nil, errors.New("options object contains unquoted keys")
	}

	// Decode JSON
	var opts FetchOptions
	if err := json.Unmarshal([]byte(optionsPart), &opts); err != nil {
		return nil, fmt.Errorf("invalid options JSON: %w", err)
	}

	return &FetchCall{
		URL:     url,
		Options: opts,
	}, nil
}

// splitOnFirstComma splits a string into two parts on the first comma not inside braces
func splitOnFirstComma(s string) []string {
	depth := 0
	for i, r := range s {
		switch r {
		case '{':
			depth++
		case '}':
			depth--
		case ',':
			if depth == 0 {
				return []string{s[:i], s[i+1:]}
			}
		}
	}
	return []string{s}
}

func isQuotedString(s string) bool {
	return len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"'
}

// hasUnquotedKeys detects JS-style keys like { method: "POST" }
func hasUnquotedKeys(obj string) bool {
	// Matches: { key: ... } or , key:
	unquotedKeyRegex := regexp.MustCompile(`[{,]\s*[a-zA-Z_][a-zA-Z0-9_]*\s*:`)
	return unquotedKeyRegex.MatchString(obj)
}
