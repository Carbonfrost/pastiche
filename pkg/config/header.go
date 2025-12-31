// Copyright 2025 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package config

import (
	"encoding/json"
	"fmt"
)

// Header represents the key-value pairs in an HTTP header.
type Header map[string][]string

// Form represents the key-value pairs in an encoded form.
type Form map[string][]string

func (h *Header) UnmarshalJSON(d []byte) error {
	head, err := makeHeader(*h, d)
	if err != nil {
		return err
	}
	*h = head
	return nil
}

func (f *Form) UnmarshalJSON(d []byte) error {
	head, err := makeHeader(*f, d)
	if err != nil {
		return err
	}
	*f = head
	return nil
}

func makeHeader[H ~map[string][]string](head H, data []byte) (map[string][]string, error) {
	values := map[string]any{}
	err := json.Unmarshal(data, &values)
	if err != nil {
		return nil, err
	}
	if head == nil {
		head = map[string][]string{}
	}
	for k, v := range values {
		switch val := v.(type) {
		case string:
			head[k] = []string{val}
		case []any:
			strs := make([]string, len(val))
			for i := range val {
				strs[i] = val[i].(string)
			}
			head[k] = strs
		default:
			return nil, fmt.Errorf("unexpected type in %T: %T", head, val)
		}
	}
	return head, nil
}

var (
	_ json.Unmarshaler = (*Header)(nil)
	_ json.Unmarshaler = (*Form)(nil)
)
