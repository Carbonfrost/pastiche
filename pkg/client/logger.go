// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"context"
	"io"

	"github.com/Carbonfrost/joe-cli-http/httpclient"
	"github.com/Carbonfrost/pastiche/pkg/internal/log"
	"github.com/Carbonfrost/pastiche/pkg/workspace/logs"
)

type historyDownloader struct {
	httpclient.Downloader

	factory historyGenerator
}

type historyWriter struct {
	io.Writer
	log     *logs.Log
	output  io.Closer
	history *logs.Entry
}

type historyGenerator func(context.Context, *httpclient.Response) (history *logs.Entry, responseBody io.Writer)

func newHistoryDownloader(d httpclient.Downloader, factory historyGenerator) httpclient.Downloader {
	return historyDownloader{
		Downloader: d,
		factory:    factory,
	}
}

func (h historyDownloader) OpenDownload(ctx context.Context, r *httpclient.Response) (io.WriteCloser, error) {
	output, err := h.Downloader.OpenDownload(ctx, r)
	if err != nil {
		return nil, err
	}

	history, responseBody := h.factory(ctx, r)
	c, ok := output.(io.Closer)
	if !ok {
		c = io.NopCloser(nil)
	}

	return &historyWriter{
		log:     logs.FromContext(ctx),
		Writer:  io.MultiWriter(output, responseBody),
		output:  c,
		history: history,
	}, nil
}

func (w *historyWriter) Close() error {
	// TODO Improve handling of errors
	if err := w.log.Append(w.history); err != nil {
		log.Warn(err)
		return nil
	}

	return w.output.Close()
}
