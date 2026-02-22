// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"encoding/json"
	"log"
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
			httpserver.WithReadyFunc(httpserver.ReportListening),
		),
		httpserver.Handle("GET /api/v0/model", handleGetModel()),
		httpserver.Handle("/", handleDashboard()),
		httpserver.RunServer(),
	)
}

func handleGetModel() http.HandlerFunc {
	// Load configuration, converted to model but canonicalized back into model
	cfg, _ := config.Load()
	mo := model.ToConfig(model.New(cfg))

	return func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(mo)
	}
}

func handleDashboard() http.Handler {
	cfg, _ := config.Load()
	mo := model.New(cfg)
	handler, err := dashboardapp.New(mo)
	if err != nil {
		log.Println(err)
		return http.NotFoundHandler()
	}
	return handler
}
