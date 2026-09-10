package actions

import (
	"fmt"

	"github.com/hashicorp/go-hclog"
	br "github.com/linyows/probe/browser"
)

func runBrowser(log hclog.Logger, with map[string]any) (map[string]any, error) {
	logParams(log, "received browser action request", with)

	within := br.WithInBrowser(func(s string, i ...any) {
		log.Debug("chromedp", "message", fmt.Sprintf(s, i...))
	})
	before := br.WithBefore(func(req *br.Req) {
		log.Debug("chromedp request prepared", "request", req)
	})
	after := br.WithAfter(func(res *br.Res) {
		log.Debug("chromedp response received", "response", res)
	})

	ret, err := br.Request(with, within, before, after)

	logOutcome(log, "browser request", ret, err)

	return ret, err
}
