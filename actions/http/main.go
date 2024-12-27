package http

import (
	"encoding/json"
	"errors"
	"fmt"
	hp "net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/linyows/probe"
	"github.com/linyows/probe/http"
)

var httpMethods = []string{
	hp.MethodGet,
	hp.MethodHead,
	hp.MethodPost,
	hp.MethodPut,
	hp.MethodPatch,
	hp.MethodDelete,
	hp.MethodConnect,
	hp.MethodOptions,
	hp.MethodTrace,
}

type Action struct {
	log hclog.Logger
}

func (a *Action) Run(args []string, with map[string]string) (map[string]string, error) {
	a.log.Debug(fmt.Sprintf("received: %#v", with))

	if err := updateMap(with); err != nil {
		return map[string]string{}, err
	}

	a.log.Debug(fmt.Sprintf("updated: %#v", with))

	before := http.WithBefore(func(req *hp.Request) {
		a.log.Debug(fmt.Sprintf("http.Request: %#v", req))
	})
	after := http.WithAfter(func(res *hp.Response) {
		a.log.Debug(fmt.Sprintf("http.Response: %#v", res))
	})
	ret, err := http.Request(with, before, after)

	a.log.Debug(fmt.Sprintf("return: %#v", ret))
	a.log.Debug(fmt.Sprintf("error: %#v", err))

	return ret, err
}

func Serve() {
	log := hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Debug,
		Output:     os.Stderr,
		JSONFormat: true,
	})

	pl := &probe.ActionsPlugin{
		Impl: &Action{log: log},
	}

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: probe.Handshake,
		Plugins:         map[string]plugin.Plugin{"actions": pl},
		GRPCServer:      plugin.DefaultGRPCServer,
	})
}

func updateMap(data map[string]string) error {
	var err error
	if err = replaceMethodAndURL(data); err != nil {
		return err
	}
	v, exists := data["headers__content-type"]
	if exists && v == "application/json" {
		if err = convertBodyToJson(data); err != nil {
			return err
		}
	} else {
		if err = convertBodyToTextWithContentType(data); err != nil {
			return err
		}
	}

	return nil
}

// replace `get: /foo/bar` and `url: http://localhost:8000` to `method: GET` and `url: http://localhost:8000/foo/bar`
func replaceMethodAndURL(data map[string]string) error {
	for _, method := range httpMethods {
		lowerMethod := strings.ToLower(method)
		route, ok := data[lowerMethod]
		if !ok {
			continue
		}

		data["method"] = method
		delete(data, lowerMethod)

		// get the base-url from url
		baseURL, ok := data["url"]
		if !ok {
			return errors.New("Error: url is missing in the map")
		}

		// renew url as full-url
		u, err := url.Parse(baseURL)
		if err != nil {
			return err
		}
		u.Path = path.Join(u.Path, route)
		data["url"] = u.String()

		break
	}

	return nil
}

func convertBodyToJson(data map[string]string) error {
	values := map[string]any{}

	for key, value := range data {
		if strings.HasPrefix(key, "body__") {
			newKey := strings.TrimPrefix(key, "body__")
			// If it is a numeric string, set the type to number
			num, err := strconv.Atoi(value)
			if err == nil {
				values[newKey] = num
			} else {
				values[newKey] = value
			}
			delete(data, key)
		}
	}

	if len(values) > 0 {
		j, err := json.Marshal(values)
		if err != nil {
			return err
		}
		data["body"] = string(j)
	}

	return nil
}

func convertBodyToTextWithContentType(data map[string]string) error {
	values := url.Values{}

	for key, value := range data {
		if strings.HasPrefix(key, "body__") {
			newKey := strings.TrimPrefix(key, "body__")
			values.Add(newKey, value)
			delete(data, key)
		}
	}

	if len(values) > 0 {
		data["body"] = values.Encode()
		data["headers__content-type"] = "application/x-www-form-urlencoded"
	}

	return nil
}
