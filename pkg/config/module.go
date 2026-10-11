// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

// Module represents a collection of files
type Module struct {
	Files []*File

	name string
}

type FileOrModule interface {
	Name() string
	fileOrModuleSigil()
}

func (m *Module) Name() string {
	return m.name
}

func (m *Module) SetName(name string) {
	m.name = name
}

func (*Module) fileOrModuleSigil() {}

var _ FileOrModule = (*Module)(nil)
