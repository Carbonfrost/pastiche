// Copyright 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/antchfx/xmlquery"
)

//counterfeiter:generate . Response

type Response interface {
	Data() (any, error)
	Document() (any, error)
	Reader() io.Reader
}

type jsonResponse struct {
	data    []byte
	history *history
}

type xmlResponse struct {
	data []byte
}

func (x *xmlResponse) Reader() io.Reader {
	return bytes.NewReader(x.data)
}

func (x *xmlResponse) Document() (any, error) {
	return xmlquery.Parse(bytes.NewReader(x.data))
}

func (x *xmlResponse) Data() (any, error) {
	return nil, fmt.Errorf("XML response cannot be filtered as data")
}

func (j *jsonResponse) Reader() io.Reader {
	return bytes.NewReader(j.data)
}

func (*jsonResponse) Document() (any, error) {
	return nil, fmt.Errorf("JSON response cannot be filtered as document")
}

func (j *jsonResponse) Data() (any, error) {
	var data any
	err := json.Unmarshal(j.data, &data)
	if err != nil {
		return nil, err
	}

	// If the history log was present, then wrap it in a metadata response
	if j.history != nil {
		return &metaResponse{
			Meta:   j.history,
			Result: data,
		}, nil
	}

	return data, nil
}
