// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package router

import (
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"path"
	"path/filepath"
	"strings"
)

// Router wraps an http.ServeMux.
type Router struct {
	mux          *http.ServeMux
	urlProvider  URLProvider
	dataProvider DataProvider
}

type Option interface {
	apply(*Router)
}

type optionFunc func(*Router)

func (f optionFunc) apply(r *Router) {
	f(r)
}

type Response interface {
	Render(w http.ResponseWriter, req *http.Request) error
}

type TemplateRenderContext struct {
	Request    *http.Request
	SourcePath string
	FS         fs.FS
}

// NewFSRouter constructs a Router from an fs.FS.
// It walks the filesystem and registers handlers
// according to NextJS-style routing conventions.
func NewFSRouter(fsys fs.FS, opts ...Option) (*Router, error) {
	router := &Router{}
	for _, o := range opts {
		o.apply(router)
	}

	var err error
	router.mux, err = buildMux2(fsys, &templateFSHandler{
		DataProvider: router.dataProvider,
		URLProvider:  router.urlProvider,
		FS:           fsys,
	})
	return router, err
}

func buildMux2(fsys fs.FS, global *templateFSHandler) (*http.ServeMux, error) {
	mux := http.NewServeMux()
	global.paths = map[string]string{}
	paths := global.paths

	global.Template = template.New("<site>").Funcs(map[string]any{
		"URL":     global.URLProvider,
		"Include": global.funcInclude(fsys),
	})

	tpls := global.Template
	err := fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		isSkipped := strings.HasPrefix(d.Name(), "_")
		if d.IsDir() {
			if isSkipped {
				return fs.SkipDir
			}
			return nil
		}

		if isSkipped {
			return nil
		}

		base := path.Base(p)

		if base == "index.html" {
			pattern := filePathToPattern(p)
			paths[pattern] = p
			return nil
		}

		// All other files → static file handler
		pattern := filePathToPattern(p)
		mux.Handle(pattern, &staticHandler{
			FS:   fsys,
			Path: p,
		})
		log.Printf("<static> %s -> %s\n", pattern, p)

		return nil
	})

	// Find all HTML files
	_ = fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		ext := path.Ext(p)
		if ext == ".html" {
			tmpl := tpls.New(p)

			// TODO Better error handling of template load and parse errors
			fileContents, _ := fs.ReadFile(fsys, p)
			_, err = tmpl.Parse(string(fileContents))

			if err != nil {
				fmt.Printf("error parsing the template: %v\n", err)
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	for pattern, path := range paths {
		mux.Handle(pattern, global)
		log.Printf("%s -> %s\n", pattern, path)
	}

	return mux, err
}

// ServeHTTP delegates to the internal mux.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

func WithURLProvider(fn URLProvider) Option {
	return optionFunc(func(r *Router) {
		r.urlProvider = fn
	})
}

func WithDataProvider(fn DataProvider) Option {
	return optionFunc(func(r *Router) {
		r.dataProvider = fn
	})
}

func filePathToPattern(p string) string {
	p = filepath.ToSlash(p)
	p = strings.TrimPrefix(p, "./")

	dir, file := path.Split(p)
	isTemplate := file == "index.html"
	if isTemplate {
		return "/" + transformSegments(strings.TrimSuffix(dir, "/"), true)
	}

	return "/" + transformSegments(p, false)
}

// transformSegments converts NextJS dynamic segments
// into Go 1.22 ServeMux pattern segments.
func transformSegments(p string, isTemplate bool) string {
	segments := strings.Split(p, "/")
	for i, s := range segments {
		switch {
		case strings.HasPrefix(s, "[...") && strings.HasSuffix(s, "]"):
			// [...param]
			name := strings.TrimSuffix(strings.TrimPrefix(s, "[..."), "]")
			segments[i] = "{" + name + "...}"

		case strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]"):
			// [param]
			name := strings.TrimSuffix(strings.TrimPrefix(s, "["), "]")
			segments[i] = "{" + name + "}"

		case strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")"):
			// Route groups → remove from URL
			segments[i] = ""
		}
	}
	if isTemplate && !strings.HasSuffix(segments[len(segments)-1], "...}") {
		segments = append(segments, "{$}")
	}

	return path.Join(segments...)
}
