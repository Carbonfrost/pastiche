package config

var (
	ExampleHTTPBinorg = &Service{
		Name:        "httpbin",
		Title:       "httpbin.org",
		Description: "A simple HTTP Request & Response Service.",
		Servers: []*Server{
			{
				Name:    "production",
				BaseURL: "https://httpbin.org/",
			},
		},
		Resource: &Resource{
			Name: "/",
			Resources: []*Resource{
				{
					Name:   "delete",
					URI:    "/delete",
					Delete: &Endpoint{},
				},
				{
					Name: "get",
					URI:  "/get",
					Get:  &Endpoint{},
				},
				{
					Name:  "patch",
					URI:   "/patch",
					Patch: &Endpoint{},
				},
				{
					Name: "post",
					URI:  "/post",
					Post: &Endpoint{},
				},
				{
					Name: "put",
					URI:  "/put",
					Put:  &Endpoint{},
				},
				{
					Name: "status",
					Resources: []*Resource{
						{
							Name: "codes",
							URI:  "/status/{codes}",
							Get:  &Endpoint{},
						},
					},
				},
			},
		},
	}
)
