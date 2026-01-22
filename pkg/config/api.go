// Copyright 2025, 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

type File struct {
	*Service

	Services []Service `json:"services,omitempty"`
}

type Service struct {
	Name        string         `json:"name"`
	Source      string         `json:"source,omitempty"`
	Title       string         `json:"title,omitempty"`
	Description string         `json:"description,omitempty"`
	Servers     []Server       `json:"servers,omitempty"`
	Resources   []Resource     `json:"resources,omitempty"`
	Links       []Link         `json:"links,omitempty"`
	Vars        map[string]any `json:"vars,omitempty"`
	Client      *Client        `json:"client,omitempty"`
}

type Server struct {
	Name        string         `json:"name"`
	Source      string         `json:"source,omitempty"`
	Description string         `json:"description"`
	Title       string         `json:"title"`
	BaseURL     string         `json:"baseUrl"`
	Headers     Header         `json:"headers"`
	Links       []Link         `json:"links,omitempty"`
	Vars        map[string]any `json:"vars,omitempty"`
}

type Resource struct {
	Name        string         `json:"name,omitempty"`
	Source      string         `json:"source,omitempty"`
	Title       string         `json:"title,omitempty"`
	Description string         `json:"description,omitempty"`
	Resources   []Resource     `json:"resources,omitempty"`
	URI         string         `json:"uri,omitempty"`
	Headers     Header         `json:"headers,omitempty"`
	Form        Form           `json:"form,omitempty"`
	Links       []Link         `json:"links,omitempty"`
	Get         *Endpoint      `json:"get,omitempty"`
	Put         *Endpoint      `json:"put,omitempty"`
	Post        *Endpoint      `json:"post,omitempty"`
	Delete      *Endpoint      `json:"delete,omitempty"`
	Options     *Endpoint      `json:"options,omitempty"`
	Head        *Endpoint      `json:"head,omitempty"`
	Trace       *Endpoint      `json:"trace,omitempty"`
	Patch       *Endpoint      `json:"patch,omitempty"`
	Query       *Endpoint      `json:"query,omitempty"`
	Body        any            `json:"body,omitempty"`
	RawBody     any            `json:"rawBody,omitempty"`
	Vars        map[string]any `json:"vars,omitempty"`
}

type Client struct {
	HTTP *HTTPClient `json:"http,omitempty"`
	GRPC *GRPCClient `json:"grpc,omitempty"`
}

type HTTPClient struct {
}

type GRPCClient struct {
	DisableReflection bool   `json:"disableReflection,omitzero"`
	ProtoSet          string `json:"protoset,omitempty"`
	Plaintext         bool   `json:"plaintext,omitzero"`
}

type Endpoint struct {
	Name        string         `json:"name,omitempty"`
	Title       string         `json:"title,omitempty"`
	Source      string         `json:"source,omitempty"`
	Description string         `json:"description,omitempty"`
	Headers     Header         `json:"headers,omitempty"`
	Form        Form           `json:"form,omitempty"`
	Links       []Link         `json:"links,omitempty"`
	Body        any            `json:"body,omitempty"`
	RawBody     any            `json:"rawBody,omitempty"`
	Vars        map[string]any `json:"vars,omitempty"`
}

type Link struct {
	HRef     string `json:"href,omitempty"`
	HRefLang string `json:"hrefLang,omitempty"`
	Audience string `json:"audience,omitempty"`
	Rel      string `json:"rel,omitempty"`
	Title    string `json:"title,omitempty"`
	Type     string `json:"title,omitempty"`
}
