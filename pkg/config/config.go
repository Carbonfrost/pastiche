// Copyright 2025 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package config

import (
	"encoding/json"
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

func (c *Config) appendService(s Service) {
	c.Services = append(c.Services, s)
}

func (c *Config) loadExamples() error {
	c.appendService(ExampleHTTPBinorg)
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
	return fs.WalkDir(os.DirFS(root), ".", func(name string, d fs.DirEntry, err error) error {
		if d == nil || d.IsDir() {
			return nil
		}

		file := filepath.Join(root, name)
		if err != nil {
			return err
		}

		if unmarshal, ok := unmarshalers[filepath.Ext(file)]; ok {
			data, err := os.ReadFile(file)
			if err != nil {
				log.Warn(err)
				return nil
			}

			service := new(Service)
			if err := unmarshal(data, service); err != nil {
				log.Warn(err)
				return nil
			}

			c.appendService(*service)
		}

		return nil
	})
}

func unmarshalYaml(data []byte, v any) error {
	return yaml.UnmarshalStrict(data, v)
}
