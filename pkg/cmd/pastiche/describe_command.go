// Copyright 2023 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package pastiche

import (
	"fmt"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/pastiche/pkg/config"
	"github.com/Carbonfrost/pastiche/pkg/model"
	"sigs.k8s.io/yaml"
)

func DescribeServiceCommand() cli.Action {
	return cli.Setup{
		Uses: cli.Pipeline(
			cli.AddArgs([]*cli.Arg{
				{
					Name:  "name",
					Value: cli.List(),
				},
			}...)),
		Action: func(c *cli.Context) error {
			name := c.List("name")
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			mo := model.New(cfg)
			if len(name) == 0 {
				for _, s := range mo.Services {
					displayService(s)
				}
				return nil
			}

			for _, nom := range name {
				s, ok := mo.Service(nom)
				if !ok {
					return fmt.Errorf("service not found: %q", nom)
				}
				displayService(s)
			}
			return nil
		},
	}
}

func displayService(s *model.Service) {
	m := &model.Model{
		Services: []*model.Service{
			s,
		},
	}

	data, _ := yaml.Marshal(model.ToConfig(m))
	fmt.Println(string(data))
}
