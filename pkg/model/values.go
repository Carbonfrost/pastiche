// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package model

import (
	"net/http"
	"net/url"
)

// MergeMode indicates how a Value's values are combined during a reduce.
type MergeMode int

// NOTE As an optimization, merge modes must match config.MergeMode

const (
	// MergeReplace replaces the existing values with the incoming ones. This
	// is the default when no merge mode is specified.
	MergeReplace MergeMode = iota

	// MergeAppend appends the incoming values to the existing ones.
	MergeAppend
)

// Values is a list of named values. It mirrors config.Values within the model
// and provides the reduce semantics used when resolving values across the
// service, server, resource, and endpoint layers.
type Values []Value

// Value is a single named value or list of values within Values.
type Value struct {
	Name     string
	Value    string
	Values   []string
	Optional bool
	Merge    MergeMode
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
			combined := append([]string{}, merged.effectiveValues()...)
			combined = append(combined, e.effectiveValues()...)
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

// reduceValues adapts Reduce to the reducer signature used by locate.
func reduceValues(x, y Values) Values {
	return x.Reduce(y)
}

// effectiveValues returns the values represented by the entry, preferring the
// explicit Values list over the singular Value.
func (v Value) effectiveValues() []string {
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
		vals := e.effectiveValues()
		if e.Optional && len(vals) == 0 {
			continue
		}
		m[e.Name] = append(m[e.Name], vals...)
	}
	return m
}
