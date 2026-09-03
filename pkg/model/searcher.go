// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package model

import (
	"fmt"
	"iter"
	"slices"
	"strings"
)

// ItemKind enumerates the kinds of items within a model which can be searched.
type ItemKind int

// The kinds of items we can search for. Resources and endpoints have can qualified
// names, but other types must have either simple name or only package qualified names
const (
	ItemKindService ItemKind = iota
	ItemKindVarSet
	ItemKindMixin
	ItemKindFlow
	ItemKindResource
	ItemKindEndpoint
	ItemKindAll
)

// Searcher enumerates the items within a model which match a search criteria.
type Searcher interface {
	Results() (iter.Seq[Item], error)
}

// SearchCriteria is the union of all of the criteria which are relevant to
// each kind of searcher.
type SearchCriteria struct {
	// Spec names the items to search for.  For var sets, services, and flows,
	// the spec must be just a name because these don't nest, but for resources
	// or endpoints, this could be a qualifying name.  When the spec is empty,
	// all items of the given kind are considered.
	Spec *ServiceSpec

	// Kind identifies which kind of item is searched for.
	Kind ItemKind

	// IncludeTags filters items to those which have all of these tags.  When
	// empty, no filtering on tags occurs.
	IncludeTags []string

	// Method filters endpoints to those which use this request method.  It is
	// only relevant to endpoints.
	Method string
}

type searchSupport struct {
	model    *Model
	criteria *SearchCriteria
}

type (
	serviceSearcher  struct{ searchSupport }
	varSetSearcher   struct{ searchSupport }
	mixinSearcher    struct{ searchSupport }
	flowSearcher     struct{ searchSupport }
	resourceSearcher struct{ searchSupport }
	endpointSearcher struct{ searchSupport }

	// allSearcher composes the other searchers, tolerating searchers which
	// can't be satisfied by the criteria (such as a var set being named by a
	// qualified name)
	allSearcher struct{ searchers []Searcher }
)

// Search provides a searcher over the items in the model which match the
// criteria.
func (m *Model) Search(criteria *SearchCriteria) Searcher {
	if criteria == nil {
		criteria = &SearchCriteria{}
	}
	support := searchSupport{model: m, criteria: criteria}

	switch criteria.Kind {
	case ItemKindService:
		return &serviceSearcher{support}
	case ItemKindVarSet:
		return &varSetSearcher{support}
	case ItemKindMixin:
		return &mixinSearcher{support}
	case ItemKindFlow:
		return &flowSearcher{support}
	case ItemKindResource:
		return &resourceSearcher{support}
	case ItemKindEndpoint:
		return &endpointSearcher{support}
	case ItemKindAll:
		return &allSearcher{
			searchers: []Searcher{
				&serviceSearcher{support},
				&varSetSearcher{support},
				&mixinSearcher{support},
				&flowSearcher{support},
				&resourceSearcher{support},
				&endpointSearcher{support},
			},
		}
	}
	return errSearcher{fmt.Errorf("unknown item kind: %v", int(criteria.Kind))}
}

func (s *serviceSearcher) Results() (iter.Seq[Item], error) {
	name, err := s.simpleName("service")
	if err != nil {
		return nil, err
	}

	items := s.model.Services
	if name != "" {
		svc, ok := s.model.Service(name)
		if !ok {
			return nil, fmt.Errorf("service not found: %q", name)
		}
		items = []*Service{svc}
	}

	return func(yield func(Item) bool) {
		for _, svc := range items {
			if !s.matchesTags(svc.Tags) {
				continue
			}
			if !yield(svc) {
				return
			}
		}
	}, nil
}

func (s *varSetSearcher) Results() (iter.Seq[Item], error) {
	name, err := s.simpleName("var set")
	if err != nil {
		return nil, err
	}

	items := s.model.VarSets
	if name != "" {
		vs, ok := s.model.VarSet(name)
		if !ok {
			return nil, fmt.Errorf("var set not found: %q", name)
		}
		items = []*VarSet{vs}
	}

	return func(yield func(Item) bool) {
		for _, vs := range items {
			if !s.matchesTags(vs.Tags) {
				continue
			}
			if !yield(vs) {
				return
			}
		}
	}, nil
}

func (s *mixinSearcher) Results() (iter.Seq[Item], error) {
	name, err := s.simpleName("mixin")
	if err != nil {
		return nil, err
	}

	items := s.model.Mixins
	if name != "" {
		mx, ok := s.model.Mixin(name)
		if !ok {
			return nil, fmt.Errorf("mixin not found: %q", name)
		}
		items = []*Mixin{mx}
	}

	return func(yield func(Item) bool) {
		for _, mx := range items {
			if !s.matchesTags(mx.Tags) {
				continue
			}
			if !yield(mx) {
				return
			}
		}
	}, nil
}

