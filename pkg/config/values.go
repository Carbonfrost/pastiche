// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

// MergeMode indicates how a Value's values are combined during a reduce.
type MergeMode int

const (
	// MergeReplace replaces the existing values with the incoming ones. This
	// is the default when no merge mode is specified.
	MergeReplace MergeMode = iota

	// MergeAppend appends the incoming values to the existing ones.
	MergeAppend
)

// Values is a list of named values. In addition to the canonical list
// representation, it supports the same abbreviated representation that Header
// does: a map from name to a value or to a list of values.
//
//	# canonical list form
//	- name: user_id
//	  value: 1        # implies a single value
//	  values:         # alternatively, explicitly name values
//	    - 1
//	    - 2
//	  optional: true  # the name is omitted from ToHeader/ToURLValues when empty
//	  merge: APPEND    # append or replace when reducing (default REPLACE)
//
//	# abbreviated map form
//	user_id: 1
//	scope: [read, write]
type Values []Value

// Value is a single named value or list of values within Values and its
// configuration for handling merging
type Value struct {
	Name     string    `json:"name,omitempty"`
	Value    string    `json:"value,omitempty"`
	Values   []string  `json:"values,omitempty"`
	Optional bool      `json:"optional,omitempty"`
	Merge    MergeMode `json:"merge,omitempty"`
}

func (m MergeMode) String() string {
	if m == MergeAppend {
		return "APPEND"
	}
	return "REPLACE"
}

// MarshalText provides the textual representation
func (m MergeMode) MarshalText() ([]byte, error) {
	return []byte(m.String()), nil
}

// UnmarshalText converts the textual representation
func (m *MergeMode) UnmarshalText(b []byte) error {
	s := string(b)
	switch strings.ToUpper(s) {
	case "", "REPLACE":
		*m = MergeReplace
	case "APPEND":
		*m = MergeAppend
	default:
		return fmt.Errorf("unknown merge mode: %q", s)
	}
	return nil
}

func (m MergeMode) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.String())
}

func (m *MergeMode) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	return m.UnmarshalText([]byte(s))
}

// ValuesFromMap creates Values from a map
func ValuesFromMap(values map[string][]string) Values {
	result := make(Values, len(values))

	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)

	for i, k := range names {
		v := values[k]
		if len(v) == 1 {
			result[i] = Value{Name: k, Value: v[0]}
		} else {
			result[i] = Value{Name: k, Values: v}
		}
	}
	return result
}

