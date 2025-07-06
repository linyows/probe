package probe

import (
	"io"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
)

func TestLoad(t *testing.T) {
	p := &Probe{
		FilePath: "./testdata/workflow.yml",
		Config: Config{
			Log:     os.Stdout,
			Verbose: true,
			RT:      false,
		},
	}
	err := p.Load()
	if err != nil {
		t.Errorf("probe load error %s", err)
	}
	expects, err := os.ReadFile("./testdata/marshaled-workflow.yml")
	if err != nil {
		t.Errorf("file read error %s", err)
	}
	got, err := yaml.Marshal(p.workflow)
	if string(got) != string(expects) {
		t.Errorf("\nExpected:\n%s\nGot:\n%s", expects, got)
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		verbose  bool
		expected *Probe
	}{
		{
			name:    "create new probe with verbose true",
			path:    "./test.yml",
			verbose: true,
			expected: &Probe{
				FilePath: "./test.yml",
				Config: Config{
					Log:     os.Stdout,
					Verbose: true,
					RT:      false,
				},
			},
		},
		{
			name:    "create new probe with verbose false",
			path:    "./another.yml",
			verbose: false,
			expected: &Probe{
				FilePath: "./another.yml",
				Config: Config{
					Log:     os.Stdout,
					Verbose: false,
					RT:      false,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := New(tt.path, tt.verbose)
			if got.FilePath != tt.expected.FilePath {
				t.Errorf("FilePath = %v, want %v", got.FilePath, tt.expected.FilePath)
			}
			if got.Config.Verbose != tt.expected.Config.Verbose {
				t.Errorf("Config.Verbose = %v, want %v", got.Config.Verbose, tt.expected.Config.Verbose)
			}
			if got.Config.RT != tt.expected.Config.RT {
				t.Errorf("Config.RT = %v, want %v", got.Config.RT, tt.expected.Config.RT)
			}
		})
	}
}

func TestIsYamlFile(t *testing.T) {
	p := &Probe{}
	tests := []struct {
		name     string
		filename string
		expected bool
	}{
		{"yml extension", "test.yml", true},
		{"yaml extension", "test.yaml", true},
		{"txt extension", "test.txt", false},
		{"no extension", "test", false},
		{"yml in middle", "test.yml.txt", false},
		{"yaml in middle", "test.yaml.txt", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.isYamlFile(tt.filename)
			if got != tt.expected {
				t.Errorf("isYamlFile(%s) = %v, want %v", tt.filename, got, tt.expected)
			}
		})
	}
}

func TestYamlFiles(t *testing.T) {
	p := &Probe{}
	tests := []struct {
		name     string
		filePath string
		wantErr  bool
	}{
		{
			name:     "single file",
			filePath: "./testdata/workflow.yml",
			wantErr:  false,
		},
		{
			name:     "nonexistent file",
			filePath: "./nonexistent.yml",
			wantErr:  true,
		},
		{
			name:     "directory",
			filePath: "./testdata",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p.FilePath = tt.filePath
			files, err := p.yamlFiles()
			if (err != nil) != tt.wantErr {
				t.Errorf("yamlFiles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(files) == 0 {
				t.Errorf("yamlFiles() returned empty files slice")
			}
		})
	}
}

func TestReadYamlFiles(t *testing.T) {
	p := &Probe{}

	tests := []struct {
		name    string
		paths   []string
		wantErr bool
	}{
		{
			name:    "existing file",
			paths:   []string{"./testdata/workflow.yml"},
			wantErr: false,
		},
		{
			name:    "nonexistent file",
			paths:   []string{"nonexistent.yml"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := p.readYamlFiles(tt.paths)
			if (err != nil) != tt.wantErr {
				t.Errorf("readYamlFiles() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSetDefaults(t *testing.T) {
	p := &Probe{}

	tests := []struct {
		name     string
		data     map[string]any
		defaults map[string]any
		expected map[string]any
	}{
		{
			name: "simple defaults",
			data: map[string]any{
				"existing": "value",
			},
			defaults: map[string]any{
				"new_key": "default_value",
			},
			expected: map[string]any{
				"existing": "value",
				"new_key":  "default_value",
			},
		},
		{
			name: "nested defaults",
			data: map[string]any{
				"nested": map[string]any{
					"existing": "value",
				},
			},
			defaults: map[string]any{
				"nested": map[string]any{
					"new_key": "default_value",
				},
			},
			expected: map[string]any{
				"nested": map[string]any{
					"existing": "value",
					"new_key":  "default_value",
				},
			},
		},
		{
			name: "overwrite protection",
			data: map[string]any{
				"key": "original",
			},
			defaults: map[string]any{
				"key": "default",
			},
			expected: map[string]any{
				"key": "original",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p.setDefaults(tt.data, tt.defaults)
			if !reflect.DeepEqual(tt.data, tt.expected) {
				t.Errorf("setDefaults() = %v, want %v", tt.data, tt.expected)
			}
		})
	}
}

func TestExitStatus(t *testing.T) {
	p := &Probe{}

	// Initialize workflow
	p.workflow = Workflow{exitStatus: 0}

	if got := p.ExitStatus(); got != 0 {
		t.Errorf("ExitStatus() = %v, want %v", got, 0)
	}

	// Set exit status to 1
	p.workflow.exitStatus = 1

	if got := p.ExitStatus(); got != 1 {
		t.Errorf("ExitStatus() = %v, want %v", got, 1)
	}
}

func TestLoadWithInvalidYaml(t *testing.T) {
	p := &Probe{
		FilePath: "./nonexistent.yml",
		Config: Config{
			Log:     io.Discard,
			Verbose: false,
			RT:      false,
		},
	}

	err := p.Load()
	if err == nil {
		t.Error("Load() should return error for invalid file path, but got nil")
	}
}

func TestYamlFilesWithMultiplePaths(t *testing.T) {
	p := &Probe{}

	// Test with comma-separated paths
	p.FilePath = "./testdata/workflow.yml,./testdata/marshaled-workflow.yml"
	files, err := p.yamlFiles()
	if err != nil {
		t.Errorf("yamlFiles() with multiple paths error = %v", err)
		return
	}

	if len(files) != 2 {
		t.Errorf("yamlFiles() with multiple paths returned %d files, want 2", len(files))
	}

	// Check that both files are included
	foundWorkflow := false
	foundMarshaled := false
	for _, file := range files {
		if strings.Contains(file, "workflow.yml") {
			foundWorkflow = true
		}
		if strings.Contains(file, "marshaled-workflow.yml") {
			foundMarshaled = true
		}
	}

	if !foundWorkflow || !foundMarshaled {
		t.Errorf("yamlFiles() did not find both expected files: %v", files)
	}
}

func TestDoWithInvalidPath(t *testing.T) {
	p := &Probe{
		FilePath: "./nonexistent.yml",
		Config: Config{
			Log:     io.Discard,
			Verbose: false,
			RT:      false,
		},
	}

	err := p.Do()
	if err == nil {
		t.Error("Do() should return error for invalid file path, but got nil")
	}
}
