// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package router

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"path/filepath"
	"strings"
)

type DataProvider func(*TemplateRenderContext) (any, error)
type URLProvider func(v any) string

type templateFSHandler struct {
	DataProvider DataProvider
	URLProvider  URLProvider
	FS           fs.FS
	Template     *template.Template
	paths        map[string]string
}

func (h *templateFSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	sourcePath := h.paths[r.Pattern]
	rsp, err := h.newTemplate(&TemplateRenderContext{
		Request:    r,
		SourcePath: sourcePath,
		FS:         h.FS,
	})
	if err != nil || rsp == nil {
		h.internalServerError(sourcePath, w, err)
		return
	}

	err = rsp.Render(w, r)

	if err != nil {
		h.internalServerError(sourcePath, w, err)
		return
	}
}

func (h *templateFSHandler) internalServerError(sourcePath string, w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	fmt.Fprint(w, "500 Internal Server Error")
	log.Printf("Error handling page %q: %v\n", sourcePath, err)
}

func (h *templateFSHandler) newTemplate(r *TemplateRenderContext) (Response, error) {
	data, err := h.DataProvider(r)
	if err != nil {
		return nil, err
	}

	tpls, err := h.newHTMLTemplate(r.FS, r.SourcePath)
	if err != nil {
		return nil, err
	}

	return &htmlTemplateResponse{data: data, template: tpls, sourcePath: r.SourcePath}, nil
}

func (h *templateFSHandler) newHTMLTemplate(fsys fs.FS, path string) (*template.Template, error) {
	tpls, err := h.Template.Clone()

	// TODO Better error handling of template load and parse errors
	fileContents, _ := fs.ReadFile(fsys, path)
	_, err = tpls.Parse(string(fileContents))

	if err != nil {
		fmt.Printf("error parsing the template: %v\n", err)
	}
	if tpls == nil {
		return template.New(path).Parse("missing template: " + path)
	}
	return tpls, nil
}

func (h *templateFSHandler) funcInclude(fsys fs.FS) func(name string, v ...any) (any, error) {
	return func(name string, v ...any) (any, error) {
		var data any
		if len(v) == 1 {
			data = v[0]
		} else {
			var err error
			data, err = mapOf(v...)
			if err != nil {
				return "", err
			}
		}

		buf := bytes.NewBuffer(nil)
		tpls, err := h.newHTMLTemplate(fsys, name)
		if err != nil {
			return "", err
		}

		err = tpls.Execute(buf, data)
		if err != nil {
			return nil, err
		}
		return template.HTML(buf.String()), nil
	}
}

func mapOf(kv ...any) (map[string]any, error) {
	if len(kv)%2 != 0 {
		return nil, fmt.Errorf("requires even number of arguments, got %d", len(kv))
	}

	m := make(map[string]any, len(kv)/2)

	for i := 0; i < len(kv); i += 2 {
		key := fmt.Sprint(kv[i])
		m[key] = kv[i+1]
	}

	return m, nil
}

type htmlTemplateResponse struct {
	data       any
	sourcePath string
	template   *template.Template
}

func (h *htmlTemplateResponse) Render(w http.ResponseWriter, req *http.Request) error {
	return h.template.Execute(w, h.data)
}

type staticHandler struct {
	FS   fs.FS
	Path string
}

func (h *staticHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f, err := h.FS.Open(h.Path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", contentTypeFromExt(h.Path))
	_, _ = io.Copy(w, f)
}

func contentTypeFromExt(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".html":
		return "text/html; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".js":
		return "application/javascript"
	case ".json":
		return "application/json"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".svg":
		return "image/svg+xml"
	case ".txt":
		return "text/plain; charset=utf-8"
	default:
		return "application/octet-stream"
	}
}
