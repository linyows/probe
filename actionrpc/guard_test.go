package actionrpc

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
)

func TestGuard_Active(t *testing.T) {
	tests := []struct {
		guard Guard
		want  bool
	}{
		{Guard{}, false},
		{Guard{AllowActions: []string{"shell"}, Keeping: []string{"http"}}, false},
		{Guard{ReadOnly: true}, true},
		{Guard{AllowHosts: []string{"localhost"}}, true},
	}
	for _, tt := range tests {
		if got := tt.guard.Active(); got != tt.want {
			t.Errorf("%+v.Active() = %v, want %v", tt.guard, got, tt.want)
		}
	}
}

func TestGuard_Runs(t *testing.T) {
	g := Guard{ReadOnly: true, Keeping: []string{"http", "db"}, AllowActions: []string{"shell"}}
	for uses, want := range map[string]bool{
		"http":                 true,
		"db":                   true,
		"shell":                true,
		"ssh":                  false,
		"./actions/mine":       false,
		"owner/repo/action@v1": false,
	} {
		if got := g.Runs(uses); got != want {
			t.Errorf("Runs(%q) = %v, want %v", uses, got, want)
		}
	}
	// Without a guard every action runs.
	if !(Guard{}).Runs("ssh") {
		t.Error("an inactive guard should run every action")
	}
}

func TestGuard_AllowsHost(t *testing.T) {
	g := Guard{AllowHosts: []string{"api.example.com", "localhost:8080", "*.internal", "127.0.0.1", "[::1]:9000", "DB.Example.com"}}
	tests := map[string]bool{
		"api.example.com:443":    true, // a host without a port allows any port
		"api.example.com:8443":   true,
		"api.example.com":        true,
		"www.example.com:443":    false,
		"localhost:8080":         true,
		"localhost:8081":         false,
		"a.internal:5432":        true,
		"a.b.internal:5432":      true,
		"internal:5432":          false, // *.internal is for the names under it
		"evilinternal:5432":      false,
		"127.0.0.1:18091":        true,
		"[::1]:9000":             true,
		"[::1]:9001":             false,
		"db.example.com:3306":    true, // names are compared without case
		"API.EXAMPLE.COM:443":    true,
		"api.example.com.evil:1": false,
	}
	for host, want := range tests {
		if got := g.AllowsHost(host); got != want {
			t.Errorf("AllowsHost(%q) = %v, want %v", host, got, want)
		}
	}
	if !(Guard{ReadOnly: true}).AllowsHost("anywhere:1") {
		t.Error("a guard without hosts should allow any host")
	}
}

func TestGuard_CheckHost(t *testing.T) {
	g := Guard{AllowHosts: []string{"localhost"}}
	if err := g.CheckHost("localhost:80"); err != nil {
		t.Errorf("CheckHost(localhost:80) = %v, want nil", err)
	}
	err := g.CheckHost("example.com:443")
	if !IsRefused(err) {
		t.Fatalf("CheckHost(example.com:443) = %v, want a refusal", err)
	}
	if want := "refused: the host example.com:443 is not one the run allows (localhost)"; err.Error() != want {
		t.Errorf("error = %q, want %q", err, want)
	}
}

func TestIsRefused(t *testing.T) {
	if !IsRefused(fmt.Errorf("wrapped: %w", Refuse("no"))) {
		t.Error("a wrapped refusal should be one")
	}
	if IsRefused(errors.New("refused: looks like one")) {
		t.Error("an error that only says refused is not one")
	}
	if IsRefused(nil) {
		t.Error("nil is not a refusal")
	}
}

// guardedAction records the guard it was told, and refuses when told to.
type guardedAction struct {
	MockActions
	guard  Guard
	refuse bool
}

func (a *guardedAction) RunStep(call Call) (map[string]any, map[string]any, error) {
	a.guard = call.Guard
	if a.refuse {
		return nil, nil, fmt.Errorf("checking: %w", Refuse("POST may write"))
	}
	return map[string]any{"ok": true}, nil, nil
}

func TestClientRunStepTellsTheGuard(t *testing.T) {
	a := &guardedAction{}
	c := &Client{client: directClient{&Server{Impl: a}}}
	guard := Guard{ReadOnly: true, AllowHosts: []string{"localhost"}, AllowActions: []string{"shell"}, Keeping: []string{"http"}}
	if _, _, err := c.RunStep(Call{With: map[string]any{}, Guard: guard}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(a.guard, guard) {
		t.Errorf("the action was told %+v, want %+v", a.guard, guard)
	}

	if _, _, err := c.RunStep(Call{With: map[string]any{}}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.guard.Active() {
		t.Errorf("the action was told %+v, want no guard", a.guard)
	}
}

func TestClientRunStepCarriesARefusal(t *testing.T) {
	c := &Client{client: directClient{&Server{Impl: &guardedAction{refuse: true}}}}
	_, _, err := c.RunStep(Call{With: map[string]any{}, Guard: Guard{ReadOnly: true}})
	var r *Refused
	if !errors.As(err, &r) {
		t.Fatalf("err = %v (%T), want a refusal", err, err)
	}
	if r.Reason != "POST may write" {
		t.Errorf("Reason = %q, want the action's", r.Reason)
	}

	// Any other error stays one of the action.
	failing := &MockActions{RunFunc: func(map[string]any) (map[string]any, error) { return nil, errors.New("boom") }}
	c = &Client{client: directClient{&Server{Impl: failing}}}
	if _, _, err := c.RunStep(Call{With: map[string]any{}}); err == nil || IsRefused(err) {
		t.Errorf("err = %v, want an error that is not a refusal", err)
	}
}
