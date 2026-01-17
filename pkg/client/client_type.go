// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package client

import (
	"encoding"
	"fmt"
	"strings"
)

// ClientType enumerates the client types available to Pastiche client
type ClientType int

const (
	ClientTypeUnspecified ClientType = iota
	ClientTypeHTTP
	ClientTypeGRPC
	maxClientType
)

var (
	clientTypeLabels = [maxClientType]string{
		"HTTP",
		"GRPC",
	}
)

// String produces a textual representation of the ClientType
func (t ClientType) String() string {
	return clientTypeLabels[t]
}

// MarshalText provides the textual representation
func (t ClientType) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// UnmarshalText converts the textual representation
func (t *ClientType) UnmarshalText(b []byte) error {
	token := strings.TrimSpace(string(b))
	for k, y := range clientTypeLabels {
		if strings.EqualFold(token, y) {
			*t = ClientType(k)
			return nil
		}
	}
	return fmt.Errorf("unknown client type %q", token)
}

var _ encoding.TextUnmarshaler = (*ClientType)(nil)
