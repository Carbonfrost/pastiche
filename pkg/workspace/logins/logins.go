// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package logins provides the JSON representation of logins and stores them
// as files within the logins directory of the workspace.
package logins

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/Carbonfrost/pastiche/pkg/internal/log"
	"github.com/Carbonfrost/pastiche/pkg/model"
)

// Entry provides the JSON representation of a login.
type Entry struct {
	Name    string     `json:"name,omitempty"`
	Server  string     `json:"server,omitempty"`
	Token   string     `json:"token"`
	Expires *time.Time `json:"expires,omitempty"`
}

// Logins stores each login as a JSON file within a directory.  The file is
// named by the service spec formed from the name and server, with slashes
// normalized to hyphens so that the directory is a flat list of files.  For
// example, the login named @example/customers for the server production is
// stored in @example-customers.production.json
type Logins struct {
	dir string
	now func() time.Time
}

const fileExt = ".json"

// New creates a login provider which reads logins from JSON files within the
// given directory
func New(dir string) *Logins {
	return &Logins{
		dir: dir,
		now: time.Now,
	}
}

// NewEntry creates the JSON representation of the given login.
func NewEntry(l *model.Login) *Entry {
	return &Entry{
		Name:    l.Name,
		Server:  l.Server,
		Token:   l.Token,
		Expires: l.Expires,
	}
}

func (e Entry) toLogin() *model.Login {
	return &model.Login{
		Name:    e.Name,
		Server:  e.Server,
		Token:   e.Token,
		Expires: e.Expires,
	}
}

// FileName obtains the name of the file which contains the login with the
// given name for the given server.
func (l *Logins) FileName(name, server string) (string, error) {
	spec := model.ServiceSpec{name}
	if server != "" {
		spec = append(spec, server)
	}
	if !validSpec(spec) {
		return "", fmt.Errorf("invalid login name %q and server %q", name, server)
	}

	base := strings.ReplaceAll(spec.Path(), "/", "-")
	return filepath.Join(l.dir, base+fileExt), nil
}

func (l *Logins) GetLogin(name, server string) (*model.Login, error) {
	path, err := l.FileName(name, server)
	if err != nil {
		return nil, err
	}

	e, err := readEntry(path)
	if err != nil {
		return nil, err
	}

	// Because slashes and hyphens share a file name, a name recorded in the
	// file which differs means that the file belongs to another login
	if (e.Name != "" && e.Name != name) || (e.Server != "" && e.Server != server) {
		return nil, &fs.PathError{Op: "open", Path: path, Err: fs.ErrNotExist}
	}

	login := e.toLogin()
	login.Name = name
	login.Server = server
	if login.Expired(l.now()) {
		return login, model.ErrLoginExpired
	}
	return login, nil
}

// SaveLogin writes the login to its file.  Because the file contains a
// token, it is readable only by the user.
func (l *Logins) SaveLogin(login *model.Login) error {
	path, err := l.FileName(login.Name, login.Server)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(NewEntry(login), "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(l.dir, 0o700); err != nil {
		return err
	}

	// Write to a temporary file first so that a partial write never
	// replaces an existing login
	f, err := os.CreateTemp(l.dir, ".login-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())

	if _, err := f.Write(append(data, '\n')); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}

func (l *Logins) ClearLogin(name, server string) error {
	path, err := l.FileName(name, server)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

func (l *Logins) ClearAllLogins() error {
	return os.RemoveAll(l.dir)
}

func (l *Logins) ListLogins() ([]*model.Login, error) {
	var result []*model.Login
	for _, f := range l.files() {
		result = append(result, f.login)
	}

	slices.SortFunc(result, func(x, y *model.Login) int {
		return cmp.Or(
			cmp.Compare(x.Name, y.Name),
			cmp.Compare(x.Server, y.Server),
		)
	})
	return result, nil
}

func (l *Logins) GC() error {
	now := l.now()
	for _, f := range l.files() {
		if f.login.Expired(now) {
			if err := os.Remove(f.path); err != nil && !errors.Is(err, fs.ErrNotExist) {
				return err
			}
		}
	}
	return nil
}

type loginFile struct {
	path  string
	login *model.Login
}

// files obtains the logins stored in the directory, skipping files which are
// not logins
func (l *Logins) files() []loginFile {
	// The pattern is valid, so Glob never returns an error
	matches, _ := filepath.Glob(filepath.Join(l.dir, "*"+fileExt))

	var result []loginFile
	for _, path := range matches {
		e, err := readEntry(path)
		if err != nil {
			// Not a login, so leave it alone
			log.Warnf("warning: %v", err)
			continue
		}

		login := e.toLogin()
		if login.Name == "" {
			login.Name, login.Server = nameFromFile(path)
		}
		result = append(result, loginFile{path, login})
	}
	return result
}

func nameFromFile(path string) (name, server string) {
	base := strings.TrimSuffix(filepath.Base(path), fileExt)
	name, server, _ = strings.Cut(base, ".")
	return name, server
}

// validSpec ensures that the name and server form a valid service spec, which
// also ensures that the file is contained in the directory.
func validSpec(spec model.ServiceSpec) bool {
	for _, s := range spec {
		if s == "" || strings.HasSuffix(s, "/") || strings.Contains(s, ".") {
			return false
		}
	}
	if len(spec) > 1 && strings.Contains(spec[1], "/") {
		return false
	}
	parsed, err := model.ParseServiceSpec(spec.Path())
	return err == nil && len(parsed) == len(spec)
}

func readEntry(path string) (*Entry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var e Entry
	if err := json.Unmarshal(data, &e); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &e, nil
}

var _ model.LoginProvider = (*Logins)(nil)
