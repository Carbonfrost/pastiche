package httpclient

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

	resolver *serviceResolver
}

type historyWriter struct {
	io.Writer
	output  io.Closer
	history *history
}

func newHistoryDownloader(d httpclient.Downloader, s *serviceResolver) httpclient.Downloader {
	return historyDownloader{
		Downloader: d,
		resolver:   s,
	}
}

func (h historyDownloader) OpenDownload(ctx context.Context, r *httpclient.Response) (io.WriteCloser, error) {
	output, err := h.Downloader.OpenDownload(ctx, r)
	if err != nil {
		return nil, err
	}

	req, _ := h.resolver.resolveRequest(ctx)
	var vars map[string]any
	if req != nil {

		vars = req.Vars() // TODO Would be better to separate input vars from compiled
	}
	var responseBody bytes.Buffer
	history := &history{
		Timestamp: time.Now(), // TODO To be persnickety, should be the exact request timing
		URL:       fmt.Sprint(r.Request.URL),
		Spec:      *h.resolver.root(ctx),
		Server:    h.resolver.server(ctx),
		Response: historyResponse{
			Headers:    r.Header,
			Status:     r.Status,
			StatusCode: r.StatusCode,
			Body:       &historyResponseBody{&responseBody},
		},
		Request: historyRequest{
			Headers: r.Request.Header,
			Method:  r.Request.Method,
		},
		Vars:    vars,
		BaseURL: sprintURL(h.resolver.base),
	}

	c, ok := output.(io.Closer)
	if !ok {
		c = io.NopCloser(nil)
	}

	return &historyWriter{
		Writer:  io.MultiWriter(output, &responseBody),
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

	logLines, _ := json.Marshal(w.history)
	_, _ = f.Write(logLines)
	_, _ = f.Write([]byte("\n"))
	_ = f.Close()

	return w.output.Close()
}

func (h historyResponseBody) MarshalJSON() ([]byte, error) {
	kind := "text"
	if json.Valid(h.buffer.Bytes()) {
		kind = "json"
	}

	return json.Marshal(map[string]any{
		kind: json.RawMessage(h.buffer.Bytes()),
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
