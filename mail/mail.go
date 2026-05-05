package mail

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"net/smtp"
	"os"
	"time"
)

type Mail struct {
	Addr             string
	MailFrom         string
	RcptTo           []string
	Data             []byte
	Auth             smtp.Auth
	StartTLSDisabled bool
	MessageCount     int
}

func (m *Mail) Send() error {
	if err := validateLine(m.MailFrom); err != nil {
		return err
	}
	for _, recp := range m.RcptTo {
		if err := validateLine(recp); err != nil {
			return fmt.Errorf("smtp rcptto validate error: %w", err)
		}
	}
	c, err := Dial(m.Addr)
	if err != nil {
		return fmt.Errorf("tcp dial error: %w", err)
	}
	defer func() { _ = c.Close() }()
	if err = c.hello(); err != nil {
		return fmt.Errorf("smtp hello error: %w", err)
	}
	if !m.StartTLSDisabled {
		if ok, _ := c.Extension("STARTTLS"); ok {
			config := &tls.Config{ServerName: c.serverName}
			if testHookStartTLS != nil {
				testHookStartTLS(config)
			}
			if err = c.StartTLS(config); err != nil {
				return fmt.Errorf("starttls error: %w", err)
			}
		}
	}
	if m.Auth != nil && c.ext != nil {
		if _, ok := c.ext["AUTH"]; !ok {
			return errors.New("smtp: server doesn't support AUTH")
		}
		if err = c.Auth(m.Auth); err != nil {
			return fmt.Errorf("smtp-auth error: %w", err)
		}
	}

	for i := 0; i < m.MessageCount; i++ {
		if err = c.Mail(m.MailFrom); err != nil {
			return fmt.Errorf("smtp mailfrom error: %w", err)
		}
		for _, addr := range m.RcptTo {
			if err = c.Rcpt(addr); err != nil {
				return fmt.Errorf("smtp rcptto error: %w", err)
			}
		}
		w, err := c.Data()
		if err != nil {
			return fmt.Errorf("smtp data error: %w", err)
		}

		_, err = w.Write(m.appendSendTimestamp(m.appendIDtoSubject(m.Data)))
		if err != nil {
			return fmt.Errorf("smtp data write error: %w", err)
		}
		err = w.Close()
		if err != nil {
			return fmt.Errorf("smtp data close error: %w", err)
		}
	}

	return c.Quit()
}

func getFQDN() string {
	if fqdn := os.Getenv("FQDN_DOMAIN"); fqdn != "" {
		return fqdn
	}
	hostname, err := os.Hostname()
	if err != nil {
		// RFC 6761 reserves .invalid for cases where a real domain is unavailable.
		return "localhost.invalid"
	}
	return hostname
}

// genMessageID returns the id-left component (ns + 128bit random) and the
// fully bracketed Message-ID. The id-left is reused as the Subject tracking
// suffix so each mail's Message-ID and Subject can be cross-referenced.
func genMessageID() (idLeft, full string) {
	ns := time.Now().UnixNano()
	uid, err := OptimisticUID()
	if err != nil {
		idLeft = fmt.Sprintf("%d", ns)
	} else {
		idLeft = fmt.Sprintf("%d.%s", ns, uid)
	}
	full = fmt.Sprintf("<%s@%s>", idLeft, getFQDN())
	return
}

// appendSendTimestamp prepends an X-Send-Timestamp-Ns header carrying the
// current time in Unix nanoseconds, captured at the moment the message is
// about to be written to the SMTP DATA stream.
func (m *Mail) appendSendTimestamp(data []byte) []byte {
	h := []byte(fmt.Sprintf("X-Send-Timestamp-Ns: %d\n", time.Now().UnixNano()))
	return append(h, data...)
}

func (m *Mail) appendIDtoSubject(data []byte) []byte {
	idLeft, msgID := genMessageID()
	lines := bytes.Split(data, []byte("\n"))

	for i, line := range lines {
		if bytes.HasPrefix(line, []byte("Subject:")) {
			lines[i] = append(line, []byte(" - "+idLeft)...)
			break
		}
	}

	mid := []byte("Message-ID: " + msgID)
	lines = append([][]byte{mid}, lines...)

	return bytes.Join(lines, []byte("\n"))
}
