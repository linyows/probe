package actions

import (
	"maps"

	"github.com/hashicorp/go-hclog"
)

func runHello(log hclog.Logger, with map[string]any) (map[string]any, error) {
	log.Info("Hello!")

	// Create response data
	res := make(map[string]any)
	maps.Copy(res, with)
	res["status"] = 0   // Always success for hello action (ExitStatusSuccess)
	res["dump"] = false // Don't dump request/response for hello action

	// Return in expected structure
	result := map[string]any{
		"req":    with,
		"res":    res,
		"rt":     "",
		"status": 0,
	}

	return result, nil
}
