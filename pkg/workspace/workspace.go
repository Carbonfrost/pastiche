// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package workspace represents the workspace for Pastiche
package workspace

import (
	"context"
	"errors"
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
)

// Workspace represents the information about the Pastiche
// workspace
type Workspace struct {
	// Action provides the action to provide when the workspace is
	// added to a pipeline.
	cli.Action

	*joeconfig.Workspace

	files []*config.File
	model *model.Model
}

// New creates a new workspace
func New() *Workspace {
	return withDefaultAction(&Workspace{
		Workspace: joeconfig.NewWorkspace(),
	})
}

func withDefaultAction(w *Workspace) *Workspace {
	w.Action = cli.Pipeline(
		w.Workspace.Action,
		ContextValue(w),
	)
	return w
}

// FromContext gets the Workspace from the context otherwise panics
func FromContext(ctx context.Context) *Workspace {
	return contextkey.Resolve(ctx, contextkey.Workspace).(*Workspace)
}

// ContextValue provides an action that sets the given value into the context.
// The only supported type is *Workspace.
func ContextValue(v *Workspace) cli.Action {
	return cli.ContextValue(contextkey.Workspace, v)
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

	return model.New(w.files...), nil
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
	w.LoadFiles(".pastiche")
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
		if d.IsDir() {
			return nil
		}

		if strings.HasPrefix(name, "_") {
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

func (w *Workspace) Env() iter.Seq2[string, string] {
	m := map[string]string{
		"PASTICHE_DIR":        w.Dir(),
		"PASTICHE_LOG_DIR":    w.LogDir(),
		"PASTICHE_CONFIG_DIR": w.ConfigDir(),
	}

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
