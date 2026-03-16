// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dashboardapp

import (
	"cmp"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"strings"

	"github.com/Carbonfrost/pastiche/pkg/model"
	"github.com/Carbonfrost/pastiche/pkg/swvlexp/router"
)

//go:embed web/**
var dashboardWeb embed.FS

func New(mo *model.Model) (http.Handler, error) {
	site, err := fs.Sub(dashboardWeb, "web")
	if err != nil {
		panic(err)
	}

	router, err := router.NewFSRouter(
		site,
		router.WithURLProvider(URL),
		router.WithDataProvider(loadData),
	)
	if err != nil {
		return nil, err
	}

	return newModelMiddleware(mo, router), nil
}

func loadData(r *router.TemplateRenderContext) (any, error) {
	req := r.Request

	switch r.SourcePath {
	case "[...spec]/index.html":
		mo := FromContext[*model.Model](req.Context())
		spec := req.PathValue("spec")
		svc, ok := mo.Service(spec)
		if !ok {
			return nil, fmt.Errorf("service not found %s", spec)
		}
		return svc, nil
	case "index.html":
		mo := FromContext[*model.Model](req.Context())
		return newPasticheIndexView(mo), nil
	}
	log.Printf("no template metadata %s\n", r.SourcePath)
	return nil, nil
}

func URL(v any) string {
	switch val := v.(type) {
	case *model.Service:
		return fmt.Sprintf("/%s", val.Name)
	default:
		return fmt.Sprintf("/?%T", val)
	}
}

type ServiceNode struct {
	Service  *model.Service
	Parent   *ServiceNode
	Children []*ServiceNode
	Title    string
	URL      string
}

func (n *ServiceNode) appendChild(c *ServiceNode) {
	n.Children = append(n.Children, c)
	c.Parent = n
}

func serviceNodes(svcs []*model.Service) []*ServiceNode {
	nodes := make([]*ServiceNode, 0, len(svcs))
	var current *ServiceNode
	for i, s := range svcs {
		ad := &ServiceNode{
			Service: svcs[i],
			Title:   cmp.Or(svcs[i].Title, svcs[i].Name),
			URL:     URL(s),
		}

		if group, _, ok := strings.Cut(s.Name, "/"); ok {
			if current != nil && current.Title == group {

			} else {
				current = &ServiceNode{
					Service: nil,
					Title:   group,
					URL:     fmt.Sprintf("/%s", group),
				}
				nodes = append(nodes, current)
			}

			current.appendChild(ad)
		} else {
			nodes = append(nodes, ad)
		}
	}
	return nodes
}

type PasticheIndexView struct {
	Services     []*model.Service
	ServiceNodes []*ServiceNode
}

func newPasticheIndexView(mo *model.Model) PasticheIndexView {
	return PasticheIndexView{
		Services:     mo.Services,
		ServiceNodes: serviceNodes(mo.Services),
	}
}
