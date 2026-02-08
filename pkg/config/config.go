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
	"os"
	"path/filepath"
	"strings"

	"github.com/Carbonfrost/pastiche/pkg/internal/log"
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
	".json": json.Unmarshal,
	".yaml": unmarshalYaml,
	".yml":  unmarshalYaml,
}

var ErrUnsupportedFileFormat = errors.New("unsupported file format")

func Load() (sc *Config, err error) {
	sc = &Config{}
	if err = sc.loadExamples(); err != nil {
		return
	}
	if err = sc.loadFromUser(); err != nil {
		return
	}
	if err = sc.loadFromWorkspace(); err != nil {
		return
	}

	return
}

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

func (c *Config) appendServices(s ...Service) {
	c.Services = append(c.Services, s...)
}

func (c *Config) loadExamples() error {
	c.appendServices(Builtins()...)
	return nil
}

func (c *Config) loadFromUser() error {
	root, err := filepath.Abs(os.ExpandEnv("$HOME/.pastiche"))
	if err != nil {
		return err
	}
	return c.loadFiles(root)
}

func (c *Config) loadFromWorkspace() error {
	root, err := filepath.Abs(".pastiche")
	if err != nil {
		return err
	}
	return c.loadFiles(root)
}

func (c *Config) loadFiles(root string) error {
	rootFS := os.DirFS(root)
	return fs.WalkDir(rootFS, ".", func(name string, d fs.DirEntry, err error) error {
		if d == nil {
			return nil
		}

		// TODO This should follow rules specified in .ignore files instead
		if d.IsDir() && d.Name() == "logs" {
			return fs.SkipDir
		}
		if d.IsDir() {
			return nil
		}

		if strings.HasPrefix(name, "_") {
			return nil
		}

		if err != nil {
			return err
		}

		file, err := LoadFile(rootFS, name)
		if err != nil {
			if errors.Is(err, ErrUnsupportedFileFormat) {
				return nil
			}

			log.Warnf("%s: %v", filepath.Join(root, name), err)
			return nil
		}

		if file.Service != nil {
			c.appendServices(*file.Service)
		}
		c.appendServices(file.Services...)

		return nil
	})
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
		resolvedFile := filepath.Join(filepath.Dir(basefilename), file)
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
		return sources(s, file, a.Resources)

	case *Server:
		// Nothing to do for servers

	case *Resource:
		err := sources(s, file, a.Resources)
		if err != nil {
			return err
		}
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

func fixRelative(basefilename string, path *string) {
	if path == nil {
		return
	}
	resolvedFile := filepath.Join(filepath.Dir(basefilename), *path)
	*path = resolvedFile
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
