package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/fatih/color"
	"github.com/linyows/probe"
	"github.com/linyows/probe/actions"
	"github.com/linyows/probe/oas"
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
	WorkflowPath   string
	SubCommand     string
	SubCommandArgs []string
	Init           bool
	Lint           bool
	Help           bool
	Version        bool
	Verbose        bool
	Timing         bool
	DagMermaid     bool
	Output         string
	validFlags     []string
	ver            string
	rev            string
	outWriter      io.Writer
	errWriter      io.Writer
	mocking        bool
}

func newCmd() *Cmd {
	return &Cmd{
		validFlags: []string{"help", "h", "version", "timing", "verbose", "v", "mermaid", "output"},
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
	skipNext := false

	for i := range args {
		if skipNext {
			skipNext = false
			continue
		}

		arg := args[i]

		if strings.HasPrefix(arg, "-") {
			// Handle flags
			flagName := strings.TrimLeft(arg, "-")

			// Handle flags with "=" (e.g., --flag=value)
			flagValue := ""
			hasValue := false
			if idx := strings.Index(flagName, "="); idx != -1 {
				flagValue = flagName[idx+1:]
				hasValue = true
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
			case "timing":
				c.Timing = true
			case "verbose", "v":
				c.Verbose = true
			case "mermaid":
				c.DagMermaid = true
			case "output":
				// Accept both --output=stream and --output stream
				if !hasValue {
					if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
						return fmt.Errorf("flag needs an argument: %s", arg)
					}
					flagValue = args[i+1]
					skipNext = true
				}
				if _, err := probe.ParseOutputMode(flagValue); err != nil {
					return err
				}
				c.Output = flagValue
			}
		} else {
			// Non-flag arguments
			nonFlagArgs = append(nonFlagArgs, arg)
		}
	}

	// Check if first non-flag argument is a subcommand
	if len(nonFlagArgs) > 0 && isSubCommand(nonFlagArgs[0]) {
		c.SubCommand = nonFlagArgs[0]
		c.SubCommandArgs = nonFlagArgs[1:]
	} else if len(nonFlagArgs) > 0 {
		c.WorkflowPath = nonFlagArgs[0]
	}

	return nil
}

var subCommands = []string{"gen", "dag"}

func isSubCommand(name string) bool {
	return slices.Contains(subCommands, name)
}

func (c *Cmd) isValidFlag(flagName string) bool {
	return slices.Contains(c.validFlags, flagName)
}

func (c *Cmd) isValid(flag string) bool {
	if idx := strings.Index(flag, "="); idx != -1 {
		flag = flag[:idx]
	}

	return slices.Contains(c.validFlags, strings.TrimLeft(flag, "-"))
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
       probe gen <openapi-file>
       probe dag [--mermaid] <workflow-file>

Arguments:
  workflow-file    Path to YAML workflow file(s). Multiple files can be
                   specified with comma-separated paths (e.g., "base.yml,override.yml")
                   to merge configurations.

Subcommands:
  gen <file>       Generate probe workflow YAML from OpenAPI specification
  dag <file>       Show job dependency graph as ASCII art (default)
                   Use --mermaid to output in Mermaid format

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
		{"", "--timing", "Show timing (start time, response time)"},
		{"-v", "--verbose", "Show verbose log"},
		{"", "--output", "Report output: auto, spinner or stream (env: PROBE_OUTPUT)"},
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

	case c.SubCommand != "":
		return c.runSubCommand()

	case c.DagMermaid:
		_, _ = fmt.Fprintf(c.errWriter, "[ERROR] --mermaid can only be used with the dag subcommand\n")
		_, _ = fmt.Fprintf(c.errWriter, "Usage: probe dag [--mermaid] <workflow-file>\n")
		return 1

	case c.WorkflowPath == "":
		_, _ = fmt.Fprintf(c.errWriter, "[ERROR] workflow is required\n")
		return 1

	default:
		if !c.mocking {
			return c.runProbe()
		}
		return 0
	}
}

func (c *Cmd) runSubCommand() int {
	switch c.SubCommand {
	case "gen":
		return c.runGen()
	case "dag":
		return c.runDag()
	default:
		_, _ = fmt.Fprintf(c.errWriter, "[ERROR] unknown subcommand: %s\n", c.SubCommand)
		return 1
	}
}

func (c *Cmd) runGen() int {
	if len(c.SubCommandArgs) == 0 {
		_, _ = fmt.Fprintf(c.errWriter, "[ERROR] OpenAPI spec file is required\n")
		_, _ = fmt.Fprintf(c.errWriter, "Usage: probe gen <openapi-file>\n")
		return 1
	}

	output, err := oas.Generate(c.SubCommandArgs[0])
	if err != nil {
		_, _ = fmt.Fprintf(c.errWriter, "[ERROR] %v\n", err)
		return 1
	}

	_, _ = fmt.Fprint(c.outWriter, output)
	return 0
}

func (c *Cmd) runProbe() int {
	p := probe.New(c.WorkflowPath, c.Verbose)
	if c.Timing {
		p.Config.Timing = true
	}

	// The flag wins over PROBE_OUTPUT, which in turn wins over auto detection.
	outputMode := c.Output
	if outputMode == "" {
		outputMode = os.Getenv("PROBE_OUTPUT")
	}
	mode, err := probe.ParseOutputMode(outputMode)
	if err != nil {
		_, _ = fmt.Fprintf(c.errWriter, "[ERROR] %v\n", err)
		return 1
	}
	p.Config.Output = mode

	if err := p.Do(); err != nil {
		_, _ = fmt.Fprintf(c.errWriter, "[ERROR] %v\n", err)
		return 1
	}
	return p.ExitStatus()
}

func (c *Cmd) runDag() int {
	if len(c.SubCommandArgs) == 0 {
		_, _ = fmt.Fprintf(c.errWriter, "[ERROR] workflow file is required\n")
		_, _ = fmt.Fprintf(c.errWriter, "Usage: probe dag [--mermaid] <workflow-file>\n")
		return 1
	}

	if c.mocking {
		return 0
	}

	p := probe.New(c.SubCommandArgs[0], c.Verbose)
	var graph string
	var err error
	if c.DagMermaid {
		graph, err = p.DagMermaid()
	} else {
		graph, err = p.DagAscii()
	}
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
	serve, ok := actions.Lookup(name)
	if !ok {
		_, _ = fmt.Fprintf(c.errWriter, "[ERROR] not supported plugin: %s\n", name)
		return
	}

	if !c.mocking {
		serve()
	}
}
