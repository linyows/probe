package probe

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"

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
	y, err := ioutil.ReadFile(p.FilePath)
	if err != nil {
		return err
	}

	v := validator.New()
	dec := yaml.NewDecoder(bytes.NewReader(y), yaml.Validator(v))
	if err = dec.Decode(&p.workflow); err != nil {
		return err
	}

	p.setDefaultsToSteps()

	return nil
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
