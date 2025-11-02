// Copyright 2025 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package log

import (
	"fmt"
	"os"
)

func Warn(v ...any) {
	fmt.Fprintln(os.Stderr, v...)
}

func Warnf(format string, v ...any) {
	fmt.Fprintf(os.Stderr, format, v...)
	fmt.Fprintln(os.Stderr)
}
