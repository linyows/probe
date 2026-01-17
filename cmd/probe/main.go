package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/linyows/probe"
	"github.com/linyows/probe/actions/browser"
	"github.com/linyows/probe/actions/db"
	"github.com/linyows/probe/actions/embedded"
	grpcaction "github.com/linyows/probe/actions/grpc"
	"github.com/linyows/probe/actions/hello"
	http "github.com/linyows/probe/actions/http"
	imapaction "github.com/linyows/probe/actions/imap"
	maillatencyaction "github.com/linyows/probe/actions/mail-latency"
	"github.com/linyows/probe/actions/shell"
	"github.com/linyows/probe/actions/smtp"
	sshaction "github.com/linyows/probe/actions/ssh"
)

var (
	version = "dev"
	commit  = "unknown"
)

func main() {
	c := newCmd()
	if c != nil {
		os.Exit(c.start(os.Args))
	}
}

type Cmd struct {
	WorkflowPath string
	Init         bool
	Lint         bool
	Help         bool
	Version      bool
	Verbose      bool
	RT           bool
	DagAscii     bool
	validFlags   []string
	ver          string
	rev          string
	outWriter    io.Writer
	errWriter    io.Writer
	mocking      bool
}

func newCmd() *Cmd {
	return &Cmd{
		validFlags: []string{"help", "h", "version", "rt", "verbose", "v", "dag-ascii"},
		ver:        version,
		rev:        commit,
		outWriter:  os.Stdout,
		errWriter:  os.Stderr,
		mocking:    false,
	}
}

func newBufferCmd() *Cmd {
	c := newCmd()
	c.outWriter = new(bytes.Buffer)
	c.errWriter = new(bytes.Buffer)
	c.mocking = true
	return c
}

// parseArgs parses command line arguments manually to allow options after arguments
func (c *Cmd) parseArgs(args []string) error {
	var nonFlagArgs []string

	for i := 0; i < len(args); i++ {
		arg := args[i]

		if strings.HasPrefix(arg, "-") {
			// Handle flags
			flagName := strings.TrimLeft(arg, "-")

			// Handle flags with "=" (e.g., --flag=value)
			if idx := strings.Index(flagName, "="); idx != -1 {
				flagName = flagName[:idx]
			}

			if !c.isValidFlag(flagName) {
				return fmt.Errorf("unknown flag: %s", arg)
			}

			// Set the appropriate flag
			switch flagName {
			case "help", "h":
				c.Help = true
			case "version":
				c.Version = true
			case "rt":
				c.RT = true
			case "verbose", "v":
				c.Verbose = true
			case "dag-ascii":
				c.DagAscii = true
			}
		} else {
			// Non-flag arguments
			nonFlagArgs = append(nonFlagArgs, arg)
		}
	}

	// Set WorkflowPath from first non-flag argument
	if len(nonFlagArgs) > 0 {
		c.WorkflowPath = nonFlagArgs[0]
	}

	return nil
}

func (c *Cmd) isValidFlag(flagName string) bool {
	for _, validFlag := range c.validFlags {
		if flagName == validFlag {
			return true
		}
	}
	return false
}

func (c *Cmd) isValid(flag string) bool {
	if idx := strings.Index(flag, "="); idx != -1 {
		flag = flag[:idx]
	}

	for _, validFlag := range c.validFlags {
		if strings.TrimLeft(flag, "-") == validFlag {
			return true
		}
	}

	return false
}

func (c *Cmd) usage() {
	logo := `
 __  __  __  __  __
|  ||  ||  ||  || _|
|  ||  /| |||  /|  |
| | |  \| |||  \| _|
|_| |_\_|__||__||__|
`

	desc := `
Probe - A YAML-based workflow automation tool.
https://github.com/linyows/probe (ver: %s, rev: %s)
`

	head := `
Usage: probe [options] <workflow-file>

Arguments:
  workflow-file    Path to YAML workflow file(s). Multiple files can be 
                   specified with comma-separated paths (e.g., "base.yml,override.yml")
                   to merge configurations.

Options:`

	blue := color.New(color.FgBlue)
	grey := color.New(color.FgHiBlack)

	_, _ = blue.Fprintln(c.errWriter, strings.TrimLeft(logo, "\n"))
	_, _ = grey.Fprintf(c.errWriter, strings.TrimLeft(desc, "\n"), c.ver, c.rev)
	_, _ = fmt.Fprintln(c.errWriter, head)
	c.printOptions()
}

