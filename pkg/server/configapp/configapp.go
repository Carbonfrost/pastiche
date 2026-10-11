// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package configapp

import (
	"net/http"

	"github.com/Carbonfrost/pastiche/pkg/config"
)

func New() (http.Handler, error) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(config.Schema())
	}), nil
}
