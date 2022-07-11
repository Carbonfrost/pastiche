package config

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/Carbonfrost/pastiche/pkg/model"
)

type ServiceConfig struct {
	Services map[string]*model.Service
}

func newServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		Services: map[string]*model.Service{},
	}
}

func Load() (sc *ServiceConfig, err error) {
	sc = newServiceConfig()
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

func (c *ServiceConfig) loadExamples() error {
	c.Services["httpbin"] = ExampleHTTPBinorg
	return nil
}

func (c *ServiceConfig) loadFromUser() error {
	root, err := filepath.Abs(os.ExpandEnv("$HOME/.pastiche"))
	if err != nil {
		return err
	}
	return c.loadFiles(root)
}

func (c *ServiceConfig) loadFromWorkspace() error {
	root, err := filepath.Abs(".pastiche")
	if err != nil {
		return err
	}
	return c.loadFiles(root)
}

func (c *ServiceConfig) loadFiles(root string) error {
	return fs.WalkDir(os.DirFS(root), ".", func(name string, d fs.DirEntry, err error) error {
		if d == nil || d.IsDir() {
			return nil
		}

		file := filepath.Join(root, name)
		if err != nil {
			return err
		}

		if filepath.Ext(file) == ".json" {
			data, err := os.ReadFile(file)
			if err != nil {
				logWarning(err)
				return nil
			}

			service := new(model.Service)
			if err := json.Unmarshal(data, service); err != nil {
				logWarning(err)
				return nil
			}

			if service.Name != "" {
				c.Services[service.Name] = service
			}
		}

		return nil
	})
}

func (c *ServiceConfig) Service(name string) (*model.Service, bool) {
	svc, ok := c.Services[name]
	return svc, ok
}

func (c *ServiceConfig) Resolve(s model.ServiceSpec) (*model.Service, *model.Resource, error) {
	if len(s) == 0 {
		return nil, nil, fmt.Errorf("no service specified")
	}
	svc, ok := c.Service(s[0])
	if !ok {
		return nil, nil, fmt.Errorf("service not found: %q", s[0])
	}

	current := svc.Resource
	for i, p := range s[1:] {
		current, ok = current.Resource(p)
		if !ok {
			path := model.ServiceSpec(s[0 : i+2]).Path()
			return svc, current, fmt.Errorf("resource not found: %q", path)
		}
	}
	return svc, current, nil
}

func logWarning(v interface{}) {
	fmt.Fprint(os.Stderr, v)
}
