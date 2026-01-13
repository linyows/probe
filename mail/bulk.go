package mail

import (
	"fmt"
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

type DeliveryResult struct {
	Sent     int
	Failed   int
	Total    int
	Sessions int
	Error    string
}

func (b *Bulk) Deliver() {
	result := b.DeliverWithResult()
	if result.Failed > 0 {
		fmt.Printf("[INFO] Delivery complete. Success: %d, Failed: %d\n", result.Sent, result.Failed)
	}
}

func (b *Bulk) DeliverWithResult() DeliveryResult {
	type sendResult struct {
		count int
		err   error
	}

	var wg sync.WaitGroup
	resultCh := make(chan sendResult, b.Session)

	for i := 0; i < b.Session; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			count, err := b.Send()
			resultCh <- sendResult{count: count, err: err}
		}()
	}

	wg.Wait()
	close(resultCh)

	totalSent := 0
	sessionsSuccess := 0
	sessionsFailed := 0
	var lastError string
	for result := range resultCh {
		if result.err != nil {
			fmt.Printf("[ERROR] Send failed: %v\n", result.err)
			sessionsFailed++
			lastError = result.err.Error()
		} else {
			sessionsSuccess++
			totalSent += result.count
		}
	}

	return DeliveryResult{
		Sent:     totalSent,
		Failed:   sessionsFailed,
		Total:    totalSent,
		Sessions: sessionsSuccess + sessionsFailed,
		Error:    lastError,
	}
}

func (b *Bulk) Send() (int, error) {
	n := b.calcMessageNumEachSession()
	if n == 0 {
		return 0, nil
	}

	m := &Mail{
		Addr:             b.Addr,
		MailFrom:         b.From,
		RcptTo:           strings.Split(b.To, ","),
		Data:             b.makeData(),
		StartTLSDisabled: true,
		MessageCount:     n,
	}

	err := m.Send()
	if err != nil {
		return 0, err
	}
	return n, nil
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

// MakeData returns the mail data for external access
func (b *Bulk) MakeData() []byte {
	return b.makeData()
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
