// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package workspace represents the workspace for Pastiche
package workspace

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"iter"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	cli "github.com/Carbonfrost/joe-cli"
	joeconfig "github.com/Carbonfrost/joe-cli/extensions/config"
	"github.com/Carbonfrost/pastiche/pkg/config"
	"github.com/Carbonfrost/pastiche/pkg/internal/contextkey"
	"github.com/Carbonfrost/pastiche/pkg/internal/log"
	"github.com/Carbonfrost/pastiche/pkg/model"
	"sigs.k8s.io/yaml"
)

// Workspace represents the information about the Pastiche
// workspace
type Workspace struct {
	// Action provides the action to provide when the workspace is
	// added to a pipeline.
	cli.Action

	ws *joeconfig.Workspace

	files []*config.File
	model *model.Model

	disableValidation bool
}

// Option sets up an option to configure the workspace
type Option interface {
	cli.Action
	apply(*Workspace)
}

type optionFunc func(*Workspace)

func (f optionFunc) Execute(ctx context.Context) error {
	f.apply(FromContext(ctx))
	return nil
}

func (f optionFunc) apply(ws *Workspace) {
	f(ws)
}

var (
	defaultOptions = []Option{
		withDefaultAction(),
	}
)

// New creates a new workspace
func New(opts ...Option) *Workspace {
	ws := &Workspace{}
	ws.ws = joeconfig.NewWorkspace(
		joeconfig.WithEnvProvider(&envProvider{ws: ws}),
	)

	for _, o := range append(defaultOptions, opts...) {
		o.apply(ws)
	}
	return ws
}

func withDefaultAction() optionFunc {
	return func(w *Workspace) {
		w.Action = cli.Pipeline(
			w.ws.Action,
			FlagsAndArgs(),
			ContextValue(w),
		)
	}
}

// FlagsAndArgs adds flags supporting the workspace. Despite its name, which is conventional,
// this action adds to args.
func FlagsAndArgs() cli.Action {
	return cli.AddFlags([]*cli.Flag{
		{Uses: SetDisableValidation()},
	}...)
}

// FromContext gets the Workspace from the context otherwise panics
func FromContext(ctx context.Context) *Workspace {
	return contextkey.Resolve(ctx, contextkey.Workspace).(*Workspace)
}

// ContextValue provides an action that sets the given value into the context.
// The only supported type is *Workspace.
func ContextValue(v *Workspace) cli.Action {
	return cli.WithContextValue(contextkey.Workspace, v)
}

func (w *Workspace) Pipeline() cli.Action {
	return w.Action
}

// Load will load the workspace, returning the error that occurred
// on load.
func (w *Workspace) Load() (*model.Model, error) {
	if err := w.loadExamples(); err != nil {
		return nil, err
	}

	if err := w.loadFromUser(); err != nil {
		return nil, err
	}

	if err := w.loadFromWorkspace(); err != nil {
		return nil, err
	}

	result := model.New(w.files...)
	if !w.disableValidation {
		return result, model.Validate(result)
	}

	return result, nil
}

// Model obtains the model for the workspace. This method implicitly
// loads the workspace and panics when the workspace cannot be loaded.
// Investigate [Workspace.Load] to load while handling load errors
func (w *Workspace) Model() *model.Model {
	if w.model == nil {
		var err error
		w.model, err = w.Load()
		if err != nil {
			panic(err)
		}
	}
	return w.model
}

func (w *Workspace) loadExamples() error {
	w.files = append(w.files, config.BuiltinFiles()...)
	return nil
}

func (w *Workspace) loadFromUser() error {
	root, err := filepath.Abs(os.ExpandEnv("$HOME/.pastiche"))
	if err != nil {
		return err
	}
	return w.loadFiles(root)
}

func (w *Workspace) loadFromWorkspace() error {
	root, err := filepath.Abs(".pastiche")
	if err != nil {
		return err
	}
	return w.loadFiles(root)
}

