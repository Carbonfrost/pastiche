// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dashboardapp

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Carbonfrost/pastiche/pkg/model"
)

type contextKey string

type modelMiddleware struct {
	next  http.Handler
	model *model.Model
}

func FromContext[T any](ctx context.Context) T {
	var t T
	return ctx.Value(contextKeyFor(t)).(T)
}

func ContextWithValue(ctx context.Context, v any) context.Context {
	return context.WithValue(ctx, contextKeyFor(v), v)
}

func contextKeyFor(v any) contextKey {
	switch v.(type) {
	case *model.Model:
		return "model"
	}
	panic(fmt.Errorf("not supported context value %T", v))
}

func newModelMiddleware(mo *model.Model, next http.Handler) http.Handler {
	return &modelMiddleware{model: mo, next: next}
}

func (m *modelMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	r = r.WithContext(ContextWithValue(ctx, m.model))
	m.next.ServeHTTP(w, r)
}

var _ http.Handler = (*modelMiddleware)(nil)
