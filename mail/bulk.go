package mail

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/linyows/probe"
)

func NewBulk(p probe.ActionsParams) (*Bulk, error) {
	var b Bulk
	if err := probe.AssignStruct(p, &b); err != nil {
		return nil, err
	}
	return &b, nil
}

type Bulk struct {
	Addr       string `map:"addr" validate:"required"`
	From       string `map:"from" validate:"required"`
	To         string `map:"to" validate:"required"`
	Subject    string `map:"subject"`
	MyHostname string `map:"myhostname"`
	Session    int    `map:"session"`
	Message    int    `map:"message"`
	Length     int    `map:"length"`

	mu    sync.Mutex
	count int
}

func (b *Bulk) Deliver() {
	var wg sync.WaitGroup
	errCh := make(chan error, b.Session)

	for i := 0; i < b.Session; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := b.Send(); err != nil {
				errCh <- err
			} else {
				errCh <- nil
			}
		}()
	}

	wg.Wait()
	close(errCh)

	success := 0
	fail := 0
	for err := range errCh {
		if err != nil {
			log.Printf("Send failed: %v", err)
			fail++
		} else {
			success++
		}
	}
	if fail > 0 {
		log.Printf("Delivery complete. Success: %d, Failed: %d", success, fail)
	}
}

func (b *Bulk) Send() error {
	n := b.calcMessageNumEachSession()
	if n == 0 {
		return nil
	}

	m := &Mail{
		Addr:             b.Addr,
		MailFrom:         b.From,
		RcptTo:           strings.Split(b.To, ","),
		Data:             b.makeData(),
		StartTLSDisabled: true,
		MessageCount:     n,
	}

	return m.Send()
}

func (b *Bulk) calcMessageNumEachSession() int {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.Message == b.Session {
		return 1
	}

	// Round up messages per session
	n := (b.Message + b.Session - 1) / b.Session

	// If over message count
	k := b.Message - b.count
	if k < n {
		b.count = 0
		return k
	}

	// If not over
	b.count = b.count + n
	return n
}

func (b *Bulk) makeData() []byte {
	now := time.Now().Format("Mon, 02 Jan 2006 15:04:05 -0700")
	appendingText := insertLF(strings.Repeat("*", b.Length), 80)
	data := fmt.Sprintf(`From: %s
To: %s
Date: %s
Subject: %s

This is a test mail.

%s
`, b.From, b.To, now, b.Subject, appendingText)
	return []byte(data)
}

func insertLF(s string, length int) string {
	n := len(s)
	if n <= length {
		return s
	}

	var result string
	for i := 0; i < n; i += length {
		if i+length < len(s) {
			result += s[i:i+length] + "\n"
		} else {
			result += s[i:]
		}
	}

	return result
}
