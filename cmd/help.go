/*
Sniperkit-Bot
- Status: analyzed
*/

package cmd

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/sniperkit/snk.fork.volt/logger"
)

// ErrShowedHelp is used in parsing argument function of subcommand when the
// subcommand showed help. Then caller can exit successfully and silently.
var ErrShowedHelp = errors.New("already showed help")

func init() {
	cmdMap["help"] = &helpCmd{}
}

type helpCmd struct{}

func (cmd *helpCmd) ProhibitRootExecution(args []string) bool { return false }

func (cmd *helpCmd) FlagSet() *flag.FlagSet {
	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	fs.Usage = func() {
		fmt.Print(
			" .----------------.  .----------------.  .----------------.  .----------------.\n" +
				"| .--------------. || .--------------. || .--------------. || .--------------. |\n" +
				"| | ____   ____  | || |     ____     | || |   _____      | || |  _________   | |\n" +
				"| ||_  _| |_  _| | || |   .'    `.   | || |  |_   _|     | || | |  _   _  |  | |\n" +
				"| |  \\ \\   / /   | || |  /  .--.  \\  | || |    | |       | || | |_/ | | \\_|  | |\n" +
				"| |   \\ \\ / /    | || |  | |    | |  | || |    | |   _   | || |     | |      | |\n" +
				"| |    \\ ' /     | || |  \\  `--'  /  | || |   _| |__/ |  | || |    _| |_     | |\n" +
				"| |     \\_/      | || |   `.____.'   | || |  |________|  | || |   |_____|    | |\n" +
				"| |              | || |              | || |              | || |              | |\n" +
				"| '--------------' || '--------------' || '--------------' || '--------------' |\n" +
				" '----------------'  '----------------'  '----------------'  '----------------'\n" +
				`
Usage
  volt COMMAND ARGS

Command
  get [-l] [-u] [{repository} ...]
    Install or upgrade given {repository} list, or add local {repository} list as plugins

  rm [-r] [-p] {repository} [{repository2} ...]
    Remove vim plugin from ~/.vim/pack/volt/opt/ directory

  list [-f {text/template string}]
    Vim plugin information extractor.
    Unless -f flag was given, this command shows vim plugins of **current profile** (not all installed plugins) by default.

  enable {repository} [{repository2} ...]
    This is shortcut of:
    volt profile add -current {repository} [{repository2} ...]

  disable {repository} [{repository2} ...]
    This is shortcut of:
    volt profile rm -current {repository} [{repository2} ...]

  profile set {name}
    Set profile name

  profile show {name}
    Show profile info

  profile list
    List all profiles

  profile new {name}
    Create new profile

  profile destroy {name}
    Delete profile

  profile rename {old} {new}
    Rename profile {old} to {new}

  profile add {name} {repository} [{repository2} ...]
    Add one or more repositories to profile

  profile rm {name} {repository} [{repository2} ...]
    Remove one or more repositories to profile

  build [-full]
    Build ~/.vim/pack/volt/ directory

  migrate {migration operation}
    Perform miscellaneous migration operations.
    See 'volt migrate -help' for all available operations

  self-upgrade [-check]
    Upgrade to the latest volt command, or if -check was given, it only checks the newer version is available

  version
    Show volt command version` + "\n\n")
		//cmd.helped = true
	}
	return fs
}

func (cmd *helpCmd) Run(args []string) int {
	if len(args) == 0 {
		cmd.FlagSet().Usage()
		return 0
	}
	if args[0] == "help" { // "volt help help"
		fmt.Println("E478: Don't panic!")
		return 0
	}

	fs, exists := cmdMap[args[0]]
	if !exists {
		logger.Errorf("Unknown command '%s'", args[0])
		return 1
	}
	args = append([]string{"-help"}, args[1:]...)
	fs.Run(args)
	return 0
}
