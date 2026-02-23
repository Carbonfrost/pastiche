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
	"github.com/antchfx/xmlquery"
	"github.com/antchfx/xpath"
	"github.com/jmespath/go-jmespath"
	"sigs.k8s.io/yaml"
)

// Filter applies a search to the response data and returns
// the response data.
type Filter interface {
	Search(response Response) ([]byte, error)
}

type defaultFilter int

type jsonFilter struct {
	e func(io.Writer) *json.Encoder
}

type jsonFilterOpts struct {
	Pretty bool `mapstructure:"pretty"`
}

type xmlFilter struct {
	options []xmlquery.OutputOption
}

type xmlFilterOpts struct {
	Pretty bool `mapstructure:"pretty"`
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

type yamlFilter struct{}

type jmesPathFilter struct {
	q *jmespath.JMESPath
}

type xpathFilter struct {
	q *xpath.Expr
}

type filteredDownload struct {
	Downloader joehttpclient.Downloader
	filter     Filter
	history    historyGenerator
}

type filteredWriter struct {
	*bytes.Buffer

	output  io.Writer
	filter  Filter
	history *history

	contentType string
}

type metaResponse struct {
	Schema string   `json:"$schema"`
	Meta   *history `json:"$meta"`
	Result any      `json:"result"`
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
			"json": {
				Factory: provider.Factory(newJSONFilter),
				Defaults: map[string]string{
					"pretty": "false",
				},
				HelpText: "Generate JSON output",
			},
			"xpath": {
				Factory: provider.Factory(newXPathFilter),
				Defaults: map[string]string{
					"query": "",
				},
				HelpText: "Apply an XPath expression",
			},
			"xml": {
				Factory:  provider.Factory(newXMLFilter),
				Defaults: map[string]string{},
				HelpText: "Generate XML output",
			},
			"yaml": {
				Factory:  provider.Factory(newYAMLFilter),
				Defaults: map[string]string{},
				HelpText: "Generate YAML output",
			},
		},
	}
)

