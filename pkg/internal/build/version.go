// Copyright 2022, 2025, 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package build

import (
	"fmt"
	"runtime/debug"
)

const pasticheURL = "https://github.com/Carbonfrost/pastiche"

var Version string

func init() {
	info, _ := debug.ReadBuildInfo()
	Version = info.Main.Version
}

func DefaultUserAgent() string {
	version := Version
	if len(version) == 0 {
		version = "development"
	}
	return fmt.Sprintf("Go-http-client/1.1 (pastiche/%s, +%s)", version, pasticheURL)
}
