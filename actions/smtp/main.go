package smtp

import (
	"fmt"
	"os"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/linyows/probe"
	"github.com/linyows/probe/mail"
)

type Action struct {
	log hclog.Logger
}

type SMTPReq struct {
	Addr       string `map:"addr"`
	From       string `map:"from"`
	To         string `map:"to"`
	Subject    string `map:"subject"`
	MyHostname string `map:"myhostname"`
	Session    int    `map:"session"`
	Message    int    `map:"message"`
	Length     int    `map:"length"`
}

type SMTPRes struct {
	Code      int    `map:"code"`
	Sent      int    `map:"sent"`
	Failed    int    `map:"failed"`
	Total     int    `map:"total"`
	Error     string `map:"error"`
}

type SMTPResult struct {
	Req    SMTPReq       `map:"req"`
	Res    SMTPRes       `map:"res"`
	RT     time.Duration `map:"rt"`
	Status int           `map:"status"`
}

func (a *Action) Run(args []string, with map[string]string) (map[string]string, error) {
	start := time.Now()
	
	// Create structured request
	req := &SMTPReq{}
	unflattenedWith := probe.UnflattenInterface(with)
	err := probe.MapToStructByTags(unflattenedWith, req)
	if err != nil {
		return a.createErrorResult(start, req, err)
	}

	// Validate required parameters
	if req.Addr == "" {
		return a.createErrorResult(start, req, fmt.Errorf("addr parameter is required"))
	}
	if req.From == "" {
		return a.createErrorResult(start, req, fmt.Errorf("from parameter is required"))
	}
	if req.To == "" {
		return a.createErrorResult(start, req, fmt.Errorf("to parameter is required"))
	}

	a.log.Debug("sending email", "from", req.From, "to", req.To, "subject", req.Subject)

	m, err := mail.NewBulk(with)
	if err != nil {
		return a.createErrorResult(start, req, err)
	}

	// Execute mail delivery
	deliveryResult := m.DeliverWithResult()
	duration := time.Since(start)

	// Determine status based on delivery success
	status := 1 // default to failure
	if deliveryResult.Failed == 0 && deliveryResult.Sent > 0 {
		status = 0 // success if no failures and at least one sent
	}

	result := &SMTPResult{
		Req: *req,
		Res: SMTPRes{
			Code:   status,
			Sent:   deliveryResult.Sent,
			Failed: deliveryResult.Failed,
			Total:  deliveryResult.Total,
			Error:  deliveryResult.Error,
		},
		RT:     duration,
		Status: status,
	}

	// Convert to map[string]string using probe's mapping function
	mapResult, err := probe.StructToMapByTags(result)
	if err != nil {
		return a.createErrorResult(start, req, err)
	}

	a.log.Debug("email delivery completed", "sent", deliveryResult.Sent, "failed", deliveryResult.Failed, "status", status)

	return probe.FlattenInterface(mapResult), nil
}

func (a *Action) createErrorResult(start time.Time, req *SMTPReq, err error) (map[string]string, error) {
	duration := time.Since(start)

	result := &SMTPResult{
		Req: *req,
		Res: SMTPRes{
			Code:   1,
			Sent:   0,
			Failed: 0,
			Total:  0,
			Error:  err.Error(),
		},
		RT:     duration,
		Status: 1, // failure
	}

	mapResult, mapErr := probe.StructToMapByTags(result)
	if mapErr != nil {
		return map[string]string{}, mapErr
	}

	return probe.FlattenInterface(mapResult), err
}

func Serve() {
	log := hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Info,
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
