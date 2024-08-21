package smtp

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

type Bulk struct {
	Addr       string
	From       string
	To         string
	Subject    string
	MyHostname string
	Session    int
	Message    int
	Length     int

	mu    sync.Mutex
	count int
}

func (b *Bulk) Deliver() {
	var wg sync.WaitGroup

	for i := 0; i < b.Session; i++ {
		wg.Add(1)
		go b.Send(&wg)
	}

	wg.Wait()
}

func (b *Bulk) Send(wg *sync.WaitGroup) error {
	defer wg.Done()

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
