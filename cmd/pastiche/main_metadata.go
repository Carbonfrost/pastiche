// Copyright 2025 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build json_marshal

package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/Carbonfrost/joe-cli/extensions/marshal"
	"github.com/Carbonfrost/pastiche/pkg/cmd/pastiche"
)

// This prints out marshal information about the app
// go run -tags json_marshal ./cmd/pastiche

func main() {
	app := pastiche.NewApp()
	_, err := app.Initialize(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	json.NewEncoder(os.Stdout).Encode(marshal.From(app))
}
