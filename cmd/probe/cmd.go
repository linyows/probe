package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/linyows/probe"
	"github.com/linyows/probe/actions/hello"
	http "github.com/linyows/probe/actions/http"
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
	case "http":
		http.Serve()
	case "hello":
		hello.Serve()
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

	flag.BoolVar(&c.Help, "help", false, "Show command usage")
	flag.BoolVar(&c.Help, "h", false, "Show command usage (shorthand)")
	flag.BoolVar(&c.Version, "version", false, "Show version information")
	//flag.BoolVar(&c.Init, "init", false, "Export a workflow template as yaml file")
	//flag.BoolVar(&c.Lint, "lint", false, "Check the syntax in workflow")
	flag.BoolVar(&c.RT, "rt", false, "Show response time")
	flag.BoolVar(&c.Verbose, "verbose", false, "Show verbose log")
	flag.BoolVar(&c.Verbose, "v", false, "Show verbose log (shorthand)")

	for _, arg := range args[1:] {
		if strings.HasPrefix(arg, "-") && !c.isValid(arg) {
			fmt.Fprintf(os.Stderr, "[ERROR] Unknown flag: %s\n", arg)
			fmt.Fprintf(os.Stderr, "[INFO] try --help to know more\n")
			return nil
		}
	}

	flag.Parse()

	// Set WorkflowPath from first non-flag argument
	if flag.NArg() > 0 {
		c.WorkflowPath = flag.Arg(0)
	}

	return &c
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
  workflow-file    Path to YAML workflow file

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
	flag.PrintDefaults()
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
