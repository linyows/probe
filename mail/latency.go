package mail

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Latency struct {
	ElapsedTimeToSent     time.Duration
	ElapsedTimeToReceived time.Duration
	SentTime              time.Time
	LastReceivedTime      time.Time
	EndToEnd              time.Duration
	FirstReceivedTime     time.Time
	Relay                 time.Duration
	ReturnPath            string
	FilePath              string
}

type Latencies struct {
	EarliestTime time.Time
	Data         []Latency
	MailDir      string
	TimeFormat   string
	TimeLocation *time.Location
}

const (
	defaultTimezone   = "Asia/Tokyo"
	defaultTimeformat = "2006-01-02 15:04:05"
)

func GetLatencies(p string, w io.Writer) error {
	l := Latencies{MailDir: p}
	var err error

	if err = l.Make(); err != nil {
		return err
	}

	if err = l.writeCSVWithHeader(w); err != nil {
		return err
	}

	return nil
}

func (l *Latencies) FindEarliestSentTime() (time.Time, error) {
	var earliestTime time.Time

	fn := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		sentTime, err := l.getDateHeader(path)
		if err != nil {
			return err
		}
		if earliestTime.IsZero() || sentTime.Before(earliestTime) {
			earliestTime = sentTime
		}
		return nil
	}

	err := filepath.WalkDir(l.MailDir, fn)
	return earliestTime, err
}

func (l *Latencies) getDateHeader(p string) (time.Time, error) {
	f, err := os.Open(p)
	if err != nil {
		return time.Time{}, err
	}
	defer func() { _ = f.Close() }()

	var date time.Time
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()
		if HasFlexedPrefix(line, "Date:") && date.IsZero() {
			date, err = l.getSentTimeWithParse(line)
			if err != nil {
				return time.Time{}, err
			}
		}
		if !date.IsZero() {
			break
		}
	}
	if date.IsZero() {
		return time.Time{}, fmt.Errorf("missing required date headers")
	}
	return date, nil
}

func (l *Latencies) getReturnPathWithParse(s string) string {
	return strings.Trim(strings.TrimSpace(strings.TrimPrefix(s, "Return-Path: ")), "<>")
}

func (l *Latencies) getSentTimeWithParse(s string) (time.Time, error) {
	return time.Parse(time.RFC1123Z, strings.TrimPrefix(s, "Date: "))
}

func (l *Latencies) getReceivedTimeWithParse(s string) (time.Time, error) {
	parts := strings.Split(s, ";")
	if len(parts) < 2 {
		return time.Time{}, fmt.Errorf("malformed Received header")
	}
	return time.Parse(time.RFC1123Z, strings.TrimSpace(strings.Split(parts[len(parts)-1], "(")[0]))
}

func (l *Latencies) ParseMail(p string) error {
	f, err := os.Open(p)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	row := Latency{FilePath: p}
	var receivedTimes []time.Time
	var receivedLines []string
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()

		if HasFlexedPrefix(line, "Return-Path:") && row.ReturnPath == "" {
			row.ReturnPath = l.getReturnPathWithParse(line)
		}

		if HasFlexedPrefix(line, "Date:") && row.SentTime.IsZero() {
			row.SentTime, err = l.getSentTimeWithParse(line)
			if err != nil {
				return err
			}
		}

		if HasFlexedPrefix(line, "Received:") {
			receivedLines = append(receivedLines, strings.TrimSpace(line))
		} else if len(receivedLines) > 0 && (strings.HasPrefix(line, "\t") || strings.HasPrefix(line, " ")) {
			receivedLines = append(receivedLines, strings.TrimSpace(line))
			if strings.Contains(line, ";") {
				rt, err := l.getReceivedTimeWithParse(strings.Join(receivedLines, " "))
				if err != nil {
					return err
				}
				receivedTimes = append(receivedTimes, rt)
				receivedLines = nil
			}
		}
	}

	if row.SentTime.IsZero() || len(receivedTimes) == 0 {
		return fmt.Errorf("missing required headers")
	}

	row.LastReceivedTime = receivedTimes[0]
	row.FirstReceivedTime = receivedTimes[len(receivedTimes)-1]
	row.EndToEnd = row.LastReceivedTime.Sub(row.SentTime)
	row.Relay = row.LastReceivedTime.Sub(row.FirstReceivedTime)
	row.ElapsedTimeToSent = row.SentTime.Sub(l.EarliestTime)
	row.ElapsedTimeToReceived = row.LastReceivedTime.Sub(l.EarliestTime)
	l.Data = append(l.Data, row)

	return nil
}

