package maillatency

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/linyows/probe/actionrpc"
)

type Action struct {
	log hclog.Logger
}

func (a *Action) Run(with map[string]any) (map[string]any, error) {
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

	a.log.Debug("received mail-latency request", "mail_dir", mailDir, "output_dir", outputDir)

	// Create a buffer to capture CSV output
	var csvBuffer bytes.Buffer

	// Measure execution time
	start := time.Now()
	// Get latencies and write to buffer
	if err := GetLatencies(mailDir, &csvBuffer); err != nil {
		a.log.Error("mail-latency request failed", "error", err)
		return map[string]any{}, err
	}
	rt := time.Since(start)

	csvContent := csvBuffer.String()

	outputFile, err := writeNewFile(outputDir, time.Now().Format("20060102-150405"), []byte(csvContent))
	if err != nil {
		a.log.Error("failed to write output file", "error", err, "dir", outputDir)
		return map[string]any{}, err
	}
	a.log.Debug("CSV written to file", "file", outputFile)

	a.log.Debug("mail-latency request completed successfully", "rt", rt)

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

// writeNewFile writes data to mail-latency.<timestamp>.csv in dir. The name
// only goes down to the second, so a run in the same second as another gets
// a numbered name instead of overwriting the earlier file.
func writeNewFile(dir, timestamp string, data []byte) (string, error) {
	for n := 1; n <= 100; n++ {
		name := "mail-latency." + timestamp + ".csv"
		if n > 1 {
			name = fmt.Sprintf("mail-latency.%s-%d.csv", timestamp, n)
		}
		path := filepath.Join(dir, name)
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		if _, err := f.Write(data); err != nil {
			_ = f.Close()
			return "", err
		}
		return path, f.Close()
	}
	return "", fmt.Errorf("no free file name for mail-latency.%s.csv in %s", timestamp, dir)
}

func Serve() {
	actionrpc.Serve(func(log hclog.Logger) actionrpc.Action {
		return &Action{log: log}
	})
}

// Params returns the keys the mail-latency action takes in with.
func Params() []string {
	return []string{"mail_dir", "output_dir"}
}