func (v *Value) UnmarshalJSON(data []byte) error {
	var raw struct {
		Name     string    `json:"name"`
		Value    any       `json:"value"`
		Values   []any     `json:"values"`
		Optional bool      `json:"optional"`
		Merge    MergeMode `json:"merge"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	v.Name = raw.Name
	v.Optional = raw.Optional
	v.Merge = raw.Merge

	if raw.Value != nil {
		s, err := coerceString(raw.Value)
		if err != nil {
			return err
		}
		v.Value = s
	}
	if raw.Values != nil {
		v.Values = make([]string, len(raw.Values))
		for i, e := range raw.Values {
			s, err := coerceString(e)
			if err != nil {
				return err
			}
			v.Values[i] = s
		}
	}
	return nil
}

// MarshalJSON writes the respresentation of Values. The method optimizes for
// the best abbreviated representation where possible; otherwise, it uses
// the full representation.
func (v Values) MarshalJSON() ([]byte, error) {
	if !v.canAbbreviate() {
		type values Values // prevent recursion into this method
		return json.Marshal(values(v))
	}

	if v.hasMultiValues() {
		m := make(map[string][]string, len(v))
		for _, e := range v {
			m[e.Name] = e.actualValues()
		}
		return json.Marshal(m)
	}

	m := make(map[string]string, len(v))
	for _, e := range v {
		m[e.Name] = e.Value
	}
	return json.Marshal(m)
}

func (v Values) canAbbreviate() bool {
	seen := make(map[string]struct{}, len(v))
	for _, e := range v {
		if e.Name == "" || e.Optional || e.Merge != MergeReplace {
			return false
		}
		if e.Value != "" && len(e.Values) > 0 {
			return false
		}
		if _, dup := seen[e.Name]; dup {
			return false
		}
		seen[e.Name] = struct{}{}
	}
	return true
}

func (v Values) hasMultiValues() bool {
	for _, e := range v {
		if len(e.Values) > 0 {
			return true
		}
	}
	return false
}

func (v *Values) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)

	// Abbreviated map form: name -> value or name -> [value...]
	if len(trimmed) > 0 && trimmed[0] == '{' {
		m := map[string]any{}
		if err := json.Unmarshal(trimmed, &m); err != nil {
			return err
		}

		names := make([]string, 0, len(m))
		for name := range m {
			names = append(names, name)
		}
		sort.Strings(names)

		result := make(Values, 0, len(names))
		for _, name := range names {
			val := Value{Name: name}
			switch raw := m[name].(type) {
			case []any:
				val.Values = make([]string, len(raw))
				for i, e := range raw {
					s, err := coerceString(e)
					if err != nil {
						return err
					}
					val.Values[i] = s
				}
			default:
				s, err := coerceString(raw)
				if err != nil {
					return err
				}
				val.Value = s
			}
			result = append(result, val)
		}
		*v = result
		return nil
	}

	var list []Value
	if err := json.Unmarshal(trimmed, &list); err != nil {
		return err
	}
	*v = list
	return nil
}

// ToHeader converts the values into a net/http.Header.
func (v Values) ToHeader() http.Header {
	return http.Header(v.toMap())
}

// ToURLValues converts the values into a url.Values.
func (v Values) ToURLValues() url.Values {
	return url.Values(v.toMap())
}

// Reduce combines the receiver with the more-specific next values, honoring
// each incoming value's merge mode. A value with the same name is appended to
// or replaces the existing one; a value with a new name is added.
func (v Values) Reduce(next Values) Values {
	result := make(Values, len(v))
	copy(result, v)

	index := make(map[string]int, len(result))
	for i, e := range result {
		index[e.Name] = i
	}

	for _, e := range next {
		i, ok := index[e.Name]
		switch {
		case ok && e.Merge == MergeAppend:
			merged := result[i]
			combined := append([]string{}, merged.actualValues()...)
			combined = append(combined, e.actualValues()...)
			merged.Value = ""
			merged.Values = combined
			merged.Optional = e.Optional
			merged.Merge = e.Merge
			result[i] = merged
		case ok:
			result[i] = e
		default:
			index[e.Name] = len(result)
			result = append(result, e)
		}
	}
	return result
}

func (v Value) actualValues() []string {
	// Prefer the list over the singular item
	if len(v.Values) > 0 {
		return v.Values
	}
	if v.Value != "" {
		return []string{v.Value}
	}
	return nil
}

func (v Values) toMap() map[string][]string {
	m := make(map[string][]string, len(v))
	for _, e := range v {
		vals := e.actualValues()
		if e.Optional && len(vals) == 0 {
			continue
		}
		m[e.Name] = append(m[e.Name], vals...)
	}
	return m
}

func coerceString(v any) (string, error) {
	switch val := v.(type) {
	case nil:
		return "", nil
	case string:
		return val, nil
	case bool:
		return strconv.FormatBool(val), nil
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64), nil
	case json.Number:
		return val.String(), nil
	default:
		return "", fmt.Errorf("unexpected type in config.Values: %T", val)
	}
}

var (
	_ json.Unmarshaler = (*Values)(nil)
	_ json.Unmarshaler = (*Value)(nil)
	_ json.Unmarshaler = (*MergeMode)(nil)
	_ json.Marshaler   = Values(nil)
	_ json.Marshaler   = MergeMode(0)
)
