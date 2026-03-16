// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"encoding"
	"fmt"
	"strings"
)

// Type enumerates the client types available to Pastiche client
type Type int

const (
	TypeUnspecified Type = iota
	TypeHTTP
	TypeGRPC
	maxType
)

var (
	typeLabels = [maxType]string{
		"HTTP",
		"GRPC",
	}
)

// String produces a textual representation of the Type
func (t Type) String() string {
	return typeLabels[t]
}

// MarshalText provides the textual representation
func (t Type) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// UnmarshalText converts the textual representation
func (t *Type) UnmarshalText(b []byte) error {
	token := strings.TrimSpace(string(b))
	for k, y := range typeLabels {
		if strings.EqualFold(token, y) {
			*t = Type(k)
			return nil
		}
	}
	return fmt.Errorf("unknown client type %q", token)
}

var _ encoding.TextUnmarshaler = (*Type)(nil)
