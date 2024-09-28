package main

import (
	"fmt"
	"os"

	"github.com/linyows/probe"
	"github.com/linyows/probe/actions/bulkmail"
	"github.com/linyows/probe/actions/hello"
)

var (
	version = "dev"
	commit  = "unknown"
)

func main() {
	if len(os.Args) >= 3 && os.Args[1] == probe.BuiltinCmd {
		runBuiltinActions(os.Args[2])
		return
	}

	cli := probe.NewCLI(version, commit)
	cli.Start()
}

func runBuiltinActions(name string) {
	switch name {
	case "hello":
		hello.Serve()
	case "bulkmail":
		bulkmail.Serve()
	default:
		fmt.Printf("builtin-actions not found: %s", name)
	}
}
