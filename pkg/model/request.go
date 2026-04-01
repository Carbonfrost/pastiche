package model

import (
	"io"
	"maps"
	"net/http"
	"net/url"

	"github.com/Carbonfrost/joe-cli-http/httpclient"
	"github.com/Carbonfrost/joe-cli-http/uritemplates"
	e "github.com/Carbonfrost/joe-cli/extensions/expr/expander"
)

type Request struct {
	URL      *url.URL
	Body     io.ReadCloser
	Headers  http.Header
	Vars     map[string]any
	Links    []Link
	Auth     Auth
	Expander e.Interface
}

type RequestOption interface {
	apply(*requestBuilder)
}

func NewRequest(r ResolvedResource, opts ...RequestOption) (*Request, error) {
	b := &requestBuilder{
		baseURL: func() (*uritemplates.URITemplate, error) {
			// Treat server baseURL as a potential URI template
			return uritemplates.Parse(r.Server().BaseURL)
		},
	}

	for _, o := range opts {
		o.apply(b)
	}
	return b.build(r)
}

func WithBaseURL(baseURL *url.URL) RequestOption {
	return requestOption(func(r *requestBuilder) {
		r.baseURL = func() (*uritemplates.URITemplate, error) {
			baseURITemplate, _ := uritemplates.Parse(baseURL.String())
			return baseURITemplate, nil
		}
	})
}

func WithVars(vars map[string]any) RequestOption {
	return requestOption(func(r *requestBuilder) {
		r.vars = vars
	})
}

type requestOption func(*requestBuilder)

func (o requestOption) apply(r *requestBuilder) {
	o(r)
}

type requestBuilder struct {
	baseURL func() (*uritemplates.URITemplate, error)
	vars    map[string]any
}

func (b *requestBuilder) build(r ResolvedResource) (*Request, error) {
	prefix := make([]string, len(r.Lineage()))
	for i, c := range r.Lineage() {
		prefix[i] = c.URITemplate.String()
	}

	baseURITemplate, err := b.baseURL()
	if err != nil {
		return nil, err
	}

	combinedVars := r.Vars()
	maps.Copy(combinedVars, b.vars)

	expander := e.Compose(
		e.Prefix("env", e.Env()),
		e.Prefix("var", e.Map(combinedVars)),
		e.Map(combinedVars),
	)

	links := resolveLinks(
		expandLinks(r.Links(), expander),
		baseURITemplate.String(),
		combinedVars,
	)

	body := func() io.ReadCloser {
		content := bodyContent(r, combinedVars)
		if content == nil {
			return nil
		}

		return io.NopCloser(content.Read())
	}()

	base := baseURITemplate.String()
	u, err := resolveURL(base, prefix, combinedVars)
	if err != nil {
		return nil, err
	}

	return &Request{
		URL:      u,
		Vars:     combinedVars,
		Headers:  expandHeader(r.Headers(), expander),
		Body:     body,
		Links:    links,
		Auth:     expandAuth(r.Auth(), expander),
		Expander: expander,
	}, nil
}

func bodyContent(r ResolvedResource, vars map[string]any) httpclient.Content {
	if r.Endpoint().Form != nil {
		return newFormContent(r.Endpoint().Form, vars)
	}
	if r.Endpoint().Body != "" {
		return newTemplateContent(r.Endpoint().Body, vars)
	}
	if r.Endpoint().RawBody != "" {
		return newRawContent(r.Endpoint().RawBody)
	}
	if r.Resource().Form != nil {
		return newFormContent(r.Resource().Form, vars)
	}
	if r.Resource().Body != "" {
		return newTemplateContent(r.Resource().Body, vars)
	}
	if r.Resource().RawBody != "" {
		return newRawContent(r.Resource().RawBody)
	}

	return nil
}
