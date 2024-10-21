package http

import (
	"bytes"
	"errors"
	"io/ioutil"
	hp "net/http"
	"strings"

	"github.com/linyows/probe"
)

type TransportOptions struct {
	Timeout      int `map:"timeout"`
	MaxIdleConns int `map:"max_idle_conns"`
}

type Req struct {
	URL     string            `map:"url" validate"required"`
	Method  string            `map:"method" validate:"required"`
	Proto   string            `map:"ver"`
	Headers map[string]string `map:"headers"`
	Body    []byte            `map:"body"`
	cb      *Callback
}

type Res struct {
	Headers map[string]string `map:"headers"`
	Body    []byte            `map:"body"`
}

type Result struct {
	Req Req `map:"req"`
	Res Res `map:"res"`
}

func NewReq() *Req {
	return &Req{
		Method: "GET",
		Proto:  "HTTP/1.1",
		Headers: map[string]string{
			"Accept":     "*/*",
			"User-Agent": "probe-http/1.0.0",
		},
	}
}

func (r *Req) Do() (*Result, error) {
	if r.URL == "" {
		return nil, errors.New("Req.URL is required")
	}

	req, err := hp.NewRequest(r.Method, r.URL, bytes.NewBuffer(r.Body))
	if err != nil {
		return nil, err
	}

	for k, v := range r.Headers {
		req.Header.Set(k, v)
	}

	// callback
	if r.cb != nil && r.cb.before != nil {
		r.cb.before(req)
	}

	cl := &hp.Client{}
	res, err := cl.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	// callback
	if r.cb != nil && r.cb.after != nil {
		r.cb.after(res)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	headers := make(map[string]string)
	for k, v := range res.Header {
		// examples:
		//   Set-Cookie: sessionid=abc123; Path=/; HttpOnly
		//   Accept: text/html, application/xhtml+xml, application/xml;q=0.9
		headers[k] = strings.Join(v, ", ")
	}

	return &Result{
		Req: *r,
		Res: Res{Headers: headers, Body: body},
	}, nil
}

type Option func(*Callback)

type Callback struct {
	before func(req *hp.Request)
	after  func(res *hp.Response)
}

func Request(data map[string]string, opts ...Option) (map[string]string, error) {
	m := probe.UnflattenInterface(data)
	r := NewReq()

	cb := &Callback{}
	for _, opt := range opts {
		opt(cb)
	}
	r.cb = cb

	if err := probe.MapToStructByTags(m, r); err != nil {
		return map[string]string{}, err
	}

	ret, err := r.Do()
	if err != nil {
		return map[string]string{}, err
	}

	mapRet, err := probe.StructToMapByTags(ret)
	if err != nil {
		return map[string]string{}, err
	}

	return probe.FlattenInterface(mapRet), nil
}

func WithBefore(f func(req *hp.Request)) Option {
	return func(c *Callback) {
		c.before = f
	}
}

func WithAfter(f func(res *hp.Response)) Option {
	return func(c *Callback) {
		c.after = f
	}
}