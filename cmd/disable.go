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
	"github.com/sniperkit/snk.fork.volt/pathutil"
)

func init() {
	cmdMap["disable"] = &disableCmd{}
}

type disableCmd struct {
	helped bool
}

func (cmd *disableCmd) ProhibitRootExecution(args []string) bool { return true }

func (cmd *disableCmd) FlagSet() *flag.FlagSet {
	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	fs.Usage = func() {
		fmt.Print(`
Usage
  volt disable [-help] {repository} [{repository2} ...]

Quick example
  $ volt disable tyru/caw.vim # will disable tyru/caw.vim plugin in current profile

Description
  This is shortcut of:
  volt profile rm {current profile} {repository} [{repository2} ...]` + "\n\n")
		//fmt.Println("Options")
		//fs.PrintDefaults()
		fmt.Println()
		cmd.helped = true
	}
	return fs
}

func (cmd *disableCmd) Run(args []string) int {
	reposPathList, err := cmd.parseArgs(args)
	if err == ErrShowedHelp {
		return 0
	}
	if err != nil {
		logger.Error("Failed to parse args: " + err.Error())
		return 10
	}

	profCmd := profileCmd{}
	err = profCmd.doRm(append(
		[]string{"-current"},
		reposPathList.Strings()...,
	))
	if err != nil {
		logger.Error(err.Error())
		return 11
	}

	return 0
}

func (cmd *disableCmd) parseArgs(args []string) (pathutil.ReposPathList, error) {
	fs := cmd.FlagSet()
	fs.Parse(args)
	if cmd.helped {
		return nil, ErrShowedHelp
	}

	if len(fs.Args()) == 0 {
		fs.Usage()
		return nil, errors.New("repository was not given")
	}

	// Normalize repos path
	reposPathList := make(pathutil.ReposPathList, 0, len(fs.Args()))
	for _, arg := range fs.Args() {
		reposPath, err := pathutil.NormalizeRepos(arg)
		if err != nil {
			return nil, err
		}
		reposPathList = append(reposPathList, reposPath)
	}

	return reposPathList, nil
}
