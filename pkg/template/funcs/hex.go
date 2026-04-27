// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package funcs

import (
	"encoding/hex"
	"fmt"
)

type HexFuncs struct{}

func (f *HexFuncs) Encode(in any) (string, error) {
	b := toBytes(in)
	return hex.EncodeToString(b), nil
}

func (f *HexFuncs) Decode(in any) (string, error) {
	out, err := hex.DecodeString(fmt.Sprint(in))
	return string(out), err
}

func (f *HexFuncs) Self() any {
	return f
}
