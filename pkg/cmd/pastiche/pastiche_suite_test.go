// Copyright 2022 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package pastiche_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPastiche(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Pastiche Suite")
}
