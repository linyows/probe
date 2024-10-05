package probe

import (
	"os"
	"testing"

	"gopkg.in/yaml.v2"
)

func TestLoad(t *testing.T) {
	p := &Probe{
		FilePath: "./testdata/workflow.yml",
		Log:      os.Stdout,
		Verbose:  true,
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
