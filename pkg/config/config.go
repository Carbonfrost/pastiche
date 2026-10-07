// Copyright 2025, 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package config provides the configuration for Pastiche service
// definitions.
package config

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"path"
	"path/filepath"
	"strings"

	"sigs.k8s.io/yaml"
)

// MaxIncludeDepth is the maximum number of cascaded inclusions via the
// source attribute, which also stops cyclic inclusions.
const MaxIncludeDepth = 16

var (
	ErrUnsupportedFileFormat = errors.New("unsupported file format")
	ErrIncludeDepthExceeded  = errors.New("maximum depth of cascaded inclusions exceeded")
)

type fileFormat struct {
	toJSON  func([]byte) ([]byte, error)
	strict  bool
	varSets bool
}

var fileFormats = map[string]fileFormat{
	".json":     {toJSON: jsonToJSON},
	".jsonvars": {toJSON: jsonToJSON, varSets: true},
	".yaml":     {toJSON: yamlToJSON, strict: true},
	".yamlvars": {toJSON: yamlToJSON, strict: true, varSets: true},
	".yml":      {toJSON: yamlToJSON, strict: true},
	".ymlvars":  {toJSON: yamlToJSON, strict: true, varSets: true},
}

// sourced is implemented by types which support the source attribute.
type sourced interface {
	source() *string

	// plain converts to a type that doesn't implement sourced, so that it
	// unmarshals using the default rules
	plain() any
}

type (
	plainService        Service
	plainServer         Server
	plainResource       Resource
	plainEndpoint       Endpoint
	plainFlow           Flow
	plainMixin          Mixin
	plainTemplateOutput TemplateOutput
	plainGRPCClient     GRPCClient
)

// sourcer unmarshals file, resolving relative paths and inclusions against it
type sourcer struct {
	f     fs.FS
	file  string
	depth int
}

// LoadFile loads the given file from the file system and name
func LoadFile(f fs.FS, filename string) (*File, error) {
	format, ok := fileFormats[filepath.Ext(filename)]
	if !ok {
		return nil, fmt.Errorf("load file %s: %w", filename, ErrUnsupportedFileFormat)
	}

	result := new(File)
	result.SetName(filename)

	var target any = result
	if format.varSets {
		target = &result.VarSets
	}

	s := sourcer{f: f, file: filename}
	if err := s.unmarshal(format, target); err != nil {
		return nil, err
	}

	if len(result.Services) > 0 && result.Service != nil {
		return nil, fmt.Errorf("must contain either service definition or services list, but not both")
	}

	// The embedded service is inlined into the file, so it isn't visited by
	// the unmarshaler for sourced types
	if result.Service != nil {
		if err := s.include(result.Service); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (s sourcer) unmarshal(format fileFormat, v any) error {
	data, err := fs.ReadFile(s.f, s.file)
	if err != nil {
		return err
	}
	data, err = format.toJSON(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v,
		json.RejectUnknownMembers(format.strict),
		json.WithUnmarshalers(s.unmarshalers()),
	)
}

func (s sourcer) unmarshalers() *json.Unmarshalers {
	return json.JoinUnmarshalers(
		json.UnmarshalFromFunc(func(dec *jsontext.Decoder, v sourced) error {
			// File only implements sourced by promotion from its embedded service
			if _, ok := v.(*File); ok {
				return errors.ErrUnsupported
			}
			if err := json.UnmarshalDecode(dec, v.plain()); err != nil {
				return err
			}
			return s.include(v)
		}),
		json.UnmarshalFromFunc(func(dec *jsontext.Decoder, v *TemplateOutput) error {
			if err := json.UnmarshalDecode(dec, (*plainTemplateOutput)(v)); err != nil {
				return err
			}
			fixRelative(s.file, &v.File)
			return nil
		}),
		json.UnmarshalFromFunc(func(dec *jsontext.Decoder, v *GRPCClient) error {
			if err := json.UnmarshalDecode(dec, (*plainGRPCClient)(v)); err != nil {
				return err
			}
			fixRelative(s.file, &v.ProtoSet)
			return nil
		}),
	)
}

// include unmarshals the file named by the source attribute into v, which
// cascades when the included file has its own source attribute
func (s sourcer) include(v sourced) error {
	src := v.source()
	file := *src
	if file == "" {
		return nil
	}
	if s.depth >= MaxIncludeDepth {
		return fmt.Errorf("%s: %w", file, ErrIncludeDepthExceeded)
	}

	format, ok := fileFormats[filepath.Ext(file)]
	if !ok || format.varSets {
		return fmt.Errorf("%s: %w", file, ErrUnsupportedFileFormat)
	}

	included := sourcer{
		f:     s.f,
		file:  path.Join(path.Dir(s.file), file),
		depth: s.depth + 1,
	}

	// Cleared so that only a source attribute in the included file cascades
	*src = ""
	if err := included.unmarshal(format, v); err != nil {
		return fmt.Errorf("%s: %w", file, err)
	}
	*src = file
	return nil
}

func (s *Service) source() *string  { return &s.Source }
func (s *Server) source() *string   { return &s.Source }
func (s *Resource) source() *string { return &s.Source }
func (s *Endpoint) source() *string { return &s.Source }
func (s *Flow) source() *string     { return &s.Source }
func (s *Mixin) source() *string    { return &s.Source }

func (s *Service) plain() any  { return (*plainService)(s) }
func (s *Server) plain() any   { return (*plainServer)(s) }
func (s *Resource) plain() any { return (*plainResource)(s) }
func (s *Endpoint) plain() any { return (*plainEndpoint)(s) }
func (s *Flow) plain() any     { return (*plainFlow)(s) }
func (s *Mixin) plain() any    { return (*plainMixin)(s) }

func jsonToJSON(data []byte) ([]byte, error) {
	return data, nil
}

func yamlToJSON(data []byte) ([]byte, error) {
	return yaml.YAMLToJSON(preprocessYAML(data))
}

func fixRelative(basefilename string, pathStr *string) {
	if pathStr == nil || *pathStr == "" || strings.HasPrefix(*pathStr, "/") {
		return
	}
	resolvedFile := path.Join(path.Dir(basefilename), *pathStr)
	*pathStr = resolvedFile
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
