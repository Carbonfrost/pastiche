// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package funcs

import (
	"encoding/base32"
	"fmt"
)

type Base32Funcs struct{}

func (f *Base32Funcs) Encode(in any) (string, error) {
	b := toBytes(in)
	return base32.StdEncoding.EncodeToString(b), nil
}

func (f *Base32Funcs) Decode(in any) (string, error) {
	out, err := base32.StdEncoding.DecodeString(fmt.Sprint(in))
	return string(out), err
}

func (f *Base32Funcs) Self() any {
	return f
}
