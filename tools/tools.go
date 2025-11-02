// Copyright 2022 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//go:build tools

package tools

import (
	_ "github.com/onsi/ginkgo/v2"
	_ "honnef.co/go/tools/cmd/staticcheck"
)
