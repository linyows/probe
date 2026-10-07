package http

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	hp "net/http"
	"os"

	"github.com/pb33f/libopenapi"
	validator "github.com/pb33f/libopenapi-validator"
	verrors "github.com/pb33f/libopenapi-validator/errors"
	"github.com/pb33f/libopenapi-validator/helpers"
)

// contract checks requests and responses against an OpenAPI document.
type contract struct {
	v validator.Validator
	// request is whether the request is checked as well as the response. A
	// step that sends what the document does not allow on purpose, to see
	// it rejected, turns it off.
	request bool
}

// takeOpenAPI removes openapi from the parameters and returns the contract it
// names, or nil when the step checks nothing: when openapi is missing, or
// false, as a step writes it to leave out a check its job's defaults ask for.
// The document is read before the request is sent, so that one that cannot
// be read fails the step without sending anything.
func takeOpenAPI(m map[string]any) (*contract, error) {
	v, ok := m["openapi"]
	if !ok {
		return nil, nil
	}
	delete(m, "openapi")

	switch o := v.(type) {
	case bool:
		if o {
			return nil, errors.New("openapi must be a map with spec, or false")
		}
		return nil, nil
	case map[string]any:
		for k := range o {
			if k != "spec" && k != "request" {
				return nil, fmt.Errorf("openapi takes spec and request, not %s", k)
			}
		}
		path, ok := o["spec"].(string)
		if !ok || path == "" {
			return nil, errors.New("openapi.spec must be the path of an OpenAPI document")
		}
		request := true
		if r, ok := o["request"]; ok {
			if request, ok = r.(bool); !ok {
				return nil, errors.New("openapi.request must be true or false")
			}
		}
		c, err := loadContract(path)
		if err != nil {
			return nil, err
		}
		c.request = request
		return c, nil
	default:
		return nil, errors.New("openapi must be a map with spec, or false")
	}
}

// loadContract reads the OpenAPI document at path.
func loadContract(path string) (*contract, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("openapi.spec: %w", err)
	}
	doc, err := libopenapi.NewDocument(data)
	if err != nil {
		return nil, fmt.Errorf("openapi.spec: %s: %w", path, err)
	}
	v, errs := validator.NewValidator(doc)
	if len(errs) > 0 {
		return nil, fmt.Errorf("openapi.spec: %s: %w", path, errors.Join(errs...))
	}
	return &contract{v: v}, nil
}

// check returns what in the request and the response the document does not
// allow, as the violations a step reports, or an empty list when it allows
// all of it. req is the request the step sent, with reqBody, the body sent;
// res is the response, with resBody, its body, which has been read.
//
// The request is checked as the step sent it, before any redirect, which
// the client made rather than the step. The response is matched to an
// operation by the request that received it, the last one when the request
// was redirected.
func (c *contract) check(req *hp.Request, reqBody []byte, res *hp.Response, resBody []byte) []any {
	out := []any{}
	if c.request {
		out = append(out, c.checkRequest(req, reqBody)...)
	}

	r := *res
	r.Body = io.NopCloser(bytes.NewReader(resBody))
	_, errs := c.v.ValidateHttpResponse(res.Request, &r)
	return append(out, violations("response", errs)...)
}

// checkRequest returns what in the request the document does not allow. A
// request to an operation the document does not have is left to the check
// of the response, which reports it, so that it is not reported twice.
func (c *contract) checkRequest(req *hp.Request, body []byte) []any {
	r := req.Clone(req.Context())
	r.Body = io.NopCloser(bytes.NewReader(body))

	_, errs := c.v.ValidateHttpRequest(r)
	kept := errs[:0]
	for _, e := range errs {
		if e.ValidationType != helpers.PathValidation {
			kept = append(kept, e)
		}
	}
	return violations("request", kept)
}

// violations turns the errors of the validator into the violations a step
// reports, found in the request or the response as in says: one for each
// failure of a schema, which names the field, and one for each other error.
func violations(in string, errs []*verrors.ValidationError) []any {
	out := []any{}
	for _, e := range errs {
		if len(e.SchemaValidationErrors) == 0 {
			out = append(out, violation(in, e.Message, e.Reason, ""))
			continue
		}
		for _, se := range e.SchemaValidationErrors {
			out = append(out, violation(in, e.Message, se.Reason, se.FieldPath))
		}
	}
	return out
}

// violation is one violation, found in the request or the response.
func violation(in, message, reason, field string) map[string]any {
	v := map[string]any{
		"in":      in,
		"message": message,
	}
	if reason != "" {
		v["reason"] = reason
	}
	if field != "" {
		v["field"] = field
	}
	return v
}
