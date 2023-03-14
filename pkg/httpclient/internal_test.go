package httpclient // intentional

import (
	"context"
	"net/url"

	"github.com/Carbonfrost/pastiche/pkg/model"
)

func NewContextWithLocation(spec model.ServiceSpec,
	resource *model.Resource,
	service *model.Service,
	ep *model.Endpoint,
	u *url.URL) context.Context {

	location := &pasticheLocation{
		spec:     spec,
		resource: resource,
		service:  service,
		ep:       ep,
		u:        u,
	}

	return context.WithValue(context.Background(), locationKey, location)
}
