package smtp

import (
	"bufio"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

func LatencyCMD() {
	maildir := flag.String("maildir", "", "Path to the Maildir directory")
	csv := flag.Bool("csv", false, "Output csv")
	flag.Parse()

	if *maildir == "" {
		log.Fatal("Maildir path is required")
	}

	var records [][]string
	latencies := []time.Duration{}

	earliestTime, err := findEarliestSendTime(maildir)
	if err != nil {
		return nil, errors.Wrap(err, "Error finding earliest send time")
	}

	jst, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		return nil, errors.Wrap(err, "Error load location")
	}
	datetimeF := "2006-01-02 15:04:05"

	if err = filepath.WalkDir(maildir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return errors.Wrap(err, "")
		}
		if d.IsDir() {
			return nil
		}

		returnPath, sendTime, subReceivedTime, receivedTime, latency, subLatency, err := getMailLatency(path)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Error processing file: %s", path))
		}

		relativeTime := sendTime.Sub(earliestTime).Seconds()
		relativeReceivedTime := receivedTime.Sub(earliestTime).Seconds()

		records = append(records, []string{
			fmt.Sprintf("%.0f", relativeTime),
			fmt.Sprintf("%.0f", relativeReceivedTime),
			fmt.Sprintf("%.0f", latency.Seconds()),
			fmt.Sprintf("%.0f", subLatency.Seconds()),
			sendTime.In(jst).Format(datetimeF),
			subReceivedTime.In(jst).Format(datetimeF),
			receivedTime.In(jst).Format(datetimeF),
			filepath.Base(path),
			returnPath,
		})
		latencies = append(latencies, latency)
		//records = append(records, []string{sendTime.Format(time.RFC3339), fmt.Sprintf("%d", latency.Seconds())})

		return nil
	}); err != nil {
		return nil, err
	}

	if csv {
		sort.Slice(records, func(i, j int) bool {
			num1, _ := strconv.Atoi(records[i][0])
			num2, _ := strconv.Atoi(records[j][0])
			return num1 < num2
		})
		return buildCSV(records), nil
	} else {
		//createHistogramByBucket(latencies, 10)
		createHistogramByDuration(latencies, 10*time.Second)
	}

	return nil, nil
}

func buildCSV(records [][]string) []string {
	data := []string{fmt.Sprintf("Time (s), Received Time(s), Latency (s), Sub Latency (s), Sent at, First Received at, Received at, File, Return-Path")}
	for _, r := range records {
		data = append(data, fmt.Sprintf("%s, %s, %s, %s, %s, %s, %s, %s, %s", r[0], r[1], r[2], r[3], r[4], r[5], r[6], r[7], r[8]))
	}

	return data
}

func findEarliestSendTime(maildir string) (time.Time, error) {
	var earliestTime time.Time

	err := filepath.WalkDir(maildir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		sendTime, err := getDateHeader(path)
		if err != nil {
			log.Println("Error processing file:", path, err)
			return nil
		}

		if earliestTime.IsZero() || sendTime.Before(earliestTime) {
			earliestTime = sendTime
		}

		return nil
	})

	return earliestTime, err
}

func getDateHeader(filePath string) (time.Time, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return time.Time{}, err
	}
	defer file.Close()

	var dateHeader time.Time
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "Date:") && dateHeader.IsZero() {
			dateHeader, err = parseDateHeader(line)
			if err != nil {
				return time.Time{}, err
			}
		}

		if !dateHeader.IsZero() {
			break
		}
	}

	if dateHeader.IsZero() {
		return time.Time{}, fmt.Errorf("missing required headers")
	}

	return dateHeader, nil
}

func getMailLatency(filePath string) (string, time.Time, time.Time, time.Time, time.Duration, time.Duration, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", time.Time{}, time.Time{}, time.Time{}, 0, 0, err
	}
	defer file.Close()

	var dateHeader time.Time
	var returnPathHeader string
	var receivedHeaders []time.Time

	scanner := bufio.NewScanner(file)
	var receivedLines []string

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "Return-Path:") && returnPathHeader == "" {
			returnPathHeader = parseReturnPathHeader(line)
		}

		if strings.HasPrefix(line, "Date:") && dateHeader.IsZero() {
			dateHeader, err = parseDateHeader(line)
			if err != nil {
				return "", time.Time{}, time.Time{}, time.Time{}, 0, 0, err
			}
		}

		if strings.HasPrefix(line, "Received:") {
			receivedLines = append(receivedLines, strings.TrimSpace(line))
			//} else if len(receivedLines) > 0 && receivedHeader.IsZero() && strings.HasPrefix(line, "\t") {
		} else if len(receivedLines) > 0 && (strings.HasPrefix(line, "\t") || strings.HasPrefix(line, " ")) {
			receivedLines = append(receivedLines, strings.TrimSpace(line))
			if strings.Contains(line, ";") {
				rH, err := parseReceivedHeader(strings.Join(receivedLines, " "))
				if err != nil {
					return "", time.Time{}, time.Time{}, time.Time{}, 0, 0, err
				}
				receivedHeaders = append(receivedHeaders, rH)
				receivedLines = nil
			}
		}

		//if !dateHeader.IsZero() && !receivedHeader.IsZero() {
		//      break
		//}
	}

	if dateHeader.IsZero() || len(receivedHeaders) == 0 {
		return "", time.Time{}, time.Time{}, time.Time{}, 0, 0, fmt.Errorf("missing required headers")
	}

	lastR := receivedHeaders[0]
	firstR := receivedHeaders[len(receivedHeaders)-1]
	latency := lastR.Sub(dateHeader)
	subLatency := lastR.Sub(firstR)
	return returnPathHeader, dateHeader, firstR, lastR, latency, subLatency, nil
}

func parseReturnPathHeader(header string) string {
	str := strings.TrimPrefix(header, "Return-Path: ")
	return strings.Trim(strings.TrimSpace(str), "<>")
}

func parseDateHeader(header string) (time.Time, error) {
	dateStr := strings.TrimPrefix(header, "Date: ")
	return time.Parse(time.RFC1123Z, dateStr)
}

func parseReceivedHeader(header string) (time.Time, error) {
	parts := strings.Split(header, ";")
	if len(parts) < 2 {
		return time.Time{}, fmt.Errorf("malformed Received header")
	}
	t := strings.TrimSpace(strings.Split(parts[len(parts)-1], "(")[0])
	return time.Parse(time.RFC1123Z, t)
}

func createHistogramByBucket(latencies []time.Duration, bucketCount int) {
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

	fmt.Println("Latency Histogram")
	for i, count := range buckets {
		fmt.Printf("%d-%d s: %d\n", i*int(interval.Seconds()), (i+1)*int(interval.Seconds()), count)
	}
}

func createHistogramByDuration(latencies []time.Duration, bucketSize time.Duration) {
	buckets := make(map[int]int)

	for _, latency := range latencies {
		bucket := int(latency / bucketSize)
		buckets[bucket]++
	}

	fmt.Println("Latency Histogram")
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
