package ssh

import (
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

// echo writes the command to stdout and "err" to stderr, and exits with 0.
func echo(s session) (uint32, bool) {
	_, _ = fmt.Fprint(s.stdout, s.cmd)
	_, _ = fmt.Fprint(s.stderr, "err")
	return 0, true
}

func TestReqDo(t *testing.T) {
	srv := newTestServer(t, echo)

	var before, after bool
	r := srv.req("hostname")
	r.cb = &Callback{
		before: func(host string, port int, user, cmd string) {
			before = host == srv.host() && port == srv.port() && user == testUser && cmd == "hostname"
		},
		after: func(*Result) { after = true },
	}

	res, err := r.Do()
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	if res.Res.Code != 0 || res.Status != 0 {
		t.Errorf("Code = %d, Status = %d, want both 0", res.Res.Code, res.Status)
	}
	if res.Res.Stdout != "hostname" || res.Res.Stderr != "err" {
		t.Errorf("Stdout = %q, Stderr = %q, want them apart", res.Res.Stdout, res.Res.Stderr)
	}
	if res.RT <= 0 {
		t.Errorf("RT = %v, want it measured", res.RT)
	}
	if !before || !after {
		t.Errorf("callbacks: before %v, after %v", before, after)
	}
}

// A command that exits with a non-zero status is a result, not an error.
func TestReqDoExitStatus(t *testing.T) {
	srv := newTestServer(t, func(session) (uint32, bool) { return 3, true })

	res, err := srv.req("false").Do()
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	if res.Res.Code != 3 || res.Status != 1 {
		t.Errorf("Code = %d, Status = %d, want 3 and 1", res.Res.Code, res.Status)
	}
}

func TestReqDoWorkdir(t *testing.T) {
	srv := newTestServer(t, echo)

	r := srv.req("ls")
	r.Workdir = "/srv/app"
	if _, err := r.Do(); err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	commands, _, _ := srv.received()
	if len(commands) != 1 || commands[0] != "cd /srv/app && ls" {
		t.Errorf("server ran %q, want the command run in workdir", commands)
	}
}

// A variable the server refuses is reported, and the command runs without it.
func TestReqDoEnv(t *testing.T) {
	srv := newTestServer(t, echo)

	var refused []string
	r := srv.req("env")
	r.Env = map[string]string{"PROBE_STAGE": "e2e", "LANG": "C"}
	r.cb = &Callback{envRefused: func(name string, _ error) { refused = append(refused, name) }}

	res, err := r.Do()
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	if res.Res.Code != 0 {
		t.Errorf("Code = %d, want the command run", res.Res.Code)
	}
	_, _, envs := srv.received()
	if len(envs) != 1 || envs[0]["PROBE_STAGE"] != "e2e" {
		t.Errorf("server got env %v, want PROBE_STAGE set", envs)
	}
	if len(refused) != 1 || refused[0] != "LANG" {
		t.Errorf("refused = %v, want LANG", refused)
	}
}

// A command that outlives the timeout is sent SIGTERM, and what it reports
// once it stops is the result.
func TestReqDoTimeout(t *testing.T) {
	srv := newTestServer(t, func(s session) (uint32, bool) {
		_, _ = fmt.Fprint(s.stdout, "started")
		select {
		case sig := <-s.signals:
			if sig == string(ssh.SIGTERM) {
				return 143, true
			}
			return 1, true
		case <-time.After(10 * time.Second):
			return 0, true
		}
	})

	r := srv.req("sleep 60")
	r.Timeout = "200ms"
	start := time.Now()
	res, err := r.Do()
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("Do() took %v, want it to stop the command at the timeout", elapsed)
	}
	if res.Res.Code != 143 || res.Status != 1 {
		t.Errorf("Code = %d, Status = %d, want 143 and 1", res.Res.Code, res.Status)
	}
	if res.Res.Stdout != "started" {
		t.Errorf("Stdout = %q, want the output from before the timeout", res.Res.Stdout)
	}
	_, signals, _ := srv.received()
	if len(signals) == 0 || signals[0] != string(ssh.SIGTERM) {
		t.Errorf("server got signals %v, want TERM first", signals)
	}
}

