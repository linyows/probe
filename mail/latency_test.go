package mail

import (
	"bytes"
	"io/ioutil"
	"testing"
)

func TestGetLatency(t *testing.T) {
	buf := new(bytes.Buffer)
	if err := GetLatencies("./testdata/mail/", buf); err != nil {
		t.Errorf("got error %s", err)
	}

	bytes, err := ioutil.ReadFile("./testdata/latency.csv")
	if err != nil {
		t.Errorf("csv read error %s", err)
	}

	got := buf.String()
	expects := string(bytes)
	if got != expects {
		t.Errorf("\nExpected:\n%s\nGot:\n%s", expects, got)
	}
}
