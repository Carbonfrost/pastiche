// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli-http/httpserver"
	"github.com/Carbonfrost/pastiche/pkg/config"
	"github.com/Carbonfrost/pastiche/pkg/model"
	"github.com/Carbonfrost/pastiche/pkg/server/dashboardapp"
)

// Serve provides the action to run the server
func Serve() cli.Action {
	return cli.Pipeline(
		cli.Prototype{
			Name:     "serve",
			Aliases:  []string{"server"},
			HelpText: "Run an API or website server for the workspace",
		},
		httpserver.New(
			httpserver.WithPort(9161),
		),
		cli.Before(cli.Pipeline(
			httpserver.Handle("GET /api/v0/model", httpserver.NewReloadableHandler(handleGetModel)),
			httpserver.Handle("/", httpserver.NewReloadableHandler(handleDashboard)),
		)),
		httpserver.RunServer(),
	)
}

func handleGetModel(_ context.Context) (http.Handler, error) {
	// Load configuration, converted to model but canonicalized back into model
	cfg, _ := config.Load()
	mo := model.ToConfig(model.New(cfg))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(mo)
	}), nil
}

func handleDashboard(_ context.Context) (http.Handler, error) {
	cfg, _ := config.Load()
	mo := model.New(cfg)
	return dashboardapp.New(mo)
}
