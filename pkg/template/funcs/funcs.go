// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package funcs defines some functions to be used in templates
package funcs

import (
	"fmt"
	"os"
	"text/template"

	"github.com/Carbonfrost/joe-cli-http/httpclient/expr"
)

type Expander = expr.Expander

func AddTo(data map[string]any) {
	data["base64"] = &Base64Funcs{}
	data["term"] = NewTermFuncs()
}

func AddToFuncs(funcMap template.FuncMap) {
	funcMap["env"] = os.Getenv
}

func AddVarResolver(funcMap template.FuncMap, e Expander) {
	funcMap["var"] = resolveVar(e)
}

func resolveVar(exp Expander) func(...string) (any, error) {
	return func(vars ...string) (any, error) {
		if len(vars) == 0 {
			return "", fmt.Errorf("var/n requires at least one var name")
		}
		for i, v := range vars {
			result := exp(v)
			if result != nil {
				return result, nil
			}
			if i > 0 && i == len(vars)-1 {
				// Last value is a literal
				return v, nil
			}
		}
		return "", fmt.Errorf("var not found: %q", vars[0])
	}
}
