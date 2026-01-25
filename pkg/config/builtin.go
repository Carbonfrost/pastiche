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

	loadBuiltins sync.Once
	cache        cacheData
)

type cacheData struct {
	files  []*File
	lookup map[string]Service
}

func Builtin(name string) Service {
	_, c := builtinCache()
	return c[name]
}

func builtinCache() ([]*File, map[string]Service) {
	loadBuiltins.Do(func() {
		files := slices.Concat(
			safelyLoadBuiltins("builtin/3rd_party.yml"),
			safelyLoadBuiltins("builtin/pastiche.yml"),
		)
		cache = cacheData{
			files:  files,
			lookup: extractLookup(files),
		}
	})
	return cache.files, cache.lookup
}

func extractLookup(files []*File) map[string]Service {
	lookup := map[string]Service{}
	for _, file := range files {
		for _, s := range file.Services {
			lookup[s.Name] = s
		}
	}
	return lookup
}

func ExampleHTTPBinorg() Service {
	return Builtin("httpbin")
}

func Builtins() []Service {
	_, c := builtinCache()
	return slices.Collect(maps.Values(c))
}

func BuiltinFiles() []*File {
	f, _ := builtinCache()
	return f
}

func safelyLoadBuiltins(filename string) []*File {
	fileP, err := LoadFile(builtin, filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Problem loading built-ins %s: %v\n", filename, err)
		return nil
	}
	return []*File{fileP}
}
