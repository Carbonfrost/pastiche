// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package model

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"unicode"

	"github.com/Carbonfrost/joe-cli/extensions/expr/expander"
	"github.com/Carbonfrost/pastiche/pkg/internal/log"
)

func newSecretExpander(secrets []*Secret) Expander {
	byName := make(map[string]*Secret, len(secrets))
	for _, s := range secrets {
		if s.Name != "" {
			byName[s.Name] = s
		}
	}
	return expander.Func(func(name string) any {
		s, ok := byName[name]
		if !ok {
			return nil
		}
		value, err := loadSecret(s)
		if err != nil {
			log.Warnf("warning: unable to load secret %q: %v", name, err)
			return fmt.Sprintf("<missing secret %s>", name)
		}
		return value
	})
}

func loadSecret(s *Secret) (string, error) {
	switch src := s.Provider.(type) {
	case *ExecSecret:
		out, err := exec.Command("sh", "-c", src.Command).Output()
		if err != nil {
			return "", err
		}
		return string(out), nil
	case *FileSecret:
		data, err := os.ReadFile(src.Path)
		if err != nil {
			return "", err
		}
		content := string(data)
		if !src.PreserveTrailingWhitespace {
			content = strings.TrimRightFunc(content, unicode.IsSpace)
		}
		return content, nil
	default:
		return "", fmt.Errorf("secret %q has no provider", s.Name)
	}
}
