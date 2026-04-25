// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package metaapi

import (
	"encoding/json"
	"net/http"

	"github.com/Carbonfrost/pastiche/pkg/model"
)

// New retrieves the meta API handler for the given model
func New(mo *model.Model) (http.Handler, error) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(mo)
	}), nil
}
