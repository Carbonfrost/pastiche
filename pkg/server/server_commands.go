// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"context"
	"net/http"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli-http/httpserver"
	"github.com/Carbonfrost/pastiche/pkg/contextual"
	"github.com/Carbonfrost/pastiche/pkg/server/dashboardapp"
	"github.com/Carbonfrost/pastiche/pkg/server/metaapi"
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
			contextual.Middleware(),
			httpserver.Handle("GET /api/v0/model", httpserver.NewReloadableHandler(handleGetModel)),
			httpserver.Handle("/", httpserver.NewReloadableHandler(handleDashboard)),
		)),
		httpserver.RunServer(),
	)
}

func handleGetModel(c context.Context) (http.Handler, error) {
	mo := contextual.Workspace(c).Model()
	return metaapi.New(mo)
}

func handleDashboard(c context.Context) (http.Handler, error) {
	mo := contextual.Workspace(c).Model()
	return dashboardapp.New(mo)
}
