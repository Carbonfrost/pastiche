package config

import (
	"github.com/Carbonfrost/joe-cli-http/uritemplates"
	"github.com/Carbonfrost/pastiche/pkg/model"
)

var (
	ExampleHTTPBinorg = &model.Service{
		Name:        "httpbin",
		Title:       "httpbin.org",
		Description: "A simple HTTP Request & Response Service.",
		Servers: []*model.Server{
			{
				Name:    "production",
				BaseURL: "https://httpbin.org/",
			},
		},
		Resource: &model.Resource{
			Name: "/",
			Resources: []*model.Resource{
				{
					Name:        "delete",
					URITemplate: t("/delete"),
					Method:      "DELETE",
				},
				{
					Name:        "get",
					URITemplate: t("/get"),
					Method:      "GET",
				},
				{
					Name:        "patch",
					URITemplate: t("/patch"),
					Method:      "PATCH",
				},
				{
					Name:        "post",
					URITemplate: t("/post"),
					Method:      "POST",
				},
				{
					Name:        "put",
					URITemplate: t("/put"),
					Method:      "PUT",
				},
				{
					Name: "status",
					Resources: []*model.Resource{
						{
							Name:        "codes",
							URITemplate: t("/status/{codes}"),
							Method:      "GET",
						},
					},
				},
			},
		},
	}
)

func t(s string) *uritemplates.URITemplate {
	res, _ := uritemplates.Parse(s)
	return res
}
