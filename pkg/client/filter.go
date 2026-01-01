// Copyright 2023, 2025 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"io"

	cli "github.com/Carbonfrost/joe-cli"
	joehttpclient "github.com/Carbonfrost/joe-cli-http/httpclient"
	"github.com/Carbonfrost/joe-cli/extensions/provider"
	"github.com/jmespath/go-jmespath"
)

type Filter interface {
	Search(data any) (any, error)
}

type jmesPathFilterOpts struct {
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
					"query": ".",
				},
				HelpText: "Use JMES Path to select matching JSON data",
			},
		},
	}
)

func NewJMESPathFilter(query string) (Filter, error) {
	return jmespath.Compile(query)
}

func newJMESPath(opts jmesPathFilterOpts) (Filter, error) {
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

func (j *jmesPathFilterOpts) UnmarshalText(data []byte) error {
	j.Query = string(data)
	return nil
}
