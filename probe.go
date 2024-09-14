package probe

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"

	"github.com/goccy/go-yaml"
	"gopkg.in/go-playground/validator.v9"
)

type Probe struct {
	FilePath string
	Log      io.Writer
	Verbose  bool
	workflow Workflow
}

type Step struct {
	Name string            `yaml:"name"`
	Use  string            `validate:"required"`
	With map[string]string `yaml:"with"`
}

type Repeat struct {
	Count    int `yaml:"count",validate:"required"`
	Interval int `yaml:"interval"`
}

type Job struct {
	Name   string `yaml:"name",validate:"required"`
	Steps  []Step `yaml:"steps",validate:"required"`
	Repeat Repeat `yaml:"repeat"`
}

type Workflow struct {
	Name string `yaml:"name",validate:"required"`
	Jobs []Job  `yaml:"jobs",validate:"required"`
}

func New() *Probe {
	return &Probe{
		FilePath: "./probe.yaml",
		Log:      os.Stdout,
		Verbose:  true,
	}
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
	return nil
}
