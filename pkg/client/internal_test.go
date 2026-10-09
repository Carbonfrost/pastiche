// Copyright 2023, 2025 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package client // intentional

import (
	"net/url"

	"github.com/Carbonfrost/joe-cli-http/uritemplates"
	"github.com/Carbonfrost/pastiche/pkg/model"
)

func NewLocation(
	resource *model.Resource,
	service *model.Service,
	server *model.Server,
	ep *model.Endpoint,
	u *url.URL) *pasticheLocation {

	loc, _ := newLocation(&model.ResolvedResource{
		Resource: resource,
		Service:  service,
		Server:   server,
		Endpoint: ep,
	})
	return loc
}

func NewLocationVars(vars uritemplates.Vars, r *model.ResolvedResource) *pasticheLocation {
	loc, _ := newLocation(r, model.WithVars(vars))
	return loc
}

// CurrentFilter exposes the filter set on the client.
func (c *Client) CurrentFilter() Filter {
	return c.filter
}

// IsNamedOutputFilter determines whether the filter resolves a named output,
// and if so, which one.
func IsNamedOutputFilter(f Filter) (string, bool) {
	n, ok := f.(namedOutputFilter)
	return n.name, ok
}