func (s *flowSearcher) Results() (iter.Seq[Item], error) {
	name, err := s.simpleName("flow")
	if err != nil {
		return nil, err
	}

	items := s.model.Flows
	if name != "" {
		f, ok := s.model.Flow(name)
		if !ok {
			return nil, fmt.Errorf("flow not found: %q", name)
		}
		items = []*Flow{f}
	}

	return func(yield func(Item) bool) {
		for _, f := range items {
			if !s.matchesTags(f.Tags) {
				continue
			}
			if !yield(f) {
				return
			}
		}
	}, nil
}

func (s *resourceSearcher) Results() (iter.Seq[Item], error) {
	resources, err := s.resources(false)
	if err != nil {
		return nil, err
	}

	return func(yield func(Item) bool) {
		for res := range resources {
			if !s.matchesTags(res.Tags) {
				continue
			}
			if !yield(res) {
				return
			}
		}
	}, nil
}

func (s *endpointSearcher) Results() (iter.Seq[Item], error) {
	// The scope itself is included because the unnamed root resource of a
	// service can also define endpoints
	resources, err := s.resources(true)
	if err != nil {
		return nil, err
	}

	return func(yield func(Item) bool) {
		for res := range resources {
			for _, ep := range res.Endpoints {
				if !s.matchesMethod(ep.Method) || !s.matchesTags(ep.Tags) {
					continue
				}
				if !yield(ep) {
					return
				}
			}
		}
	}, nil
}

func (a *allSearcher) Results() (iter.Seq[Item], error) {
	var (
		seqs     []iter.Seq[Item]
		firstErr error
	)

	for _, s := range a.searchers {
		seq, err := s.Results()
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		seqs = append(seqs, seq)
	}

	// Only when no searcher at all could be satisfied is the criteria
	// considered erroneous
	if len(seqs) == 0 && firstErr != nil {
		return nil, firstErr
	}

	return func(yield func(Item) bool) {
		for _, seq := range seqs {
			for item := range seq {
				if !yield(item) {
					return
				}
			}
		}
	}, nil
}

type errSearcher struct {
	err error
}

func (e errSearcher) Results() (iter.Seq[Item], error) {
	return nil, e.err
}

func (s searchSupport) simpleName(kind string) (string, error) {
	spec := s.criteria.Spec
	if spec == nil || len(*spec) == 0 {
		return "", nil
	}
	if len(*spec) > 1 {
		return "", fmt.Errorf("%s cannot be named by a qualified name: %q", kind, spec.Path())
	}
	return (*spec)[0], nil
}

func (s searchSupport) resources(includeScope bool) (iter.Seq[*Resource], error) {
	scope, exact, err := s.scope()
	if err != nil {
		return nil, err
	}

	return func(yield func(*Resource) bool) {
		for _, r := range scope {
			items := descendants(r)
			switch {
			case exact:
				items = slices.Values([]*Resource{r})
			case includeScope:
				items = selfAndDescendants(r)
			}

			for res := range items {
				if !yield(res) {
					return
				}
			}
		}
	}, nil
}

func (s searchSupport) scope() (res []*Resource, exact bool, err error) {
	spec := s.criteria.Spec
	if spec == nil || len(*spec) == 0 {
		for _, svc := range s.model.Services {
			if svc.Resource != nil {
				res = append(res, svc.Resource)
			}
		}
		return res, false, nil
	}

	svc, ok := s.model.Service((*spec)[0])
	if !ok {
		return nil, false, fmt.Errorf("service not found: %q", (*spec)[0])
	}

	current := svc.Resource
	for i, name := range (*spec)[1:] {
		found := false
		if current != nil {
			current, found = current.Resource(name)
		}
		if !found {
			return nil, false, fmt.Errorf("resource not found: %q", ServiceSpec((*spec)[0:i+2]).Path())
		}
	}
	if current == nil {
		return nil, false, nil
	}
	return []*Resource{current}, len(*spec) > 1, nil
}

func (s searchSupport) matchesTags(tags []string) bool {
	for _, required := range s.criteria.IncludeTags {
		if !slices.Contains(tags, required) {
			return false
		}
	}
	return true
}

func (s searchSupport) matchesMethod(method string) bool {
	if s.criteria.Method == "" {
		return true
	}
	return strings.EqualFold(method, s.criteria.Method)
}

func descendants(r *Resource) iter.Seq[*Resource] {
	return func(yield func(*Resource) bool) {
		for _, c := range r.Resources {
			if !yield(c) {
				return
			}
			for d := range descendants(c) {
				if !yield(d) {
					return
				}
			}
		}
	}
}

func selfAndDescendants(r *Resource) iter.Seq[*Resource] {
	return func(yield func(*Resource) bool) {
		if !yield(r) {
			return
		}
		for d := range descendants(r) {
			if !yield(d) {
				return
			}
		}
	}
}
