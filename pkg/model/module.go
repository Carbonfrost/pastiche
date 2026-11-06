// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package model

import (
	"cmp"
	"slices"
)

type Module struct {
	services []*Service
	varSets  []*VarSet
	flows    []*Flow
	mixins   []*Mixin
}

func (m *Module) Services() []*Service {
	return m.services
}

func (m *Module) VarSets() []*VarSet {
	return m.varSets
}

func (m *Module) Flows() []*Flow {
	return m.flows
}

func (m *Module) Mixins() []*Mixin {
	return m.mixins
}

func (m *Module) Service(name string) (*Service, bool) {
	return m.searchModule(name, m.services, func(s *Service) string {
		return s.Name
	})
}

func (m *Module) Server(spec ServiceSpec) (*Server, bool) {
	svc, ok := m.Service(spec.ServiceName())
	if !ok {
		return nil, false
	}
	if len(spec) < 2 {
		if len(svc.Servers) == 0 {
			return nil, false
		}
		return svc.Servers[0], true
	}
	return svc.Server(spec[1])
}

func (m *Module) VarSet(name string) (*VarSet, bool) {
	return m.searchModule(name, m.varSets, func(s *VarSet) string {
		return s.Name
	})
}

func (m *Module) Mixin(name string) (*Mixin, bool) {
	return m.searchModule(name, m.mixins, func(s *Mixin) string {
		return s.Name
	})
}

func (m *Module) Flow(name string) (*Flow, bool) {
	return m.searchModule(name, m.flows, func(s *Flow) string {
		return s.Name
	})
}

func (m *Module) searchModule[T any](name string, items []T, fn func(T) string) (result T, ok bool) {
	idx, ok := slices.BinarySearchFunc(items, name, func(x T, y string) int {
		return cmp.Compare(fn(x), y)
	})
	return items[idx], ok
}

func (m *Module) copyToModel(dst *Model) {
	dst.Services = append(dst.Services, m.Services()...)
	dst.VarSets = append(dst.VarSets, m.VarSets()...)
	dst.Flows = append(dst.Flows, m.Flows()...)
	dst.Mixins = append(dst.Mixins, m.Mixins()...)
}
