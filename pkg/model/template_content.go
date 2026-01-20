// Copyright 2025 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"text/template"

	joehttpclient "github.com/Carbonfrost/joe-cli-http/httpclient"
	"github.com/Carbonfrost/joe-cli-http/httpclient/expr"
	"github.com/Carbonfrost/pastiche/pkg/internal/log"
	"github.com/Carbonfrost/pastiche/pkg/template/funcs"
)

type templateContent struct {
	*contentSupport
	tpl string
}

type formContent struct {
	*contentSupport
}

type Expander = expr.Expander

type objectContent struct {
	*contentSupport
	value any
}

type contentSupport struct {
	form url.Values
	vars map[string]any
}

func newContentSupport(vars map[string]any) *contentSupport {
	return &contentSupport{
		form: url.Values{},
		vars: vars,
	}
}

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

func newFormContent(form map[string][]string, vars map[string]any) joehttpclient.Content {
	return &formContent{
		contentSupport: &contentSupport{
			form: form,
			vars: vars,
		},
	}
}

func newTemplateContent(body any, vars map[string]any) joehttpclient.Content {
	if str, ok := body.(string); ok {
		return &templateContent{
			contentSupport: newContentSupport(vars),
			tpl:            str,
		}
	}
	return &objectContent{
		contentSupport: newContentSupport(vars),
		value:          body,
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
	tpl, err := template.New("<body content>").
		Funcs(template.FuncMap{
			"env": os.Getenv,
			"var": resolveVar(t.Expander()),
		}).
		Parse(t.tpl)

	if err != nil {
		log.Warn("error parsing template", err)
		return firstReadError{err}
	}

	data := map[string]any{}
	funcs.AddTo(data)

	var result bytes.Buffer
	err = tpl.Execute(&result, data)
	if err != nil {
		log.Warn("error executing template", err)
		return firstReadError{err}
	}

	return bytes.NewReader(result.Bytes())
}

type firstReadError struct {
	err error
}

func (f firstReadError) Read([]byte) (int, error) {
	return 0, f.err
}

func (t *objectContent) Read() io.Reader {
	var result bytes.Buffer
	expander := t.Expander()
	err := json.NewEncoder(&result).Encode(
		expandObject(t.value, expander),
	)
	if err != nil {
		log.Warn("error encoding body", err)
		return firstReadError{err}
	}

	return bytes.NewReader(result.Bytes())
}

func (t *formContent) Read() io.Reader {
	expander := t.Expander()
	values := expandObject(t.form, expander).(url.Values)

	return strings.NewReader(values.Encode())
}

func (t *contentSupport) Expander() Expander {
	return expr.ComposeExpanders(
		expr.Prefix("var", expr.ExpandMap(t.vars)),
		expr.Prefix("form", expandURLValues(t.form)),
		expandURLValues(t.form),
		expr.ExpandMap(t.vars),
	)
}

func (t *contentSupport) Set(name, value string) error {
	t.form.Add(name, value)
	return nil
}

func (t *contentSupport) SetFile(name, file io.Reader) error {
	panic("not impl")
}

func (t *contentSupport) Query() (url.Values, error) {
	panic("not impl")
}

func (t *contentSupport) ContentType() string {
	return ""
}

func expandObject(v any, e Expander) any {
	switch value := v.(type) {
	case nil:
		return nil
	case string:
		return expandString(value, e)
	case map[string]any:
		newValues := map[string]any{}
		for k, v := range value {
			// TODO Support expanding keys too
			newValues[k] = expandObject(v, e)
		}
		return newValues
	case []any:
		newValues := make([]any, len(value))
		for i := range value {
			newValues[i] = expandObject(value[i], e)
		}
		return newValues
	case []string:
		newValues := make([]string, len(value))
		for i := range value {
			newValues[i] = expandString(value[i], e)
		}
		return newValues
	case http.Header:
		return http.Header(expandHeader(value, e))
	case url.Values:
		return url.Values(expandHeader(value, e))

	case map[string][]string:
		return expandHeader(value, e)
	default:
		return v
	}
}

func expandHeader(value map[string][]string, e Expander) map[string][]string {
	copy := map[string][]string{}
	for k, v := range value {
		result := make([]string, len(v))
		for i, str := range v {
			result[i] = expandString(str, e)
		}
		copy[k] = result
	}
	return copy
}

func expandString(s string, e Expander) string {
	return expr.SyntaxRecursive.CompilePattern(s, "${", "}").Expand(e)
}

func expandURLValues(u url.Values) expr.Expander {
	return func(s string) any {
		if u.Has(s) {
			return strings.Join(u[s], ",")
		}
		return nil
	}
}
