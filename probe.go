package probe

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/goccy/go-yaml"
)

type Probe struct {
	FilePath string
	workflow Workflow
	Config   Config
}

type Config struct {
	Log     io.Writer
	Verbose bool
	RT      bool
}

func New(path string, v bool) *Probe {
	return &Probe{
		FilePath: path,
		Config: Config{
			Log:     os.Stdout,
			Verbose: v,
			RT:      false,
		},
	}
}

func (p *Probe) Do() error {
	if err := p.Load(); err != nil {
		return err
	}

	return p.workflow.Start(p.Config)
}

func (p *Probe) ExitStatus() int {
	return p.workflow.exitStatus
}

func (p *Probe) Load() error {
	files, err := p.yamlFiles()
	if err != nil {
		return NewFileError("load_yaml_files", "failed to locate YAML files", err).
			WithContext("workflow_path", p.FilePath)
	}
	y, err := p.readYamlFiles(files)
	if err != nil {
		return NewFileError("read_yaml_files", "failed to read YAML files", err).
			WithContext("files", files)
	}

	v := validator.New()
	dec := yaml.NewDecoder(bytes.NewReader([]byte(y)), yaml.Validator(v))
	if err = dec.Decode(&p.workflow); err != nil {
		return NewConfigurationError("decode_yaml", "failed to decode YAML workflow", err).
			WithContext("workflow_path", p.FilePath)
	}

	p.setDefaultsToSteps()

	return nil
}

func (p *Probe) readYamlFiles(paths []string) (string, error) {
	var y strings.Builder

	for i, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		y.Write(data)
		
		// Add newline between files to prevent concatenation issues
		if i < len(paths)-1 && len(data) > 0 && data[len(data)-1] != '\n' {
			y.WriteByte('\n')
		}
	}

	return y.String(), nil
}

// yamlFiles resolves paths and returns a list of YAML (.yml and .yaml) files
func (p *Probe) yamlFiles() ([]string, error) {
	var files []string
	paths := strings.Split(p.FilePath, ",")

	for _, path := range paths {
		path = strings.TrimSpace(path)

		// Check if it's a single file or a directory
		if info, err := os.Stat(path); err == nil {
			if info.IsDir() {
				// Read all YAML files from the directory
				rfiles, err := os.ReadDir(path)
				if err != nil {
					return nil, err
				}
				for _, rf := range rfiles {
					if !rf.IsDir() && p.isYamlFile(rf.Name()) {
						files = append(files, filepath.Join(path, rf.Name()))
					}
				}
			} else if p.isYamlFile(path) {
				// Single file path
				files = append(files, path)
			}
		} else {
			// Handle wildcard pattern
			matches, err := filepath.Glob(path)
			if err != nil {
				return nil, err
			}
			if len(matches) == 0 {
				return nil, fmt.Errorf("%s: no such file or directory", path)
			}
			for _, match := range matches {
				if p.isYamlFile(match) {
					files = append(files, match)
				}
			}
		}
	}

	return files, nil
}

// isYAMLFile checks if the filename has a .yml or .yaml extension
func (p *Probe) isYamlFile(f string) bool {
	return strings.HasSuffix(f, ".yml") || strings.HasSuffix(f, ".yaml")
}

func (p *Probe) setDefaultsToSteps() {
	for _, job := range p.workflow.Jobs {
		if job.Defaults == nil {
			continue
		}

		dataMap, ok := job.Defaults.(map[string]any)
		if !ok {
			continue
		}

		for key, values := range dataMap {
			defaults, defok := values.(map[string]any)
			if !defok {
				continue
			}

			for _, s := range job.Steps {
				if s.Uses != key {
					continue
				}
				p.setDefaults(s.With, defaults)
			}
		}
	}
}

func (p *Probe) setDefaults(data, defaults map[string]any) {
	for key, defaultValue := range defaults {
		// If key does not exist in data
		if _, exists := data[key]; !exists {
			data[key] = defaultValue
			continue
		}

		// If you have a nested map with a key of data
		if nestedDefault, ok := defaultValue.(map[string]any); ok {
			if nestedData, ok := data[key].(map[string]any); ok {
				// Recursively set default values
				p.setDefaults(nestedData, nestedDefault)
			}
		}
	}
}
