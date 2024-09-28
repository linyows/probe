package probe

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

type CLI struct {
	WorkflowPath string
	Init         bool
	Lint         bool
	Help         bool
	validFlags   []string
	ver          string
	rev          string
}

func NewCLI(v, r string) *CLI {
	c := CLI{
		validFlags: []string{"help", "init", "lint", "workflow"},
		ver:        v,
		rev:        r,
	}

	flag.StringVar(&c.WorkflowPath, "workflow", "", "Specify yaml-path of workflow")
	flag.BoolVar(&c.Help, "help", false, "Show command usage")
	flag.BoolVar(&c.Init, "init", false, "Export a workflow template as yaml file")
	flag.BoolVar(&c.Lint, "lint", false, "Check the syntax in workflow")

	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "-") && !c.isValid(arg) {
			fmt.Printf("Unknown flag: %s\n", arg)
			fmt.Println("try --help to know more")
			os.Exit(0)
		}
	}

	flag.Parse()
	return &c
}

func (c *CLI) isValid(flag string) bool {
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

func (c *CLI) Usage() {
	h := `
Probe - scenario testing tool (ver: %s [%s])

Usage: probe [options] <command>
`
	h = strings.TrimPrefix(h, "\n")
	fmt.Fprint(flag.CommandLine.Output(), fmt.Sprintf(h, c.ver, c.rev))
}

func (c *CLI) Start() {
	switch {
	case c.Help:
		c.Usage()
	case c.Lint:
	case c.Init:
	default:
		name := "hello"
		args := []string{"w", "date"}
		with := map[string]string{
			"a": "aaa",
			"b": "bbb",
			"c": "ccc",
		}
		_, err := RunActions(name, args, with)
		if err != nil {
			fmt.Printf("error: %s\n", err)
		}
	}
}
