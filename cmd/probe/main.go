package main

import (
	"flag"
	"fmt"
	"os"
	"sync"
	"time"

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
		case1 := smtp.Bulk{
			Addr:       "localhost:5871",
			From:       "alice@msa1.local",
			To:         "bob@mx1.local",
			MyHostname: "msa1-local",
			Subject:    "Experiment: Case 1",
			Session:    30,
			Message:    1000,
			Length:     800,
		}
		case2 := smtp.Bulk{
			Addr:       "localhost:5872",
			From:       "carol@msa2.local",
			To:         "bob@mx2.local",
			MyHostname: "msa2-local",
			Subject:    "Experiment: Case 2",
			Session:    30,
			Message:    10000,
			Length:     800,
		}
		case3 := smtp.Bulk{
			Addr:       "localhost:5873",
			From:       "mallory@msa3.local",
			To:         "bob@mx3.local",
			MyHostname: "msa3-local",
			Subject:    "Experiment: Case 3",
			Session:    30,
			Message:    1000,
			Length:     800,
		}

		sixTh := 6
		tenSec := 10
		tenMin := 10
		repeat := sixTh * tenMin

		fmt.Fprintf(os.Stdout, "Case1: total %d (%d messages, %d times every %d seconds)\n",
			case1.Message*repeat, case1.Message, repeat, tenSec)
		fmt.Fprintf(os.Stdout, "Case2: total %d (%d messages, %d times every %d seconds)\n",
			case2.Message*repeat, case2.Message, repeat, tenSec)
		fmt.Fprintf(os.Stdout, "Case3: total %d (%d messages, %d times every %d seconds)\n",
			case3.Message*repeat, case3.Message, repeat, tenSec)

		var wg sync.WaitGroup

		for i := 0; i < repeat; i++ {
			go func() {
				defer wg.Done()
				wg.Add(1)
				case1.Deliver()
			}()
			go func() {
				defer wg.Done()
				wg.Add(1)
				case2.Deliver()
			}()
			go func() {
				defer wg.Done()
				wg.Add(1)
				case3.Deliver()
			}()
			time.Sleep(time.Duration(tenSec) * time.Second)
		}

		wg.Wait()

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
