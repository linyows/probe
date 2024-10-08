package http

import (
	"io"
	hp "net/http"
	"net/url"

	"github.com/linyows/probe"
)

type TransportOptions struct {
	Timeout      int `map:"timeout"`
	MaxIdleConns int `map:"max_idle_conns"`
}

type Req struct {
	Host    string            `map:"host" validate:"required"`
	Path    string            `map:"path" validate:"required"`
	Ver     string            `map:"ver"`
	Method  string            `map:"method" validate:"required"`
	Accept  string            `map:"accept"`
	UA      string            `map:"ua"`
	Data    []string          `map:"data"`
	Headers map[string]string `map:"headers"`
}

func NewReq(p map[string]string) (*Req, error) {
	r := &Req{
		Ver:    "HTTP/1.1",
		Method: "GET",
		Accept: "*/*",
		UA:     "probe/1.0.0",
	}

	if err := probe.AssignStruct(p, r); err != nil {
		return nil, err
	}

	return r, nil
}

func (r *Req) Do() ([]byte, error) {
	url, err := url.JoinPath(r.Host, r.Path)
	if err != nil {
		return []byte{}, err
	}

	cl := &hp.Client{}

	// d := strings.NewReader(*data)
	req, err := hp.NewRequest(r.Method, url, nil)
	if err != nil {
		return []byte{}, err
	}

	req.Header.Set("Accept", r.Accept)
	req.Header.Set("User-Agent", r.UA)

	res, err := cl.Do(req)
	if err != nil {
		return []byte{}, err
	}
	defer res.Body.Close()

	return io.ReadAll(res.Body)
}
