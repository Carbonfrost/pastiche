// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package router

import (
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
	rendererFunc TemplateRendererFunc
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

type TemplateRendererFunc func(*TemplateRenderContext) (Response, error)

// NewFSRouter constructs a Router from an fs.FS.
// It walks the filesystem and registers handlers
// according to NextJS-style routing conventions.
func NewFSRouter(fsys fs.FS, opts ...Option) (*Router, error) {
	router := &Router{}
	for _, o := range opts {
		o.apply(router)
	}

	var err error
	router.mux, err = buildMux(fsys, router.rendererFunc)
	return router, err
}

func buildMux(fsys fs.FS, fn TemplateRendererFunc) (*http.ServeMux, error) {
	mux := http.NewServeMux()

	err := fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		base := path.Base(p)

		if base == "index.html" {
			pattern := filePathToPattern(p)
			mux.Handle(pattern, &pageHandler{
				SourcePath:           p,
				TemplateRendererFunc: fn,
				FS:                   fsys,
			})
			log.Printf("%s -> %s\n", pattern, p)
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

	return mux, err
}

// ServeHTTP delegates to the internal mux.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

func WithTemplateRenderer(fn TemplateRendererFunc) Option {
	return optionFunc(func(r *Router) {
		r.rendererFunc = fn
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