func (l *Latencies) writeCSVWithHeader(w io.Writer) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	header := []string{
		"Elapsed Time (sec) - To Sent Time",
		"Elapsed Time (sec) - To Received Time",
		"Sent Time",
		"Last Received Time",
		"End-to-End Latency (sec)",
		"First Received Time",
		"Relay Latency (sec)",
		"Return Path",
		"File Path",
	}
	if err := writer.Write(header); err != nil {
		return err
	}

	for _, r := range l.Data {
		record := []string{
			fmt.Sprintf("%.0f", r.ElapsedTimeToSent.Seconds()),
			fmt.Sprintf("%.0f", r.ElapsedTimeToReceived.Seconds()),
			r.SentTime.In(l.TimeLocation).Format(l.TimeFormat),
			r.LastReceivedTime.In(l.TimeLocation).Format(l.TimeFormat),
			fmt.Sprintf("%.0f", r.EndToEnd.Seconds()),
			r.FirstReceivedTime.In(l.TimeLocation).Format(l.TimeFormat),
			fmt.Sprintf("%.0f", r.Relay.Seconds()),
			r.ReturnPath,
			filepath.Base(r.FilePath),
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}
	return nil
}

func (l *Latencies) Make() error {
	var err error
	if l.TimeLocation == nil {
		z, err := time.LoadLocation(defaultTimezone)
		if err != nil {
			return err
		}
		l.TimeLocation = z
	}
	if l.TimeFormat == "" {
		l.TimeFormat = defaultTimeformat
	}
	l.EarliestTime, err = l.FindEarliestSentTime()
	if err != nil {
		return err
	}

	fn := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && IsMailText(path) {
			if err = l.ParseMail(path); err != nil {
				return err
			}
		}
		return nil
	}

	if err = filepath.WalkDir(l.MailDir, fn); err != nil {
		return err
	}

	sort.Slice(l.Data, func(i, j int) bool {
		return l.Data[i].ElapsedTimeToSent < l.Data[j].ElapsedTimeToSent
	})

	return nil
}

func ReadFirstBytes(p string, bytes int) ([]byte, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	buf := make([]byte, bytes)
	n, err := io.ReadFull(f, buf)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return nil, err
	}
	return buf[:n], nil
}

func IsMailText(p string) bool {
	b, err := ReadFirstBytes(p, 20)
	if err != nil {
		return false
	}
	t := string(b)
	return HasFlexedPrefix(t, "Return-Path:") || HasFlexedPrefix(t, "Delivered-To:")
}

// HasPrefix with Case-Insensitive
func HasFlexedPrefix(s, prefix string) bool {
	if len(s) < len(prefix) {
		return false
	}
	lower := strings.ToLower(prefix)
	str := strings.ToLower(s[0:len(lower)])
	return str == lower
}

func CreateHistogramByBucket(latencies []time.Duration, bucketCount int) {
	buckets := make([]int, bucketCount)

	minLatency := latencies[0]
	maxLatency := latencies[0]

	for _, latency := range latencies {
		if latency < minLatency {
			minLatency = latency
		}
		if latency > maxLatency {
			maxLatency = latency
		}
	}

	interval := (maxLatency - minLatency) / time.Duration(bucketCount)

	for _, latency := range latencies {
		bucket := int((latency - minLatency) / interval)
		if bucket >= bucketCount {
			bucket = bucketCount - 1
		}
		buckets[bucket]++
	}

	for i, count := range buckets {
		fmt.Printf("%d-%d s: %d\n", i*int(interval.Seconds()), (i+1)*int(interval.Seconds()), count)
	}
}

func CreateHistogramByDuration(latencies []time.Duration, bucketSize time.Duration) {
	buckets := make(map[int]int)

	for _, latency := range latencies {
		bucket := int(latency / bucketSize)
		buckets[bucket]++
	}

	for i := 0; i <= maxBucket(buckets); i++ {
		fmt.Printf("%d-%d s: %d\n", i*int(bucketSize.Seconds()), (i+1)*int(bucketSize.Seconds()), buckets[i])
	}
}

func maxBucket(buckets map[int]int) int {
	max := 0
	for k := range buckets {
		if k > max {
			max = k
		}
	}
	return max
}
