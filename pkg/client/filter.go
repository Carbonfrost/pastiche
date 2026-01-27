// Copyright 2023, 2025, 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"text/template"

	cli "github.com/Carbonfrost/joe-cli"
	joehttpclient "github.com/Carbonfrost/joe-cli-http/httpclient"
	"github.com/Carbonfrost/joe-cli-http/httpclient/expr"
	"github.com/Carbonfrost/joe-cli/extensions/provider"
	"github.com/Carbonfrost/pastiche/pkg/template/funcs"
	"github.com/jmespath/go-jmespath"
)

// Filter applies a search to the response data.
type Filter interface {
	Search(data any) (any, error)
}

type templateFilter struct {
	loader func() (string, error)
}

type templateOpts struct {
	Text string `mapstructure:"text"`
	File string `mapstructure:"file"`
}

type digFilter string

type filterOpts struct {
	Query string `mapstructure:"query"`
}

type filteredDownload struct {
	Downloader joehttpclient.Downloader
	filter     Filter
}

type filteredWriter struct {
	*bytes.Buffer

	output io.Writer
	filter Filter
}

var (
	// FilterRegistry contains all filters available to the client.
	FilterRegistry = &provider.Registry{
		Name: "filter",
		Providers: provider.Details{
			"jmespath": {
				Factory: provider.Factory(newJMESPath),
				Defaults: map[string]string{
					"query": "@",
				},
				HelpText: "Use JMES Path to select matching JSON data",
			},
			"dig": {
				Factory: provider.Factory(newDig),
				Defaults: map[string]string{
					"query": "",
				},
				HelpText: "Use a simple expression to retrieve a value",
			},
			"gotpl": {
				Factory: provider.Factory(newTemplate),
				Defaults: map[string]string{
					"text": "Result: {{ .Result }}",
					"file": "",
				},
				HelpText: "Use Go template to manipulate matching data",
			},
		},
	}
)

// NewJMESPathFilter provides a filter which uses the given query to search
// a data structure with JSON semantics using JMESPath.
func NewJMESPathFilter(query string) (Filter, error) {
	return jmespath.Compile(query)
}

func newJMESPath(opts filterOpts) (Filter, error) {
	return NewJMESPathFilter(opts.Query)
}

// NewDigFilter creates a filter which resolves a qualified name in
// a response value.
func NewDigFilter(query string) (Filter, error) {
	return digFilter(query), nil
}

func newDig(opts filterOpts) (Filter, error) {
	return NewDigFilter(opts.Query)
}

// NewFilterDownloader applies the filter to an underlying downloader.
func NewFilterDownloader(f Filter, d joehttpclient.Downloader) joehttpclient.Downloader {
	return &filteredDownload{
		filter:     f,
		Downloader: d,
	}
}

func newFilteredWriter(output io.Writer, f Filter) *filteredWriter {
	return &filteredWriter{
		Buffer: new(bytes.Buffer),
		output: output,
		filter: f,
	}
}

func (f *filteredDownload) OpenDownload(ctx context.Context, r *joehttpclient.Response) (io.WriteCloser, error) {
	output, err := f.Downloader.OpenDownload(ctx, r)
	if err != nil {
		return nil, err
	}

	return newFilteredWriter(output, f.filter), nil
}

func (c *filteredWriter) Close() error {
	var data any

	err := json.Unmarshal(c.Buffer.Bytes(), &data)
	if err != nil {
		return err
	}

	res, err := c.filter.Search(data)
	if err != nil {
		return err
	}

	// Write directly if the filter is to []byte, else codec output
	if out, ok := res.([]byte); ok {
		c.output.Write(out)

	} else {
		e := json.NewEncoder(c.output)
		err = e.Encode(res)
		if err != nil {
			return err
		}
	}

	if closer, ok := c.output.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

func (d digFilter) Search(data any) (any, error) {
	var err error
	for name := range strings.SplitSeq(strings.TrimLeft(string(d), "."), ".") {
		data, err = dig(data, name)
		if err != nil {
			break
		}
	}

	return data, err
}

func dig(data any, name string) (any, error) {
	switch d := data.(type) {
	case string:
		return nil, fmt.Errorf("cannot index string with `%s'", name)

	case map[string]any:
		if result, ok := d[name]; ok {
			return result, nil
		}
		return nil, fmt.Errorf("key not found `%s'", name)

	case map[any]any:
		if result, ok := d[name]; ok {
			return result, nil
		}
		return nil, fmt.Errorf("key not found `%s'", name)

	case []any:
		return index(d, name)

	case []string:
		return index(d, name)

	default:
		// TODO Reflection via structs and slices
		return nil, fmt.Errorf("cannot index %T with `%s'", d, name)
	}
}

func index[T any](values []T, index string) (any, error) {
	in, err := strconv.Atoi(index)
	if err == nil && in >= 0 && in < len(values) {
		return values[in], nil
	}
	return nil, fmt.Errorf("cannot index array with `%s'", index)
}

// ListFilters provides an action which lists all filters available to the filter registry.
func ListFilters() cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "list-filters",
			HelpText: "List available output filters then exit",
		},
		provider.ListProviders("filter"),
	)
}

// SetFilter provides an action which sets the filter which will be used in the response.
// This also provides an accessory flag.
func SetFilter(f ...*provider.Value) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "filter",
			Aliases:  []string{"l"},
			HelpText: "Apply a filter query to the response data using a supported format",
		},
		withBinding((*Client).setFilterHelper, f),
		cli.Accessory("-", (*provider.Value).ArgumentFlag),
	)
}

func (j *filterOpts) UnmarshalText(data []byte) error {
	j.Query = string(data)
	return nil
}

// NewTemplateFilter creates a filter which resolves a qualified name in
// a response value.
func NewTemplateFilter(tpl string) (Filter, error) {
	return newTemplateFilterString(tpl), nil
}

func (t templateFilter) Search(data any) (any, error) {
	text, err := t.loader()
	if err != nil {
		return nil, err
	}

	// TODO This should be expander capable of vars, form, etc.
	expander := expr.ExpandGlobals

	var results bytes.Buffer

	funcMap := template.FuncMap{}
	funcs.AddToFuncs(funcMap)
	funcs.AddVarResolver(funcMap, expander)

	tpl, err := template.New("<filter>").Funcs(funcMap).Parse(text)

	templateData := map[string]any{
		"Result": data,
	}
	funcs.AddTo(templateData)

	err = tpl.Execute(&results, templateData)
	return results.Bytes(), err
}

func newTemplateFilterString(tpl string) templateFilter {
	return templateFilter{
		loader: func() (string, error) {
			return tpl, nil
		},
	}
}

func newTemplateFilterFile(filename string) templateFilter {
	return templateFilter{
		loader: func() (string, error) {
			// TODO This should use context fs.FS
			data, err := os.ReadFile(filename)
			return string(data), err
		},
	}
}

func newTemplate(opts templateOpts) (Filter, error) {
	if opts.File != "" {
		return newTemplateFilterFile(opts.File), nil
	}
	return newTemplateFilterString(opts.Text), nil
}
