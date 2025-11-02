// Copyright 2023 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package httpclient // intentional

import (
	"context"
	"net/url"

	"github.com/Carbonfrost/pastiche/pkg/model"
)

func NewContextWithLocation(spec model.ServiceSpec,
	resource *model.Resource,
	service *model.Service,
	server *model.Server,
	ep *model.Endpoint,
	u *url.URL) context.Context {

	location := &pasticheLocation{
		spec:     spec,
		resource: resource,
		service:  service,
		server:   server,
		ep:       ep,
		u:        u,
	}

	return context.WithValue(context.Background(), locationKey, location)
}
