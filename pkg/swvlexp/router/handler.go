// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package router

import (
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"path/filepath"
	"strings"
)

type pageHandler struct {
	SourcePath           string
	TemplateRendererFunc TemplateRendererFunc
	FS                   fs.FS
}

func (h *pageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	rsp, err := h.TemplateRendererFunc(&TemplateRenderContext{
		Request:    r,
		SourcePath: h.SourcePath,
		FS:         h.FS,
	})
	if err != nil || rsp == nil {
		h.internalServerError(w, err)
		return
	}

	err = rsp.Render(w, r)

	if err != nil {
		h.internalServerError(w, err)
		return
	}
}

func (h *pageHandler) internalServerError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	fmt.Fprint(w, "500 Internal Server Error")
	log.Printf("Error handling page %q: %v\n", h.SourcePath, err)
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
