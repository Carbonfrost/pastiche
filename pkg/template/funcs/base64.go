// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package funcs

import (
	"encoding/base64"
	"fmt"
)

type Base64Funcs struct{}

func (f *Base64Funcs) Encode(in any) (string, error) {
	b := toBytes(in)
	return base64.StdEncoding.EncodeToString(b), nil
}

func (f *Base64Funcs) Decode(in any) (string, error) {
	out, err := base64.StdEncoding.DecodeString(fmt.Sprint(in))
	return string(out), err
}

func toBytes(in any) []byte {
	if in == nil {
		return []byte{}
	}
	if s, ok := in.([]byte); ok {
		return s
	}
	if s, ok := in.(interface{ Bytes() []byte }); ok {
		return s.Bytes()
	}
	if s, ok := in.(string); ok {
		return []byte(s)
	}
	return fmt.Append(nil, in)
}
