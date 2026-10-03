package mail

import (
	"bufio"
	"errors"
	"net"
	"strings"
	"sync"
	"testing"
)

// greetingRecorder is an SMTP server that accepts mail and remembers the
// EHLO and HELO lines it was sent, which the shared mock server does not.
type greetingRecorder struct {
	ln    net.Listener
	mu    sync.Mutex
	lines []string
}

func newGreetingRecorder(t *testing.T) *greetingRecorder {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	r := &greetingRecorder{ln: ln}
	t.Cleanup(func() { _ = ln.Close() })
	go r.serve()
	return r
}

func (r *greetingRecorder) addr() string { return r.ln.Addr().String() }

func (r *greetingRecorder) greetings() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.lines...)
}

func (r *greetingRecorder) serve() {
	for {
		conn, err := r.ln.Accept()
		if err != nil {
			return
		}
		go r.session(conn)
	}
}

func (r *greetingRecorder) session(conn net.Conn) {
	defer func() { _ = conn.Close() }()
	rd := bufio.NewReader(conn)
	write := func(s string) { _, _ = conn.Write([]byte(s + "\r\n")) }

	write("220 recorder ESMTP")
	inData := false
	for {
		line, err := rd.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		if inData {
			if line == "." {
				inData = false
				write("250 2.0.0 Ok: queued")
			}
			continue
		}
		cmd := strings.ToUpper(strings.SplitN(line, " ", 2)[0])
		switch cmd {
		case "EHLO", "HELO":
			r.mu.Lock()
			r.lines = append(r.lines, line)
			r.mu.Unlock()
			if cmd == "EHLO" {
				write("250-recorder")
				write("250 8BITMIME")
			} else {
				write("250 recorder")
			}
		case "MAIL", "RCPT", "RSET", "NOOP":
			write("250 Ok")
		case "DATA":
			inData = true
			write("354 End data with <CR><LF>.<CR><LF>")
		case "QUIT":
			write("221 Bye")
			return
		default:
			write("502 Command not implemented")
		}
	}
}

func stubHostname(t *testing.T, name string, err error) {
	t.Helper()
	orig := osHostname
	osHostname = func() (string, error) { return name, err }
	t.Cleanup(func() { osHostname = orig })
}

func TestMailSend_EHLOName(t *testing.T) {
	tests := []struct {
		name      string
		localName string
		host      string
		hostErr   error
		want      string
	}{
		{"given name", "probe-client.local", "machine-a", nil, "EHLO probe-client.local"},
		{"machine host name when none is given", "", "machine-a", nil, "EHLO machine-a"},
		{"localhost when the host name is unknown", "", "", errors.New("no hostname"), "EHLO localhost"},
		{"localhost when the host name is empty", "", "", nil, "EHLO localhost"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stubHostname(t, tt.host, tt.hostErr)
			srv := newGreetingRecorder(t)

			m := &Mail{
				Addr:             srv.addr(),
				LocalName:        tt.localName,
				MailFrom:         "from@example.com",
				RcptTo:           []string{"to@example.com"},
				Data:             []byte("Subject: hi\r\n\r\nbody\r\n"),
				StartTLSDisabled: true,
				MessageCount:     1,
			}
			if err := m.Send(); err != nil {
				t.Fatalf("Send: %v", err)
			}

			got := srv.greetings()
			if len(got) != 1 || got[0] != tt.want {
				t.Errorf("greetings = %q, want [%q]", got, tt.want)
			}
		})
	}
}

func TestMailSend_RejectsLineBreakInName(t *testing.T) {
	srv := newGreetingRecorder(t)
	m := &Mail{
		Addr:             srv.addr(),
		LocalName:        "evil.example\r\nRSET",
		MailFrom:         "from@example.com",
		RcptTo:           []string{"to@example.com"},
		Data:             []byte("x\r\n"),
		StartTLSDisabled: true,
		MessageCount:     1,
	}
	if err := m.Send(); err == nil {
		t.Error("a name with a line break must be refused, not sent as a second command")
	}
	if got := srv.greetings(); len(got) != 0 {
		t.Errorf("nothing should have been sent, got %q", got)
	}
}

// TestSend_MyHostname follows the myhostname parameter from the action's
// input through Bulk to the EHLO line, the path that dropped it before.
func TestSend_MyHostname(t *testing.T) {
	stubHostname(t, "machine-a", nil)
	srv := newGreetingRecorder(t)

	_, err := Send(map[string]any{
		"addr":       srv.addr(),
		"from":       "from@example.com",
		"to":         "to@example.com",
		"subject":    "hi",
		"myhostname": "probe-client.local",
		"session":    2,
		"message":    2,
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	got := srv.greetings()
	if len(got) != 2 {
		t.Fatalf("greetings = %q, want one per session", got)
	}
	for _, g := range got {
		if g != "EHLO probe-client.local" {
			t.Errorf("greeting = %q, want EHLO probe-client.local", g)
		}
	}
}