// NewJMESPathFilter provides a filter which uses the given query to search
// a data structure with JSON semantics using JMESPath.
func NewJMESPathFilter(query string) (Filter, error) {
	q, err := jmespath.Compile(query)
	if err != nil {
		return nil, err
	}
	return jmesPathFilter{q}, nil
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

func newJSONFilter(opts jsonFilterOpts) (Filter, error) {
	newEncoder := func(w io.Writer) *json.Encoder {
		e := json.NewEncoder(w)
		if opts.Pretty {
			e.SetIndent("", "  ")
		}
		return e
	}
	return jsonFilter{newEncoder}, nil
}

func newXMLFilter(opts xmlFilterOpts) (Filter, error) {
	var options []xmlquery.OutputOption
	if opts.Pretty {
		options = []xmlquery.OutputOption{
			xmlquery.WithoutPreserveSpace(),
			xmlquery.WithIndentation("  "),
		}
	}
	return xmlFilter{options: options}, nil
}

func newYAMLFilter(opts struct{}) (Filter, error) {
	return yamlFilter{}, nil
}

func newXPathFilter(opts filterOpts) (Filter, error) {
	return NewXPathFilter(opts.Query)
}

// NewXPathFilter generates a new XPath filter
func NewXPathFilter(query string) (Filter, error) {
	q, err := xpath.Compile(query)
	if err != nil {
		return nil, err
	}
	return xpathFilter{q}, nil
}

// NewFilterDownloader applies the filter to an underlying downloader.
func NewFilterDownloader(f Filter, d joehttpclient.Downloader, h historyGenerator) joehttpclient.Downloader {
	if f == nil {
		f = defaultFilter(0)
	}

	return &filteredDownload{
		filter:     f,
		Downloader: d,
		history:    h,
	}
}

func newFilteredWriter(output io.Writer, f Filter, h *history, ct string) *filteredWriter {
	return &filteredWriter{
		Buffer:      new(bytes.Buffer),
		output:      output,
		filter:      f,
		history:     h,
		contentType: ct,
	}
}

func (f *filteredDownload) OpenDownload(ctx context.Context, r *joehttpclient.Response) (io.WriteCloser, error) {
	output, err := f.Downloader.OpenDownload(ctx, r)
	if err != nil {
		return nil, err
	}
	var h *history
	if f.history != nil {
		h, _ = f.history(ctx, r)
	}

	ct := r.Header.Get("Content-Type")
	return newFilteredWriter(output, f.filter, h, ct), nil
}

func (c *filteredWriter) parseResponse(data []byte) (Response, error) {
	ct := c.contentType

	switch {
	case strings.HasPrefix(ct, "application/x-www-form-urlencoded"),
		strings.HasPrefix(ct, "multipart/form-data"):
		panic("not implement: multipart forms")

	case strings.HasPrefix(ct, "application/xml"),
		strings.HasPrefix(ct, "text/xml"):
		return &xmlResponse{data}, nil

	default:
		return &jsonResponse{data, c.history}, nil
	}
}

func (c *filteredWriter) Close() error {
	if closer, ok := c.output.(io.Closer); ok {
		defer closer.Close()
	}

	resp, err := c.parseResponse(c.Buffer.Bytes())
	if err != nil {
		return err
	}

	out, err := c.filter.Search(resp)
	if err != nil {
		return err
	}

	_, err = c.output.Write(out)
	return err
}

func (defaultFilter) Search(resp Response) ([]byte, error) {
	switch resp.(type) {
	case *xmlResponse:
		return unwrap(newXMLFilter(xmlFilterOpts{Pretty: true})).Search(resp)

	case *jsonResponse:
		return unwrap(newJSONFilter(jsonFilterOpts{Pretty: true})).Search(resp)
	}
	return io.ReadAll(resp.Reader())
}

func (x xpathFilter) Search(resp Response) ([]byte, error) {
	doc, err := resp.Document()
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	for _, node := range xmlquery.QuerySelectorAll(doc.(*xmlquery.Node), x.q) {
		err = node.Write(&out, true)
		if err != nil {
			return nil, err
		}
		out.Write([]byte("\n"))
	}
	return out.Bytes(), nil
}

func (j jmesPathFilter) Search(resp Response) ([]byte, error) {
	data, err := resp.Data()
	if err != nil {
		return nil, err
	}

	data, err = j.q.Search(data)
	if err != nil {
		return nil, err
	}
	return json.Marshal(data)
}

func (d digFilter) Search(resp Response) ([]byte, error) {
	data, err := resp.Data()
	if err != nil {
		return nil, err
	}

	for name := range strings.SplitSeq(strings.TrimLeft(string(d), "."), ".") {
		data, err = dig(data, name)
		if err != nil {
			return nil, err
		}
	}

	return json.Marshal(data)
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

func (x xmlFilter) Search(resp Response) ([]byte, error) {
	doc, err := resp.Document()
	if err != nil {
		return nil, err
	}

	// Pretty print the resposne
	var buf bytes.Buffer
	doc.(*xmlquery.Node).WriteWithOptions(
		&buf,
		x.options...,
	)
	return buf.Bytes(), nil
}

func (j jsonFilter) Search(resp Response) ([]byte, error) {
	data, err := resp.Data()
	if err != nil {
		return nil, err
	}

	var results bytes.Buffer
	err = j.e(&results).Encode(data)
	return results.Bytes(), err
}

func (y yamlFilter) Search(resp Response) ([]byte, error) {
	data, err := resp.Data()
	if err != nil {
		return nil, err
	}
	return yaml.Marshal(data)
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

func SetIncludeMetadata(f ...bool) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "include-meta",
			Value:    new(bool),
			HelpText: "Include metadata in the output",
		},
		withBinding((*Client).SetIncludeMetadata, f),
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

func (templateFilter) IncludeMetadata() {}

func (t templateFilter) Search(resp Response) ([]byte, error) {
	text, err := t.loader()
	if err != nil {
		return nil, err
	}
	data, err := resp.Data()
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

	var templateData map[string]any
	if md, ok := data.(*metaResponse); ok {
		templateData = map[string]any{
			"Schema": md.Schema,
			"Meta":   md.Meta,
			"Result": md.Result,
		}
	} else {
		templateData = map[string]any{
			"Result": data,
		}
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

func unwrap[T any](v T, _ any) T {
	return v
}
