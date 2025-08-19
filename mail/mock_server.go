package mail

import (
	"bufio"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
)

const (
	crlf      string = "\r\n"
	outgoing  string = ">"
	incomming string = "<"
	inserver  string = "-"
)

type TLS struct {
	CertPath string
	KeyPath  string
}

type MockServer struct {
	Addr string
	Name string
	Log  *log.Logger
	*TLS
}

type MockServerSession struct {
	id                string
	server            *MockServer
	reader            *bufio.Reader
	writer            *bufio.Writer
	nowDataInProgress bool
}

func (s *MockServer) Serve() error {
	if s.TLS == nil {
		s.TLS = &TLS{
			CertPath: "keys/cert.pem",
			KeyPath:  "keys/key.pem",
		}
	}
	if s.Log == nil {
		s.Log = log.New(os.Stderr, "", log.LstdFlags)
	}

	listener, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return err
	}
	defer func() { _ = listener.Close() }()

	s.Log.Printf("The mocking SMTP server is listening on %s\n", s.Addr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			s.Log.Println("Accept error:", err)
			continue
		}

		sess := &MockServerSession{server: s}
		go sess.handle(conn)
	}
}

func (s *MockServerSession) handle(conn net.Conn) {
	defer func() { _ = conn.Close() }()

	err := s.setOptimisticID()
	if err != nil {
		s.server.Log.Println("UID generating error:", err)
		return
	}

	s.reader = bufio.NewReader(conn)
	s.writer = bufio.NewWriter(conn)

	s.writeStringWithLog(fmt.Sprintf("220 %s ESMTP Server", s.server.Name))
	_ = s.writer.Flush()
	s.nowDataInProgress = false

	for {
		line, err := s.reader.ReadString('\n')
		if err != nil {
			if !errors.Is(err, io.EOF) {
				s.server.Log.Printf("%s %s conn ReadString error: %#v", s.id, inserver, err)
			}
			return
		}

		s.server.Log.Printf("%s %s %s", s.id, incomming, line)
		line = strings.TrimSpace(line)
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		commands := strings.Split(line, crlf)
		for _, cmd := range commands {
			s.handleCommand(cmd)
		}

		_ = s.writer.Flush()
	}
}

func (s *MockServerSession) handleCommand(cmd string) {
	first := ""
	second := ""
	parts := strings.Fields(strings.TrimSpace(cmd))
	if len(parts) == 0 {
		return
	}
	first = strings.ToUpper(parts[0])
	if len(parts) > 1 {
		second = strings.ToUpper(parts[1])
	}
	switch first {
	case "EHLO":
		str := `250-%s
250-PIPELINING
250-SIZE 10240000
250-STARTTLS
250 8BITMIME`
		s.writeStringWithLog(fmt.Sprintf(strings.ReplaceAll(str, "\n", crlf), s.server.Name))
	case "HELO":
		s.writeStringWithLog(fmt.Sprintf("250 Hello %s", parts[1]))
	case "MAIL":
		if strings.Contains(second, "FROM:") {
			s.writeStringWithLog("250 2.1.0 Ok")
		}
	case "RCPT":
		if strings.Contains(second, "TO:") {
			s.writeStringWithLog("250 2.1.5 Ok")
		}
	case "DATA":
		s.nowDataInProgress = true
		s.writeStringWithLog("354 End data with <CR><LF>.<CR><LF>")
	case "QUIT":
		s.writeStringWithLog("221 2.0.0 Bye")
		_ = s.writer.Flush()
		return
	case ".":
		s.nowDataInProgress = false
		s.writeStringWithLog("250 2.0.0 Ok: queued")
	case "RSET":
		s.writeStringWithLog("250 2.0.0 Ok")
	case "NOOP":
		s.writeStringWithLog("250 2.0.0 Ok")
	case "VRFY":
		s.writeStringWithLog("502 5.5.1 VRFY command is disabled")
	case "STARTTLS":
		s.writeStringWithLog("220 2.0.0 Ready to start TLS")
		_ = s.writer.Flush()
	default:
		if !s.nowDataInProgress {
			s.writeStringWithLog("500 Command not recognized")
		}
	}
}

func (s *MockServerSession) setOptimisticID() error {
	uid, err := OptimisticUID()
	if err != nil {
		return err
	}
	s.id = uid
	return nil
}

//nolint:unused // Reserved for future TLS support
func (s *MockServerSession) startTLS(conn net.Conn) {
	cert, err := tls.LoadX509KeyPair(s.server.CertPath, s.server.KeyPath)
	if err != nil {
		s.server.Log.Printf("%s %s Error loading server certificate: %#v", s.id, inserver, err)
		return
	}
	tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}}
	tlsConn := tls.Server(conn, tlsConfig)
	if err := tlsConn.Handshake(); err != nil {
		s.writeStringWithLog("550 5.0.0 Handshake error")
		return
	}
	s.reader = bufio.NewReader(tlsConn)
	s.writer = bufio.NewWriter(tlsConn)
}

func (s *MockServerSession) writeStringWithLog(str string) {
	_, err := s.writer.WriteString(str + crlf)
	if err != nil {
		s.server.Log.Printf("%s %s WriteString error: %#v", s.id, outgoing, err)
	}
	s.server.Log.Printf("%s %s %s", s.id, outgoing, strings.ReplaceAll(str, crlf, "\\r\\n"))
}
