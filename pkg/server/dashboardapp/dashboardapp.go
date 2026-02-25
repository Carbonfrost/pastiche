// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dashboardapp

import (
	"embed"
	"fmt"
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
		router.WithURLProvider(URL),
		router.WithDataProvider(loadData),
	)
	if err != nil {
		return nil, err
	}

	return newModelMiddleware(mo, router), nil
}

func loadData(r *router.TemplateRenderContext) (any, error) {
	req := r.Request

	switch r.SourcePath {
	case "[...spec]/index.html":
		mo := FromContext[*model.Model](req.Context())
		spec := req.PathValue("spec")
		svc, ok := mo.Service(spec)
		if !ok {
			return nil, fmt.Errorf("service not found %s", spec)
		}
		return svc, nil
	case "index.html":
		mo := FromContext[*model.Model](req.Context())
		return mo, nil
	}
	log.Printf("no template metadata %s\n", r.SourcePath)
	return nil, nil
}

func URL(v any) string {
	switch val := v.(type) {
	case *model.Service:
		return fmt.Sprintf("/%s", val.Name)
	default:
		return fmt.Sprintf("/?%T", val)
	}
}
