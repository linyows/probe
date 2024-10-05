package mail

import (
	"bytes"
	"crypto/tls"
	"errors"
	"net/smtp"
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
			return err
		}
	}
	c, err := Dial(m.Addr)
	if err != nil {
		return err
	}
	defer c.Close()
	if err = c.hello(); err != nil {
		return err
	}
	if !m.StartTLSDisabled {
		if ok, _ := c.Extension("STARTTLS"); ok {
			config := &tls.Config{ServerName: c.serverName}
			if testHookStartTLS != nil {
				testHookStartTLS(config)
			}
			if err = c.StartTLS(config); err != nil {
				return err
			}
		}
	}
	if m.Auth != nil && c.ext != nil {
		if _, ok := c.ext["AUTH"]; !ok {
			return errors.New("smtp: server doesn't support AUTH")
		}
		if err = c.Auth(m.Auth); err != nil {
			return err
		}
	}

	for i := 0; i < m.MessageCount; i++ {
		if err = c.Mail(m.MailFrom); err != nil {
			return err
		}
		for _, addr := range m.RcptTo {
			if err = c.Rcpt(addr); err != nil {
				return err
			}
		}
		w, err := c.Data()
		if err != nil {
			return err
		}

		_, err = w.Write(m.appendIDtoSubject(m.Data))
		if err != nil {
			return err
		}
		err = w.Close()
		if err != nil {
			return err
		}
	}

	return c.Quit()
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

	return bytes.Join(lines, []byte("\n"))
}
