package maillatency

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteNewFile(t *testing.T) {
	// Runs in the same second get their own files instead of overwriting the
	// first one.
	dir := t.TempDir()
	var paths []string
	for i, data := range []string{"first", "second", "third"} {
		path, err := writeNewFile(dir, "20261003-120000", []byte(data))
		if err != nil {
			t.Fatalf("writeNewFile() #%d error: %v", i+1, err)
		}
		paths = append(paths, path)
	}

	want := []string{
		"mail-latency.20261003-120000.csv",
		"mail-latency.20261003-120000-2.csv",
		"mail-latency.20261003-120000-3.csv",
	}
	for i, path := range paths {
		if filepath.Base(path) != want[i] {
			t.Errorf("file #%d = %s, want %s", i+1, filepath.Base(path), want[i])
		}
	}
	got, err := os.ReadFile(paths[0])
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "first" {
		t.Errorf("first file = %q, want it left as it was", got)
	}
}
