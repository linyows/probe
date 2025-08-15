package http

import (
	"errors"
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
	// Use default truncate length, can be overridden by caller
	truncateLength := probe.MaxLogStringLength

	// Truncate long parameters for logging to prevent log bloat
	truncatedParams := probe.TruncateMapStringString(with, truncateLength)
	a.log.Debug("received request parameters", "params", truncatedParams)

	if err := updateMap(with); err != nil {
		a.log.Error("failed to update request parameters", "error", err)
		return map[string]string{}, err
	}

	// Truncate again after processing
	truncatedUpdatedParams := probe.TruncateMapStringString(with, truncateLength)
	a.log.Debug("updated request parameters", "params", truncatedUpdatedParams)

	before := http.WithBefore(func(req *hp.Request) {
		a.log.Debug("http request prepared", "request", req)
	})
	after := http.WithAfter(func(res *hp.Response) {
		a.log.Debug("http response received", "response", res)
	})
	ret, err := http.Request(with, before, after)

	if err != nil {
		a.log.Error("http request failed", "error", err)
	} else {
		// Truncate result for logging to prevent log bloat
		truncatedResult := probe.TruncateMapStringString(ret, truncateLength)
		a.log.Debug("http request completed successfully", "result", truncatedResult)
	}

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
		if err = probe.ConvertBodyToJson(data); err != nil {
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
// or `get: https://api.example.com/users` to `method: GET` and `url: https://api.example.com/users`
func replaceMethodAndURL(data map[string]string) error {
	for _, method := range httpMethods {
		lowerMethod := strings.ToLower(method)
		route, ok := data[lowerMethod]
		if !ok {
			continue
		}

		data["method"] = method
		delete(data, lowerMethod)

		// If route is a complete URL (starts with http:// or https://), use it directly
		if strings.HasPrefix(route, "http://") || strings.HasPrefix(route, "https://") {
			data["url"] = route
		} else {
			// If route is a relative path, combine with base URL
			baseURL, ok := data["url"]
			if !ok {
				return errors.New("url is missing for relative path")
			}

			// renew url as full-url
			u, err := url.Parse(baseURL)
			if err != nil {
				return err
			}
			u.Path = path.Join(u.Path, route)
			data["url"] = u.String()
		}

		break
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
