package mail

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"net/smtp"
	"os"
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
	defer c.Close()
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

		_, err = w.Write(m.appendIDtoSubject(m.Data))
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
		return "localhost"
	}
	return hostname
}

func genMsgID(id string) string {
	return fmt.Sprintf("<%s@%s>", id, getFQDN())
}

func (m *Mail) appendIDtoSubject(data []byte) []byte {
	id := HashID()
	lines := bytes.Split(data, []byte("\n"))

	for i, line := range lines {
		if bytes.HasPrefix(line, []byte("Subject:")) {
			lines[i] = append(line, []byte(" - "+id)...)
			break
		}
	}

	mid := []byte("Message-ID: " + genMsgID(id))
	lines = append([][]byte{mid}, lines...)

	return bytes.Join(lines, []byte("\n"))
}
