/*
Sniperkit-Bot
- Status: analyzed
*/

// +build go1.9

package main

import (
	"os"

	"github.com/sniperkit/snk.fork.volt/cmd"
	"github.com/sniperkit/snk.fork.volt/logger"
)

func main() {
	os.Exit(doMain())
}

func doMain() int {
	if os.Getenv("VOLT_DEBUG") != "" {
		logger.SetLevel(logger.DebugLevel)
	}
	if len(os.Args) <= 1 {
		os.Args = append(os.Args, "help")
	}
	return cmd.Run(os.Args[1], os.Args[2:])
}
