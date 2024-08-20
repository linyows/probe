package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/carlescere/scheduler"
	"github.com/linyows/probe/smtp"
)

var (
	version = "dev"
	commit  = ""
	date    = ""
	builtBy = ""
	m       = smtp.MockServer{}
	b       = smtp.Bulk{}
	verFlag bool
)

func init() {
	// server command options
	flag.StringVar(&m.Addr, "listen", "localhost:1025", "")
	flag.StringVar(&m.Name, "hostname", "probe", "")
	// bulk command options
	flag.StringVar(&b.Addr, "server", "localhost:1025", "")
	flag.StringVar(&b.From, "from", "alice@example.com", "")
	flag.StringVar(&b.To, "to", "bob@example.com", "")
	flag.StringVar(&b.MyHostname, "my-hostname", "localhost", "")
	flag.StringVar(&b.Subject, "subject", "Test", "")
	flag.IntVar(&b.Session, "session", 1, "")
	flag.IntVar(&b.Message, "message", 1, "")
	flag.IntVar(&b.Length, "length", 400, "")
	// global options
	flag.BoolVar(&verFlag, "version", false, "")

	flag.Parse()
}

func main() {
	hundle()
}

func usage() {
	header := `Probe - scenario testing tool

Usage: probe [options] <command>

Commands:
	start
  server
  bulk
`
	fmt.Fprint(flag.CommandLine.Output(), header)
}

func hundle() {
	args := flag.Args()
	narg := flag.NArg()

	if verFlag || narg == 0 {
		fmt.Fprintf(os.Stderr, buildInfo(version, commit, date, builtBy)+"\n")
		return
	}

	switch args[0] {
	case "server":
		if err := m.Serve(); err != nil {
			m.Log.Printf("Raised fatal error: %#v\n", err)
		}
	case "bulk":
		b.Deliver()

	case "start":
		// case 1: Send 3 emails every 10 seconds
		c1, _ := scheduler.Every(10).Seconds().Run(func() {
			b := smtp.Bulk{
				Addr:       "localhost:5871",
				From:       "alice@msa1.local",
				To:         "bob@mx1.local",
				MyHostname: "msa1-local",
				Subject:    "Experiment: Case 1",
				Session:    3,
				Message:    3,
				Length:     200,
			}
			b.Deliver()
		})
		// case 2: Send 30 emails every 30 seconds
		c2, _ := scheduler.Every(30).Seconds().Run(func() {
			b := smtp.Bulk{
				Addr:       "localhost:5872",
				From:       "carol@msa2.local",
				To:         "bob@mx2.local",
				MyHostname: "msa2-local",
				Subject:    "Experiment: Case 2",
				Session:    3,
				Message:    30,
				Length:     800,
			}
			b.Deliver()
		})
		// case 3: Send 1 email every 10 seconds
		c3, _ := scheduler.Every(10).Seconds().Run(func() {
			b := smtp.Bulk{
				Addr:       "localhost:5873",
				From:       "mallory@msa3.local",
				To:         "bob@mx3.local",
				MyHostname: "msa3-local",
				Subject:    "Experiment: Case 3",
				Session:    1,
				Message:    1,
				Length:     800,
			}
			b.Deliver()
		})

		time.Sleep(10 * time.Minute)
		c1.Quit <- true
		c2.Quit <- true
		c3.Quit <- true

	default:
		usage()
	}
}

func buildInfo(version, commit, date, builtBy string) string {
	var result = version
	if commit != "" {
		result = fmt.Sprintf("%s\ncommit: %s", result, commit)
	}
	if date != "" {
		result = fmt.Sprintf("%s\nbuilt at: %s", result, date)
	}
	if builtBy != "" {
		result = fmt.Sprintf("%s\nbuilt by: %s", result, builtBy)
	}
	return result
}
