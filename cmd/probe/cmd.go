package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/linyows/probe"
	"github.com/linyows/probe/actions/db"
	"github.com/linyows/probe/actions/hello"
	http "github.com/linyows/probe/actions/http"
	"github.com/linyows/probe/actions/shell"
	"github.com/linyows/probe/actions/smtp"
)

type Cmd struct {
	WorkflowPath string
	Init         bool
	Lint         bool
	Help         bool
	Version      bool
	Verbose      bool
	RT           bool
	validFlags   []string
	ver          string
	rev          string
}

func runBuiltinActions(name string) {
	switch name {
	case "db":
		db.Serve()
	case "http":
		http.Serve()
	case "hello":
		hello.Serve()
	case "shell":
		shell.Serve()
	case "smtp":
		smtp.Serve()
	}
}

func newCmd(args []string) *Cmd {
	if len(args) >= 3 && args[1] == probe.BuiltinCmd {
		runBuiltinActions(args[2])
		return nil
	}

	c := Cmd{
		validFlags: []string{"help", "h", "version", "rt", "verbose", "v"},
		ver:        version,
		rev:        commit,
	}

	// Parse arguments manually to allow options after arguments
	if err := c.parseArgs(args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] %v\n", err)
		fmt.Fprintf(os.Stderr, "[INFO] try --help to know more\n")
		return nil
	}

	return &c
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

	_, err := blue.Fprintln(flag.CommandLine.Output(), strings.TrimLeft(logo, "\n"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] %v\n", err)
	}
	_, err = grey.Fprintf(flag.CommandLine.Output(), strings.TrimLeft(desc, "\n"), c.ver, c.rev)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] %v\n", err)
	}
	_, err = fmt.Fprintln(flag.CommandLine.Output(), head)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] %v\n", err)
	}
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
	}

	for _, opt := range options {
		if opt.short != "" {
			_, err := fmt.Fprintf(flag.CommandLine.Output(), "  %s, %-12s %s\n", opt.short, opt.long, opt.description)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[ERROR] %v\n", err)
			}
		} else {
			_, err := fmt.Fprintf(flag.CommandLine.Output(), "      %-12s %s\n", opt.long, opt.description)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[ERROR] %v\n", err)
			}
		}
	}
}

func (c *Cmd) start() int {
	switch {
	case c.Help:
		c.usage()
	case c.Version:
		c.printVersion()
	//case c.Lint:
	//case c.Init:
	case c.WorkflowPath == "":
		fmt.Fprintf(os.Stderr, "[ERROR] workflow is required\n")
		return 1
	default:
		p := probe.New(c.WorkflowPath, c.Verbose)
		if c.RT {
			p.Config.RT = true
		}
		if err := p.Do(); err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] %v\n", err)
		} else {
			return p.ExitStatus()
		}
	}

	return 1
}

func (c *Cmd) printVersion() {
	fmt.Printf("Probe Version %s (commit: %s)\n", c.ver, c.rev)
}
