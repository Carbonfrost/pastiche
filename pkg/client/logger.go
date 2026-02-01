// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/Carbonfrost/joe-cli-http/httpclient"
	"github.com/Carbonfrost/pastiche/pkg/internal/log"
)

type (
	history struct {
		Timestamp time.Time       `json:"ts"`
		Spec      []string        `json:"spec"`
		URL       string          `json:"url"`
		Server    string          `json:"server,omitempty"`
		Response  historyResponse `json:"response"`
		Request   historyRequest  `json:"request"`
		Vars      map[string]any  `json:"vars,omitempty"`
		BaseURL   *string         `json:"baseUrl"`
	}

	historyResponse struct {
		Headers    map[string][]string  `json:"headers,omitempty"`
		Status     string               `json:"status"`
		StatusCode int                  `json:"statusCode"`
		Body       *historyResponseBody `json:"body"`
	}

	historyResponseBody struct {
		buffer *bytes.Buffer
	}

	historyRequest struct {
		Method  string              `json:"method"`
		Headers map[string][]string `json:"headers,omitempty"`
	}
)

type historyDownloader struct {
	httpclient.Downloader

	factory historyGenerator
}

type historyWriter struct {
	io.Writer
	output  io.Closer
	history *history
}

type historyGenerator func(context.Context, *httpclient.Response) (history *history, responseBody io.Writer)

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
		Writer:  io.MultiWriter(output, responseBody),
		output:  c,
		history: history,
	}, nil
}

func (w *historyWriter) Close() error {
	// TODO This directory should be the workspace
	logDir := filepath.Join(".pastiche", "logs")
	os.MkdirAll(logDir, 0755)
	fileName := filepath.Join(logDir, fmt.Sprintf("requests.%s.json", time.Now().Format("2006-01-02")))

	// TODO Improve handling of errors
	f, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil
	}

	defer f.Close()

	logLines, err := json.Marshal(w.history)
	if err != nil {
		log.Warn(err)
		return nil
	}
	_, err = f.Write(logLines)
	if err != nil {
		log.Warn(err)
		return nil
	}

	_, _ = f.Write([]byte("\n"))

	return w.output.Close()
}

func (h historyResponseBody) MarshalJSON() ([]byte, error) {
	if json.Valid(h.buffer.Bytes()) {
		return json.Marshal(map[string]any{
			"json": json.RawMessage(h.buffer.Bytes()),
		})
	}

	return json.Marshal(map[string]any{
		"text": string(h.buffer.Bytes()),
	})
}

func sprintURL(u *url.URL) *string {
	if u == nil {
		return nil
	}
	s := u.String()
	return &s
}

var _ json.Marshaler = (*historyResponseBody)(nil)
