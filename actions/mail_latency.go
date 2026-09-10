package actions

import (
	"bytes"
	"errors"
	"os"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/linyows/probe/mail"
)

func runMailLatency(log hclog.Logger, with map[string]any) (map[string]any, error) {
	// Validate that required parameters are provided
	if len(with) == 0 {
		return map[string]any{}, errors.New("mail-latency action requires parameters in 'with' section. Please specify 'mail_dir' and 'output_dir' parameters")
	}

	mailDir, ok := with["mail_dir"].(string)
	if !ok || mailDir == "" {
		return map[string]any{}, errors.New("mail-latency action requires 'mail_dir' parameter in 'with' section")
	}

	// Get required output_dir parameter
	outputDir, ok := with["output_dir"].(string)
	if !ok || outputDir == "" {
		return map[string]any{}, errors.New("mail-latency action requires 'output_dir' parameter in 'with' section")
	}

	log.Debug("received mail-latency request", "mail_dir", mailDir, "output_dir", outputDir)

	// Create a buffer to capture CSV output
	var csvBuffer bytes.Buffer

	// Measure execution time
	start := time.Now()
	// Get latencies and write to buffer
	if err := mail.GetLatencies(mailDir, &csvBuffer); err != nil {
		log.Error("mail-latency request failed", "error", err)
		return map[string]any{}, err
	}
	rt := time.Since(start)

	csvContent := csvBuffer.String()

	// Generate filename with timestamp
	timestamp := time.Now().Format("20060102-150405")
	filename := "mail-latency." + timestamp + ".csv"
	outputFile := outputDir + "/" + filename

	// Write to file
	if err := os.WriteFile(outputFile, []byte(csvContent), 0644); err != nil {
		log.Error("failed to write output file", "error", err, "file", outputFile)
		return map[string]any{}, err
	}
	log.Debug("CSV written to file", "file", outputFile)

	log.Debug("mail-latency request completed successfully", "rt", rt)

	// Create response data
	res := map[string]any{
		"output_file": outputFile,
		"status":      0,
	}

	// Return in expected structure
	result := map[string]any{
		"req":    with,
		"res":    res,
		"rt":     rt.String(),
		"status": 0,
	}

	return result, nil
}
