// Copyright 2025, 2026 The Pastiche Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

type File struct {
	Schema string `json:"$schema,omitempty"`

	*Service

	Services []Service `json:"services,omitempty"`
	name     string
}

type Service struct {
	Schema      string         `json:"$schema,omitempty"`
	Name        string         `json:"name"`
	Comment     string         `json:"comment,omitempty"`
	Source      string         `json:"source,omitempty"`
	Title       string         `json:"title,omitempty"`
	Description string         `json:"description,omitempty"`
	Servers     []Server       `json:"servers,omitempty"`
	Resources   []Resource     `json:"resources,omitempty"`
	Links       []Link         `json:"links,omitempty"`
	Vars        map[string]any `json:"vars,omitempty"`
	Client      *Client        `json:"client,omitempty"`
	Auth        *Auth          `json:"auth,omitempty"`
}

type Server struct {
	Schema      string         `json:"$schema,omitempty"`
	Name        string         `json:"name"`
	Comment     string         `json:"comment,omitempty"`
	Source      string         `json:"source,omitempty"`
	Description string         `json:"description,omitempty"`
	Title       string         `json:"title,omitempty"`
	BaseURL     string         `json:"baseUrl"`
	Headers     Header         `json:"headers,omitempty"`
	Links       []Link         `json:"links,omitempty"`
	Vars        map[string]any `json:"vars,omitempty"`
	Auth        *Auth          `json:"auth,omitempty"`
}

type Resource struct {
	Schema      string         `json:"$schema,omitempty"`
	Name        string         `json:"name,omitempty"`
	Comment     string         `json:"comment,omitempty"`
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
	Auth        *Auth          `json:"auth,omitempty"`
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

type Auth struct {
	Basic *BasicAuth `json:"basic,omitempty"`
}

type BasicAuth struct {
	User     string `json:"user,omitempty"`
	Password string `json:"password,omitempty"`
}

type Endpoint struct {
	Schema      string         `json:"$schema,omitempty"`
	Name        string         `json:"name,omitempty"`
	Comment     string         `json:"comment,omitempty"`
	Title       string         `json:"title,omitempty"`
	Source      string         `json:"source,omitempty"`
	Description string         `json:"description,omitempty"`
	Headers     Header         `json:"headers,omitempty"`
	Form        Form           `json:"form,omitempty"`
	Links       []Link         `json:"links,omitempty"`
	Body        any            `json:"body,omitempty"`
	RawBody     any            `json:"rawBody,omitempty"`
	Vars        map[string]any `json:"vars,omitempty"`
	Auth        *Auth          `json:"auth,omitempty"`
}

type Link struct {
	HRef       string `json:"href,omitempty"`
	HRefLang   string `json:"hrefLang,omitempty"`
	Audience   string `json:"audience,omitempty"`
	Rel        string `json:"rel,omitempty"`
	Title      string `json:"title,omitempty"`
	Type       string `json:"type,omitempty"`
	IsTemplate bool   `json:"isTemplate,omitempty"`
}

func (f *File) Name() string {
	return f.name
}

func (f *File) SetName(name string) {
	f.name = name
}
