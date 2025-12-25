// Copyright 2025 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/Carbonfrost/pastiche/pkg/internal/log"
	"sigs.k8s.io/yaml"
)

type Config struct {
	Services []Service
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
		if err := unmarshal(data, result); err != nil {
			return nil, err
		}

		if len(result.Services) > 0 && result.Service != nil {
			return nil, fmt.Errorf("must contain either service definition or services list, but not both")
		}

		return result, nil
	}

	return nil, fmt.Errorf("load file %s: %w", filename, ErrUnsupportedFileFormat)
}

func (c *Config) appendServices(s ...Service) {
	c.Services = append(c.Services, s...)
}

func (c *Config) loadExamples() error {
	c.appendServices(ExampleHTTPBinorg)
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
		if d == nil || d.IsDir() {
			return nil
		}
		if err != nil {
			return err
		}

		file, err := LoadFile(rootFS, name)
		if err != nil && !errors.Is(err, ErrUnsupportedFileFormat) {
			log.Warn(err)
			return nil
		}

		if file.Service != nil {
			c.appendServices(*file.Service)
		}
		c.appendServices(file.Services...)

		return nil
	})
}

func unmarshalYaml(data []byte, v any) error {
	return yaml.UnmarshalStrict(data, v)
}
