// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"embed"
	"fmt"
	"maps"
	"os"
	"slices"
	"sync"
)

var (
	//go:embed builtin
	builtin embed.FS

	builtinCache = sync.OnceValue(loadBuiltins)
)

func Builtin(name string) Service {
	c := builtinCache()
	return c[name]
}

func loadBuiltins() map[string]Service {
	c := map[string]Service{}
	maps.Copy(c, safelyLoadBuiltins("builtin/3rd_party.yml"))
	maps.Copy(c, safelyLoadBuiltins("builtin/pastiche.yml"))

	return c
}

func ExampleHTTPBinorg() Service {
	return Builtin("httpbin")
}

func Builtins() []Service {
	return slices.Collect(maps.Values(builtinCache()))
}

func safelyLoadBuiltins(filename string) map[string]Service {
	c := map[string]Service{}
	fileP, err := LoadFile(builtin, filename)
	if err == nil {
		for _, s := range fileP.Services {
			c[s.Name] = s
		}
	} else {
		fmt.Fprintf(os.Stderr, "Problem loading built-ins %s: %v\n", filename, err)
	}
	return c
}
