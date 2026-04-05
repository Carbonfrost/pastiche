// Copyright 2025, 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"path"
	"path/filepath"
	"strings"

	"sigs.k8s.io/yaml"
)

type Config struct {
	Services []Service
}

type sourcer struct {
	f fs.FS
}

type unmarshaler func([]byte, any) error

var unmarshalers = map[string]unmarshaler{
	".json":     json.Unmarshal,
	".yaml":     unmarshalYaml,
	".yamlvars": unmarshalYamlVarSet,
	".yml":      unmarshalYaml,
	".ymlvars":  unmarshalYamlVarSet,
}

var ErrUnsupportedFileFormat = errors.New("unsupported file format")

// LoadFile loads the given file from the file system and name
func LoadFile(f fs.FS, filename string) (*File, error) {
	if unmarshal, ok := unmarshalers[filepath.Ext(filename)]; ok {
		data, err := fs.ReadFile(f, filename)
		if err != nil {
			return nil, err
		}

		result := new(File)
		result.SetName(filename)
		if err := unmarshal(data, result); err != nil {
			return nil, err
		}

		if len(result.Services) > 0 && result.Service != nil {
			return nil, fmt.Errorf("must contain either service definition or services list, but not both")
		}

		src := sourcer{f: f}
		err = src.source(filename, result)
		if err != nil {
			return nil, err
		}

		return result, nil
	}

	return nil, fmt.Errorf("load file %s: %w", filename, ErrUnsupportedFileFormat)
}

func (s sourcer) source(basefilename string, v any) error {
	var file string

	switch a := v.(type) {
	case *File:
		if a.Service == nil {
			return sources(s, basefilename, a.Services)
		}
		return s.source(basefilename, a.Service)
	case *Service:
		if a == nil {
			return nil
		}
		file = a.Source
	case *Server:
		if a == nil {
			return nil
		}
		file = a.Source
	case *Resource:
		if a == nil {
			return nil
		}
		file = a.Source
	case *Endpoint:
		if a == nil {
			return nil
		}
		file = a.Source
	}

	if file == "" {
		file = basefilename
	} else {
		resolvedFile := path.Join(path.Dir(basefilename), file)
		reader, err := s.f.Open(resolvedFile)
		if err != nil {
			return err
		}

		data, err := io.ReadAll(reader)
		if err != nil {
			return err
		}
		unmarshal, ok := unmarshalers[filepath.Ext(file)]
		if !ok {
			return fmt.Errorf("%s: %w", file, ErrUnsupportedFileFormat)
		}
		err = unmarshal(data, v)
		if err != nil {
			return fmt.Errorf("%s: %w", file, err)
		}

		// Further references in recursive includes use this new base file
		file = resolvedFile
	}

	switch a := v.(type) {
	case *Service:
		err := sources(s, file, a.Servers)
		if err != nil {
			return err
		}
		if a.Client != nil && a.Client.GRPC != nil {
			fixRelative(basefilename, &a.Client.GRPC.ProtoSet)
		}
		a.Output = fixOutputsRelative(basefilename, a.Output)
		return sources(s, file, a.Resources)

	case *Server:
		a.Output = fixOutputsRelative(basefilename, a.Output)

	case *Resource:
		err := sources(s, file, a.Resources)
		if err != nil {
			return err
		}
		a.Output = fixOutputsRelative(basefilename, a.Output)
		return s.sources(file, a.Get, a.Put, a.Post, a.Delete, a.Options, a.Head, a.Trace, a.Patch, a.Query)

	case *Endpoint:
		// Nothing to do for endpoints
	}
	return nil
}

func (s sourcer) sources(basefilename string, values ...any) error {
	for _, v := range values {
		err := s.source(basefilename, v)
		if err != nil {
			return err
		}
	}
	return nil
}

func unmarshalYaml(data []byte, v any) error {
	return yaml.UnmarshalStrict(preprocessYAML(data), v)
}

func unmarshalYamlVarSet(data []byte, v any) error {
	return unmarshalYaml(data, &v.(*File).VarSets)
}

func sources[V any](s sourcer, basefilename string, values []V) error {
	for i, v := range values {
		err := s.source(basefilename, &v)
		values[i] = v
		if err != nil {
			return err
		}
	}
	return nil
}

func fixRelative(basefilename string, pathStr *string) {
	if pathStr == nil || *pathStr == "" || strings.HasPrefix(*pathStr, "/") {
		return
	}
	resolvedFile := path.Join(path.Dir(basefilename), *pathStr)
	*pathStr = resolvedFile
}

func fixOutputsRelative(basefilename string, out []Output) []Output {
	for i := range out {
		if out[i].Template != nil {
			fixRelative(basefilename, &out[i].Template.File)
		}
	}
	return out
}

func preprocessYAML(data []byte) []byte {
	// If input is a map, remove root-level keys starting with ".".
	// Other types such as slices, etc. and invalid YAML can be ignored
	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return data
	}

	maps.DeleteFunc(doc, func(k string, _ any) bool {
		return strings.HasPrefix(k, ".")
	})

	output, err := yaml.Marshal(doc)
	if err != nil {
		return data
	}
	return output
}
