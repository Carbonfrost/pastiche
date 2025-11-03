// Copyright 2023 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package httpclient // intentional

import (
	"net/url"

	"github.com/Carbonfrost/joe-cli-http/uritemplates"
	"github.com/Carbonfrost/pastiche/pkg/model"
	"github.com/Carbonfrost/pastiche/pkg/model/modelfakes"
)

func NewLocation(
	resource *model.Resource,
	service *model.Service,
	server *model.Server,
	ep *model.Endpoint,
	u *url.URL) *pasticheLocation {

	loc, _ := newLocation(nil, nil, &modelfakes.FakeResolvedResource{
		ResourceStub: func() *model.Resource {
			return resource
		},
		ServiceStub: func() *model.Service {
			return service
		},
		ServerStub: func() *model.Server {
			return server
		},
		EndpointStub: func() *model.Endpoint {
			return ep
		},
		URLStub: func(*url.URL, uritemplates.Vars) (*url.URL, error) {
			return u, nil
		},
	})
	return loc
}