func (c *Cmd) printOptions() {
	options := []struct {
		short, long, description string
	}{
		{"-h", "--help", "Show command usage"},
		{"", "--version", "Show version information"},
		{"", "--rt", "Show response time"},
		{"-v", "--verbose", "Show verbose log"},
		{"", "--dag-ascii", "Show job dependency graph as ASCII art"},
	}

	for _, opt := range options {
		if opt.short != "" {
			_, _ = fmt.Fprintf(c.errWriter, "  %s, %-12s %s\n", opt.short, opt.long, opt.description)
		} else {
			_, _ = fmt.Fprintf(c.errWriter, "      %-12s %s\n", opt.long, opt.description)
		}
	}
}

func (c *Cmd) start(args []string) int {
	if len(args) >= 3 && args[1] == probe.BuiltinCmd {
		c.runBuiltinActions(args[2])
		return 0
	}

	// Parse arguments manually to allow options after arguments
	if err := c.parseArgs(args[1:]); err != nil {
		_, _ = fmt.Fprintf(c.errWriter, "[ERROR] %v\ntry --help to know more\n", err)
		return 1
	}

	switch {
	case c.Help:
		c.usage()
		return 1

	case c.Version:
		c.printVersion()
		return 0

	case c.WorkflowPath == "":
		_, _ = fmt.Fprintf(c.errWriter, "[ERROR] workflow is required\n")
		return 1

	case c.DagAscii:
		if !c.mocking {
			return c.runDagAscii()
		}
		return 0

	default:
		if !c.mocking {
			return c.runProbe()
		}
		return 0
	}
}

func (c *Cmd) runProbe() int {
	p := probe.New(c.WorkflowPath, c.Verbose)
	if c.RT {
		p.Config.RT = true
	}

	if err := p.Do(); err != nil {
		_, _ = fmt.Fprintf(c.errWriter, "[ERROR] %v\n", err)
		return 1
	}
	return p.ExitStatus()
}

func (c *Cmd) runDagAscii() int {
	p := probe.New(c.WorkflowPath, c.Verbose)
	graph, err := p.DagAscii()
	if err != nil {
		_, _ = fmt.Fprintf(c.errWriter, "[ERROR] %v\n", err)
		return 1
	}
	_, _ = fmt.Fprint(c.outWriter, graph)
	return 0
}

func (c *Cmd) printVersion() {
	_, _ = fmt.Fprintf(c.outWriter, "Probe Version %s (commit: %s)\n", c.ver, c.rev)
}

func (c *Cmd) runBuiltinActions(name string) {
	switch name {
	case "hello":
		if !c.mocking {
			hello.Serve()
		}
	case "http":
		if !c.mocking {
			http.Serve()
		}
	case "smtp":
		if !c.mocking {
			smtp.Serve()
		}
	case "db":
		if !c.mocking {
			db.Serve()
		}
	case "shell":
		if !c.mocking {
			shell.Serve()
		}
	case "browser":
		if !c.mocking {
			browser.Serve()
		}
	case "embedded":
		if !c.mocking {
			embedded.Serve()
		}
	case "grpc":
		if !c.mocking {
			grpcaction.Serve()
		}
	case "ssh":
		if !c.mocking {
			sshaction.Serve()
		}
	case "imap":
		if !c.mocking {
			imapaction.Serve()
		}
	case "mail-latency":
		if !c.mocking {
			maillatencyaction.Serve()
		}

	default:
		_, _ = fmt.Fprintf(c.errWriter, "[ERROR] not supported plugin: %s\n", name)
	}
}
