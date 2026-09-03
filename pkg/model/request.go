package model

import (
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"slices"

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
			if r.Server() == nil {
				return nil, nil
			}
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
		prefix[i] = fmt.Sprint(c.URITemplate)
	}

	baseURITemplate, err := b.baseURL()
	if err != nil {
		return nil, err
	}

	combinedVars := resolveVars(r)
	maps.Copy(combinedVars, b.vars)

	expander := e.Compose(
		e.Prefix("env", e.Env()),
		e.Prefix("secret", newSecretExpander(r.Secrets())),
		e.Prefix("var", e.Map(combinedVars)),
		e.Map(combinedVars),
	)

	links := resolveLinks(
		expandLinks(resolveLinks2(r), expander),
		fmt.Sprint(baseURITemplate),
		combinedVars,
	)

	body := func() io.ReadCloser {
		content := bodyContent(r, combinedVars)
		if content == nil {
			return nil
		}

		return io.NopCloser(content.Read())
	}()

	base := fmt.Sprint(baseURITemplate)
	u, err := resolveURL(base, prefix, combinedVars)
	mergeQuery(u, expandHeader(resolveQuery(r), expander))

	if err != nil {
		return nil, err
	}

	return &Request{
		URL:      u,
		Vars:     combinedVars,
		Headers:  expandHeader(resolveHeaders(r), expander),
		Body:     body,
		Links:    links,
		Auth:     expandAuth(resolveAuth(r), expander),
		Expander: expander,
	}, nil
}

// bodyContent obtains the content of the request body from the most specific
// layer which defines one.  A mixin, which the caller selected explicitly, is
// more specific than the endpoint or the resource.
func bodyContent(r ResolvedResource, vars map[string]any) httpclient.Content {
	for _, m := range slices.Backward(r.Mixins()) {
		if content := newContent(m.Form, m.Body, m.RawBody, vars); content != nil {
			return content
		}
	}
	if ep := r.Endpoint(); ep != nil {
		if content := newContent(ep.Form, ep.Body, ep.RawBody, vars); content != nil {
			return content
		}
	}
	if res := r.Resource(); res != nil {
		return newContent(res.Form, res.Body, res.RawBody, vars)
	}
	return nil
}

// newContent obtains the content which one layer defines, preferring the form
// over the templated body over the raw body
func newContent(form Values, body any, rawBody any, vars map[string]any) httpclient.Content {
	if form != nil {
		return newFormContent(form.toMap(), vars)
	}
	if body != "" {
		return newTemplateContent(body, vars)
	}
	if rawBody != "" {
		return newRawContent(rawBody)
	}
	return nil
}
