// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package model

import (
	"errors"
	"fmt"
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
	return wrapError(
		"service "+s.Name,
		checkQName(s.Name),
		validate(s.Servers, validateServer),
		validateVars(s.Vars),
		validateResource(s.Resource),
	)
}

func wrapError(msg string, errs ...error) error {
	err := errors.Join(errs...)
	if err != nil {
		return fmt.Errorf("%s: %w", msg, err)
	}
	return nil
}

func validateServer(s *Server) error {
	if s == nil {
		return nil
	}
	if err := checkName(s.Name); err != nil {
		return err
	}
	if err := checkURLString(s.BaseURL); err != nil {
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
	if err := checkURLString(s.URITemplate); err != nil {
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

func checkURLString(v any) error {
	s := fmt.Sprint(v)
	if strings.Contains(s, "${") {
		return fmt.Errorf("URL cannot contain template expressions ${...}")
	}
	return nil
}

func checkQName(name string) error {
	qname, ok := strings.CutPrefix(name, "@")
	if ok {
		for _, name := range strings.SplitN(qname, "/", 2) {
			if err := checkName(name); err != nil {
				return fmt.Errorf("%q: %s", qname, badQIdentifierDetail)
			}
		}
		return nil
	}
	return checkName(name)
}

func checkName(name string) error {
	if len(name) == 0 || identifierPattern.MatchString(name) {
		return nil
	}

	return fmt.Errorf("%q: %s", name, badIdentifierDetail)
}
