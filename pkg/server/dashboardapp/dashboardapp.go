// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dashboardapp

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"

	"github.com/Carbonfrost/pastiche/pkg/model"
	"github.com/Carbonfrost/pastiche/pkg/swvlexp/router"
)

//go:embed web/**
var dashboardWeb embed.FS

func New(mo *model.Model) (http.Handler, error) {
	site, err := fs.Sub(dashboardWeb, "web")
	if err != nil {
		panic(err)
	}

	router, err := router.NewFSRouter(
		site,
		router.WithTemplateRenderer(newTemplate),
	)
	if err != nil {
		return nil, err
	}

	return newModelMiddleware(mo, router), nil
}

func newTemplate(r *router.TemplateRenderContext) (router.Response, error) {
	data, err := loadData(r)
	if err != nil {
		return nil, err
	}

	tpls, err := newHTMLTemplate(r.FS, r.SourcePath)
	if err != nil {
		return nil, err
	}

	return &htmlTemplateResponse{data: data, template: tpls}, nil
}

func loadData(r *router.TemplateRenderContext) (any, error) {
	req := r.Request

	switch r.SourcePath {
	case "[...spec]/index.html":
		mo := FromContext[*model.Model](req.Context())
		svc, ok := mo.Service(req.PathValue("spec"))
		if !ok {
			return nil, fmt.Errorf("not found")
		}
		return svc, nil
	case "index.html":
		mo := FromContext[*model.Model](req.Context())
		return mo, nil
	}
	log.Printf("no template metadata %s\n", r.SourcePath)
	return nil, nil
}

func newHTMLTemplate(fsys fs.FS, path string) (*template.Template, error) {
	content := func() string {
		result, _ := fs.ReadFile(fsys, path)
		return string(result)
	}()
	tpls, err := template.New("<site>").Funcs(map[string]any{
		"URL":     URL,
		"Include": funcInclude(fsys),
	}).Parse(content)
	return tpls, err
}

func funcInclude(fsys fs.FS) func(name string, v ...any) (any, error) {
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
		tpls, err := newHTMLTemplate(fsys, name)
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

// TODO Push up into exp
type htmlTemplateResponse struct {
	data     any
	template *template.Template
}

func (h *htmlTemplateResponse) Render(w http.ResponseWriter, req *http.Request) error {
	return h.template.Execute(w, h.data)
}

func URL(v any) string {
	switch val := v.(type) {
	case *model.Service:
		return fmt.Sprintf("/%s", val.Name)
	default:
		return fmt.Sprintf("/?%T", val)
	}
}