// A session that ends without an exit status, as when the connection drops,
// is an error: there is no result to test.
func TestReqDoNoExitStatus(t *testing.T) {
	srv := newTestServer(t, func(session) (uint32, bool) { return 0, false })

	_, err := srv.req("true").Do()
	if err == nil || !strings.Contains(err.Error(), "SSH command execution failed") {
		t.Fatalf("Do() error = %v, want the command reported as failed", err)
	}
}

func TestReqDoAuth(t *testing.T) {
	srv := newTestServer(t, echo)
	// Encrypting a key is slow, so the two cases that need one share it.
	encrypted := srv.writeClientKey(t, "phrase")

	tests := []struct {
		name    string
		setup   func(t *testing.T, r *Req)
		wantErr string
	}{
		{name: "password", setup: func(*testing.T, *Req) {}},
		{
			name:    "wrong password",
			setup:   func(_ *testing.T, r *Req) { r.Password = "nope" },
			wantErr: "failed to connect to SSH server",
		},
		{
			name: "key file",
			setup: func(t *testing.T, r *Req) {
				r.Password = ""
				r.KeyFile = srv.writeClientKey(t, "")
			},
		},
		{
			name: "key file with a passphrase",
			setup: func(t *testing.T, r *Req) {
				r.Password = ""
				r.KeyFile = encrypted
				r.KeyPassphrase = "phrase"
			},
		},
		{
			name: "key file with a wrong passphrase",
			setup: func(t *testing.T, r *Req) {
				r.Password = ""
				r.KeyFile = encrypted
				r.KeyPassphrase = "other"
			},
			wantErr: "failed to parse private key",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := srv.req("whoami")
			tt.setup(t, r)
			res, err := r.Do()
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("Do() error = %v, want one containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Do() error = %v", err)
			}
			if res.Res.Code != 0 {
				t.Errorf("Code = %d, want 0", res.Res.Code)
			}
		})
	}
}

func TestReqDoHostKey(t *testing.T) {
	srv := newTestServer(t, echo)

	otherPub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	other, err := ssh.NewPublicKey(otherPub)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("a known host", func(t *testing.T) {
		r := srv.req("true")
		r.StrictHostCheck = true
		r.KnownHosts = srv.writeKnownHosts(t, srv.hostKey)
		if _, err := r.Do(); err != nil {
			t.Fatalf("Do() error = %v", err)
		}
	})

	t.Run("a host whose key changed", func(t *testing.T) {
		ran, _, _ := srv.received()
		r := srv.req("true")
		r.StrictHostCheck = true
		r.KnownHosts = srv.writeKnownHosts(t, other)
		_, err := r.Do()
		if err == nil || !strings.Contains(err.Error(), "key mismatch") {
			t.Fatalf("Do() error = %v, want a host key mismatch", err)
		}
		if commands, _, _ := srv.received(); len(commands) != len(ran) {
			t.Errorf("server ran %q for a client that did not trust it", commands[len(ran):])
		}
	})
}

func TestExecute(t *testing.T) {
	srv := newTestServer(t, echo)

	data := map[string]any{
		"host":              srv.host(),
		"port":              fmt.Sprint(srv.port()),
		"user":              testUser,
		"password":          testPassword,
		"cmd":               "uptime",
		"strict_host_check": "false",
		"env":               map[string]any{"PROBE_STAGE": "e2e"},
	}
	got, err := Execute(data)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	res, _ := got["res"].(map[string]any)
	if res["code"] != 0 || res["stdout"] != "uptime" {
		t.Errorf("Execute() res = %v, want the command's output", got["res"])
	}
	if _, ok := data["port"].(string); !ok {
		t.Error("Execute() changed the caller's data")
	}
}
