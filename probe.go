package probe

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/goccy/go-yaml"
)

type Probe struct {
	FilePath string
	Log      io.Writer
	Verbose  bool
	workflow Workflow
}

func New(path string) *Probe {
	return &Probe{
		FilePath: path,
		Log:      os.Stdout,
		Verbose:  true,
	}
}

func (p *Probe) Do() error {
	if err := p.Load(); err != nil {
		return err
	}

	p.workflow.Start()

	return nil
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

	p.setDefaults()

	return nil
}

func (p *Probe) setDefaults() {
	for _, job := range p.workflow.Jobs {
		if job.Defaults == nil {
			continue
		}

		dataMap, ok := job.Defaults.(map[string]interface{})
		if !ok {
			continue
		}

		for key, values := range dataMap {
			defaults, defok := values.(map[string]interface{})
			if !defok {
				continue
			}

			for _, s := range job.Steps {
				if s.Uses != key {
					continue
				}
				for defk, defv := range defaults {
					_, exists := s.With[defk]
					if !exists {
						s.With[defk] = defv
					}
				}
			}
		}
	}
}

func getEnvMap() map[string]string {
	envmap := make(map[string]string)

	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			envmap[parts[0]] = parts[1]
		}
	}

	return envmap
}
