// Copyright 2025 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"text/template"

	joehttpclient "github.com/Carbonfrost/joe-cli-http/httpclient"
	"github.com/Carbonfrost/joe-cli-http/httpclient/expr"
	"github.com/Carbonfrost/pastiche/pkg/internal/log"
)

type templateContent struct {
	tpl  *template.Template
	form map[string]any
	err  error
}

var _ joehttpclient.Content = (*templateContent)(nil)

type Expander = expr.Expander

func resolveVar(exp Expander) func(...string) (any, error) {
	return func(vars ...string) (any, error) {
		if len(vars) == 0 {
			return "", fmt.Errorf("var/n requires at least one var name")
		}
		for i, v := range vars {
			result := exp(v)
			if result != nil {
				return result, nil
			}
			if i > 0 && i == len(vars)-1 {
				// Last value is a literal
				return v, nil
			}
		}
		return "", fmt.Errorf("var not found: %q", vars[0])
	}
}

func newTemplateContent(body any, vars map[string]any) joehttpclient.Content {
	form := map[string]any{}
	t, err := template.New("<body content>").
		Funcs(template.FuncMap{
			"env": os.Getenv,
			"var": resolveVar(expr.ComposeExpanders(
				expr.ExpandMap(form),
				expr.ExpandMap(vars),
			)),
		}).
		Parse(string(bodyToBytes(body)))

	return &templateContent{
		tpl:  t,
		form: form,
		err:  err,
	}
}

func bodyToBytes(data any) []byte {
	if s, ok := data.(string); ok {
		return []byte(s)
	}
	result, _ := json.Marshal(data)
	return result
}

func (t *templateContent) Read() io.Reader {
	if t.err != nil {
		return firstReadError{t.err}
	}

	var result bytes.Buffer
	err := t.tpl.Execute(&result, nil)
	if err != nil {
		log.Warn("error executing template", err)
		return firstReadError{err}
	}

	return bytes.NewReader(result.Bytes())
}

func (t *templateContent) Query() (url.Values, error) {
	panic("not impl")
}

func (t *templateContent) ContentType() string {
	return ""
}

func (t *templateContent) Set(name, value string) error {
	t.form[name] = value
	return nil
}

func (t *templateContent) SetFile(name, file io.Reader) error {
	panic("not impl")
}

type firstReadError struct {
	err error
}

func (f firstReadError) Read([]byte) (int, error) {
	return 0, f.err
}
