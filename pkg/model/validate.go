// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package model

import (
	"errors"
	"regexp"
	"strings"
)

const (
	badIdentifierDetail  = "A name must start with a letter or underscore and may contain only letters, digits, underscores, and dashes."
	badQIdentifierDetail = "A package-scoped name must have two parts that each are valid identifiers"
)

var (
	identifierPattern = regexp.MustCompile(`^(?i)[_a-z][a-z0-9_-]*$`)
)

func Validate(m *Model) error {
	if m == nil {
		return nil
	}
	for _, s := range m.Services {
		err := validateService(s)
		if err != nil {
			return err
		}
	}
	return nil
}

func validateService(s *Service) error {
	if s == nil {
		return nil
	}
	if err := checkQName(s.Name); err != nil {
		return err
	}
	if err := validate(s.Servers, validateServer); err != nil {
		return err
	}
	if err := validateVars(s.Vars); err != nil {
		return err
	}
	return validateResource(s.Resource)
}

func validateServer(s *Server) error {
	if s == nil {
		return nil
	}
	if err := checkName(s.Name); err != nil {
		return err
	}
	return validateVars(s.Vars)
}

func validateResource(s *Resource) error {
	if s == nil {
		return nil
	}
	if err := checkName(s.Name); err != nil {
		return err
	}
	if err := validateVars(s.Vars); err != nil {
		return err
	}
	if err := validate(s.Endpoints, validateEndpoint); err != nil {
		return err
	}
	return validate(s.Resources, validateResource)
}

func validateEndpoint(s *Endpoint) error {
	if s == nil {
		return nil
	}
	err := checkName(s.Name)
	if err != nil {
		return err
	}
	return validateVars(s.Vars)
}

func validateVars(vars map[string]any) error {
	for k := range vars {
		if err := checkName(k); err != nil {
			return err
		}
	}
	return nil
}

func validate[V any](values []V, fn func(V) error) error {
	for _, v := range values {
		err := fn(v)
		if err != nil {
			return err
		}
	}
	return nil
}

func checkQName(name string) error {
	qname, ok := strings.CutPrefix(name, "@")
	if ok {
		for _, name := range strings.SplitN(qname, "/", 2) {
			if err := checkName(name); err != nil {
				return errors.New(badQIdentifierDetail)
			}
		}
	}
	return checkName(name)
}

func checkName(name string) error {
	if len(name) == 0 || identifierPattern.MatchString(name) {
		return nil
	}

	return errors.New(badIdentifierDetail)
}
