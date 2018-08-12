/*
Sniperkit-Bot
- Status: analyzed
*/

package cmd

import (
	"errors"
	"flag"
	"os/user"
	"runtime"

	"github.com/sniperkit/snk.fork.volt/logger"
)

var cmdMap = make(map[string]Cmd)

// Cmd represents volt's subcommand interface.
// All subcommands must implement this.
type Cmd interface {
	ProhibitRootExecution(args []string) bool
	Run(args []string) int
	FlagSet() *flag.FlagSet
}

// Run is invoked by main(), each argument means 'volt {subcmd} {args}'.
func Run(subCmd string, args []string) int {
	self, exists := cmdMap[subCmd]
	if !exists {
		logger.Error("Unknown command '" + subCmd + "'")
		return 3
	}
	if self.ProhibitRootExecution(args) {
		err := detectPriviledgedUser()
		if err != nil {
			logger.Error(err.Error())
			return 4
		}
	}
	return self.Run(args)
}

// On Windows, this function always returns nil.
// Because if even administrator user creates a file, the file can be
// overwritten by normal user.
// On Linux, if current user's uid == 0, returns non-nil error.
func detectPriviledgedUser() error {
	if runtime.GOOS == "windows" {
		return nil
	}
	u, err := user.Current()
	if err != nil {
		return errors.New("Cannot get current user: " + err.Error())
	}
	if u.Uid == "0" {
		return errors.New(
			"Cannot run this sub command with root priviledge. " +
				"Please run as normal user")
	}
	return nil
}
