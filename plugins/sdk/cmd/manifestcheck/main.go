package main

import (
	"fmt"
	"os"

	"github.com/open-wanderer/wanderer/plugins/sdk/manifestcheck"
)

func main() {
	path := "plugin.json"
	if len(os.Args) > 1 {
		path = os.Args[1]
	}
	if err := manifestcheck.PrintFile(os.Stdout, path); err != nil {
		fmt.Fprintf(os.Stderr, "manifestcheck: %v\n", err)
		os.Exit(1)
	}
}
