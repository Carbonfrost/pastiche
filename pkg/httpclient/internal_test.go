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
