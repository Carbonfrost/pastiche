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
	"strconv"
	"strings"

	cli "github.com/Carbonfrost/joe-cli"
	joehttpclient "github.com/Carbonfrost/joe-cli-http/httpclient"
	"github.com/Carbonfrost/joe-cli/extensions/provider"
	"github.com/jmespath/go-jmespath"
)

type Filter interface {
	Search(data any) (any, error)
}

func NewDigFilter(query string) (Filter, error) {
	return digFilter(query), nil
}

func newDig(opts filterOpts) (Filter, error) { // FIXME opts
	return NewDigFilter(opts.Query)
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
		},
	}
)

func NewJMESPathFilter(query string) (Filter, error) {
	return jmespath.Compile(query)
}

func newJMESPath(opts filterOpts) (Filter, error) {
	return NewJMESPathFilter(opts.Query)
}

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

	e := json.NewEncoder(c.output)
	err = e.Encode(res)
	if err != nil {
		return err
	}

	if closer, ok := c.output.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

func (d digFilter) Search(data any) (any, error) {
	var err error
	for _, name := range strings.Split(string(d), ".") {
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

func ListFilters() cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "list-filters",
			HelpText: "List available output filters then exit",
		},
		provider.ListProviders("filter"),
	)
}

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
