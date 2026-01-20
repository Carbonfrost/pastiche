// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package funcs defines some functions to be used in templates
package funcs

func AddTo(f map[string]any) {
	f["base64"] = &Base64Funcs{}
}