func (w *Workspace) loadFiles(root string) error {
	rootFS := os.DirFS(root)
	return fs.WalkDir(rootFS, ".", func(name string, d fs.DirEntry, err error) error {
		if d == nil {
			return nil
		}

		// TODO This should follow rules specified in .ignore files instead
		if d.IsDir() && d.Name() == "logs" {
			return fs.SkipDir
		}
		if strings.HasPrefix(d.Name(), "_") {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		if d.IsDir() {
			return nil
		}

		if err != nil {
			return err
		}

		file, err := config.LoadFile(rootFS, name)
		if err != nil {
			if errors.Is(err, config.ErrUnsupportedFileFormat) {
				return nil
			}

			log.Warnf("%s: %v", filepath.Join(root, name), err)
			return nil
		}

		w.files = append(w.files, file)
		return nil
	})
}

func (w *Workspace) Dir() string {
	return w.ws.Dir()
}

func (w *Workspace) ConfigDir() string {
	return w.ws.ConfigDir()
}

func (w *Workspace) environ() map[string]string {
	return map[string]string{
		"PASTICHE_DIR":        w.Dir(),
		"PASTICHE_LOG_DIR":    w.LogDir(),
		"PASTICHE_CONFIG_DIR": w.ConfigDir(),
	}
}

type envProvider struct {
	ws *Workspace
}

func (e *envProvider) Environ() iter.Seq2[string, string] {
	m := e.ws.environ()
	return func(yield func(string, string) bool) {
		for _, key := range slices.Sorted(maps.Keys(m)) {
			if !yield(key, m[key]) {
				return
			}
		}
	}
}

func (w *Workspace) LogDir() string {
	logDir := filepath.Join(w.Dir(), ".pastiche", "logs")
	os.MkdirAll(logDir, 0755)

	return logDir
}

func (w *Workspace) ClearLogDir() error {
	err := os.RemoveAll(w.LogDir())
	if err != nil {
		return err
	}

	_ = w.LogDir() // Recreate the directory
	return nil
}

func (w *Workspace) Describe(c *model.SearchCriteria) error {
	mo, err := w.Load()
	if err != nil {
		return err
	}

	results, err := mo.Search(c).Results()
	if err != nil {
		return err
	}

	var items describeResults
	for item := range results {
		if err := items.add(item); err != nil {
			return err
		}
	}

	// A spec names the items which are expected to exist, so when the other
	// criteria filter them all out, this is an error rather than empty output
	if items.empty() && c.Spec != nil && len(*c.Spec) > 0 {
		if len(c.IncludeTags) > 0 {
			return fmt.Errorf("not found with tags %v: %q", c.IncludeTags, c.Spec.Path())
		}
		return fmt.Errorf("not found: %q", c.Spec.Path())
	}

	return displayItems(&items)
}

type describeResults struct {
	Schema    string             `json:"$schema,omitempty"`
	Services  []config.Service   `json:"services,omitempty"`
	VarSets   []config.VarSet    `json:"varSets,omitempty"`
	Flows     []config.Flow      `json:"flows,omitempty"`
	Resources []config.Resource  `json:"resources,omitempty"`
	Endpoints []describeEndpoint `json:"endpoints,omitempty"`
}

type describeEndpoint struct {
	Method string `json:"method,omitempty"`
	config.Endpoint
}

func (d *describeResults) add(item model.Item) error {
	switch it := item.(type) {
	case *model.Service:
		d.Services = append(d.Services, toConfig[config.Service](it))
	case *model.VarSet:
		d.VarSets = append(d.VarSets, toConfig[config.VarSet](it))
	case *model.Flow:
		d.Flows = append(d.Flows, toConfig[config.Flow](it))
	case *model.Resource:
		d.Resources = append(d.Resources, *toConfig[*config.Resource](it))
	case *model.Endpoint:
		d.Endpoints = append(d.Endpoints, describeEndpoint{
			Method:   it.Method,
			Endpoint: *toConfig[*config.Endpoint](it),
		})
	default:
		return fmt.Errorf("cannot describe %T", item)
	}
	return nil
}

func (d *describeResults) empty() bool {
	return len(d.Services)+len(d.VarSets)+len(d.Flows)+len(d.Resources)+len(d.Endpoints) == 0
}

// fileSchema identifies the results as a configuration file, which is only
// accurate when every item within it can be written as one
func (d *describeResults) fileSchema() string {
	if len(d.Resources) > 0 || len(d.Endpoints) > 0 {
		return ""
	}
	return config.SchemaFile
}

func toConfig[V any](item model.Item) V {
	return model.ToConfig(item).(V)
}

func displayItems(d *describeResults) error {
	d.Schema = d.fileSchema()

	data, err := yaml.Marshal(d)
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func DisableValidation() Option {
	return optionFunc(func(w *Workspace) {
		w.disableValidation = true
	})
}

var _ joeconfig.EnvProvider = (*envProvider)(nil)
