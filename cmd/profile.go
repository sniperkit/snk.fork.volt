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

	"github.com/hashicorp/go-multierror"

	"github.com/sniperkit/snk.fork.volt/cmd/builder"
	"github.com/sniperkit/snk.fork.volt/lockjson"
	"github.com/sniperkit/snk.fork.volt/logger"
	"github.com/sniperkit/snk.fork.volt/pathutil"
	"github.com/sniperkit/snk.fork.volt/transaction"
)

type profileCmd struct {
	helped bool
}

var profileSubCmd = make(map[string]func([]string) error)

func init() {
	cmdMap["profile"] = &profileCmd{}
}

func (cmd *profileCmd) ProhibitRootExecution(args []string) bool {
	if len(args) == 0 {
		return true
	}
	subCmd := args[0]
	switch subCmd {
	case "show":
		return false
	case "list":
		return false
	default:
		return true
	}
}

func (cmd *profileCmd) FlagSet() *flag.FlagSet {
	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	fs.Usage = func() {
		fmt.Print(`
Usage
  profile [-help] {command}

Command
  profile set [-n] {name}
    Set profile name to {name}.

  profile show [-current | {name}]
    Show profile info of {name}.

  profile list
    List all profiles.

  profile new {name}
    Create new profile of {name}. This command does not switch to profile {name}.

  profile destroy {name}
    Delete profile of {name}.
    NOTE: Cannot delete current profile.

  profile rename {old} {new}
    Rename profile {old} to {new}.

  profile add [-current | {name}] {repository} [{repository2} ...]
    Add one or more repositories to profile {name}.

  profile rm [-current | {name}] {repository} [{repository2} ...]
    Remove one or more repositories from profile {name}.

Quick example
  $ volt profile list   # default profile is "default"
  * default
  $ volt profile new foo   # will create profile "foo"
  $ volt profile list
  * default
    foo
  $ volt profile set foo   # will switch profile to "foo"
  $ volt profile list
    default
  * foo

  $ volt profile set default   # on profile "default"

  $ volt enable tyru/caw.vim    # enable loading tyru/caw.vim on current profile
  $ volt profile add foo tyru/caw.vim    # enable loading tyru/caw.vim on "foo" profile

  $ volt disable tyru/caw.vim   # disable loading tyru/caw.vim on current profile
  $ volt profile rm foo tyru/caw.vim    # disable loading tyru/caw.vim on "foo" profile

  $ volt profile destroy foo   # will delete profile "foo"` + "\n\n")
		cmd.helped = true
	}
	return fs
}

func (cmd *profileCmd) Run(args []string) int {
	// Parse args
	args, err := cmd.parseArgs(args)
	if err == ErrShowedHelp {
		return 0
	}
	if err != nil {
		logger.Error(err.Error())
		return 10
	}

	subCmd := args[0]
	switch subCmd {
	case "set":
		err = cmd.doSet(args[1:])
	case "show":
		err = cmd.doShow(args[1:])
	case "list":
		err = cmd.doList(args[1:])
	case "new":
		err = cmd.doNew(args[1:])
	case "destroy":
		err = cmd.doDestroy(args[1:])
	case "rename":
		err = cmd.doRename(args[1:])
	case "add":
		err = cmd.doAdd(args[1:])
	case "rm":
		err = cmd.doRm(args[1:])
	default:
		logger.Error("unknown subcommand: " + subCmd)
		return 11
	}

	if err != nil {
		logger.Error(err.Error())
		return 20
	}

	return 0
}

func (cmd *profileCmd) parseArgs(args []string) ([]string, error) {
	fs := cmd.FlagSet()
	fs.Parse(args)
	if cmd.helped {
		return nil, ErrShowedHelp
	}
	if len(fs.Args()) == 0 {
		fs.Usage()
		logger.Error("must specify subcommand")
		return nil, ErrShowedHelp
	}
	return fs.Args(), nil
}

func (*profileCmd) getCurrentProfile() (string, error) {
	lockJSON, err := lockjson.Read()
	if err != nil {
		return "", errors.New("failed to read lock.json: " + err.Error())
	}
	return lockJSON.CurrentProfileName, nil
}

func (cmd *profileCmd) doSet(args []string) error {
	// Parse args
	createProfile := false
	if len(args) > 0 && args[0] == "-n" {
		createProfile = true
		args = args[1:]
	}
	if len(args) == 0 {
		cmd.FlagSet().Usage()
		logger.Error("'volt profile set' receives profile name.")
		return nil
	}
	profileName := args[0]

	// Read lock.json
	lockJSON, err := lockjson.Read()
	if err != nil {
		return errors.New("failed to read lock.json: " + err.Error())
	}

	// Exit if current profile is same as profileName
	if lockJSON.CurrentProfileName == profileName {
		return fmt.Errorf("'%s' is current profile", profileName)
	}

	// Create given profile unless the profile exists
	if _, err = lockJSON.Profiles.FindByName(profileName); err != nil {
		if !createProfile {
			return err
		}
		if err = cmd.doNew([]string{profileName}); err != nil {
			return err
		}
		// Read lock.json again
		lockJSON, err = lockjson.Read()
		if err != nil {
			return errors.New("failed to read lock.json: " + err.Error())
		}
		if _, err = lockJSON.Profiles.FindByName(profileName); err != nil {
			return err
		}
	}

	// Begin transaction
	err = transaction.Create()
	if err != nil {
		return err
	}
	defer transaction.Remove()

	// Set profile name
	lockJSON.CurrentProfileName = profileName

	// Write to lock.json
	err = lockJSON.Write()
	if err != nil {
		return err
	}

	logger.Info("Changed current profile: " + profileName)

	// Build ~/.vim/pack/volt dir
	err = builder.Build(false)
	if err != nil {
		return errors.New("could not build " + pathutil.VimVoltDir() + ": " + err.Error())
	}

	return nil
}

func (cmd *profileCmd) doShow(args []string) error {
	if len(args) == 0 {
		cmd.FlagSet().Usage()
		logger.Error("'volt profile show' receives profile name.")
		return nil
	}

	// Read lock.json
	lockJSON, err := lockjson.Read()
	if err != nil {
		return errors.New("failed to read lock.json: " + err.Error())
	}

	var profileName string
	if args[0] == "-current" {
		profileName = lockJSON.CurrentProfileName
	} else {
		profileName = args[0]
		if lockJSON.Profiles.FindIndexByName(profileName) == -1 {
			return fmt.Errorf("profile '%s' does not exist", profileName)
		}
	}

	return (&listCmd{}).list(fmt.Sprintf(`name: %s
repos path:
{{- with profile %q -}}
{{- range .ReposPath }}
  {{ . }}
{{- end -}}
{{- end }}
`, profileName, profileName))
}

func (cmd *profileCmd) doList(args []string) error {
	return (&listCmd{}).list(`
{{- range .Profiles -}}
{{- if eq .Name $.CurrentProfileName -}}*{{- else }} {{ end }} {{ .Name }}
{{ end -}}
`)
}

func (cmd *profileCmd) doNew(args []string) error {
	if len(args) == 0 {
		cmd.FlagSet().Usage()
		logger.Error("'volt profile new' receives profile name.")
		return nil
	}
	profileName := args[0]

	// Read lock.json
	lockJSON, err := lockjson.Read()
	if err != nil {
		return errors.New("failed to read lock.json: " + err.Error())
	}

	// Return error if profiles[]/name matches profileName
	_, err = lockJSON.Profiles.FindByName(profileName)
	if err == nil {
		return errors.New("profile '" + profileName + "' already exists")
	}

	// Begin transaction
	err = transaction.Create()
	if err != nil {
		return err
	}
	defer transaction.Remove()

	// Add profile
	lockJSON.Profiles = append(lockJSON.Profiles, lockjson.Profile{
		Name:      profileName,
		ReposPath: make([]pathutil.ReposPath, 0),
	})

	// Write to lock.json
	err = lockJSON.Write()
	if err != nil {
		return err
	}

	logger.Info("Created new profile '" + profileName + "'")

	return nil
}

func (cmd *profileCmd) doDestroy(args []string) error {
	if len(args) == 0 {
		cmd.FlagSet().Usage()
		logger.Error("'volt profile destroy' receives profile name.")
		return nil
	}

	// Read lock.json
	lockJSON, err := lockjson.Read()
	if err != nil {
		return errors.New("failed to read lock.json: " + err.Error())
	}

	// Begin transaction
	err = transaction.Create()
	if err != nil {
		return err
	}
	defer transaction.Remove()

	var merr *multierror.Error
	for i := range args {
		profileName := args[i]

		// Skip if current profile matches profileName
		if lockJSON.CurrentProfileName == profileName {
			merr = multierror.Append(merr, errors.New("cannot destroy current profile: "+profileName))
			continue
		}
		// Skip if profiles[]/name does not match profileName
		index := lockJSON.Profiles.FindIndexByName(profileName)
		if index < 0 {
			merr = multierror.Append(merr, errors.New("profile '"+profileName+"' does not exist"))
			continue
		}

		// Remove the specified profile
		lockJSON.Profiles = append(lockJSON.Profiles[:index], lockJSON.Profiles[index+1:]...)

		// Remove $VOLTPATH/rc/{profile} dir
		rcDir := pathutil.RCDir(profileName)
		os.RemoveAll(rcDir)
		if pathutil.Exists(rcDir) {
			return errors.New("failed to remove " + rcDir)
		}

		logger.Info("Deleted profile '" + profileName + "'")
	}

	// Write to lock.json
	err = lockJSON.Write()
	if err != nil {
		return err
	}

	return merr.ErrorOrNil()
}

func (cmd *profileCmd) doRename(args []string) error {
	if len(args) != 2 {
		cmd.FlagSet().Usage()
		logger.Error("'volt profile rename' receives profile name.")
		return nil
	}
	oldName := args[0]
	newName := args[1]

	// Read lock.json
	lockJSON, err := lockjson.Read()
	if err != nil {
		return errors.New("failed to read lock.json: " + err.Error())
	}

	// Return error if profiles[]/name does not match oldName
	index := lockJSON.Profiles.FindIndexByName(oldName)
	if index < 0 {
		return errors.New("profile '" + oldName + "' does not exist")
	}

	// Return error if profiles[]/name does not match newName
	if lockJSON.Profiles.FindIndexByName(newName) >= 0 {
		return errors.New("profile '" + newName + "' already exists")
	}

	// Begin transaction
	err = transaction.Create()
	if err != nil {
		return err
	}
	defer transaction.Remove()

	// Rename profile names
	lockJSON.Profiles[index].Name = newName
	if lockJSON.CurrentProfileName == oldName {
		lockJSON.CurrentProfileName = newName
	}

	// Rename $VOLTPATH/rc/{profile} dir
	oldRCDir := pathutil.RCDir(oldName)
	if pathutil.Exists(oldRCDir) {
		newRCDir := pathutil.RCDir(newName)
		if err = os.Rename(oldRCDir, newRCDir); err != nil {
			return fmt.Errorf("could not rename %s to %s", oldRCDir, newRCDir)
		}
	}

	// Write to lock.json
	err = lockJSON.Write()
	if err != nil {
		return err
	}

	logger.Infof("Renamed profile '%s' to '%s'", oldName, newName)

	return nil
}

func (cmd *profileCmd) doAdd(args []string) error {
	// Read lock.json
	lockJSON, err := lockjson.Read()
	if err != nil {
		return errors.New("failed to read lock.json: " + err.Error())
	}

	// Parse args
	profileName, reposPathList, err := cmd.parseAddArgs(lockJSON, "add", args)
	if err != nil {
		return errors.New("failed to parse args: " + err.Error())
	}

	if profileName == "-current" {
		profileName = lockJSON.CurrentProfileName
	}

	// Read modified profile and write to lock.json
	lockJSON, err = cmd.transactProfile(lockJSON, profileName, func(profile *lockjson.Profile) {
		// Add repositories to profile if the repository does not exist
		for _, reposPath := range reposPathList {
			if profile.ReposPath.Contains(reposPath) {
				logger.Warn("repository '" + reposPath.String() + "' is already enabled")
			} else {
				profile.ReposPath = append(profile.ReposPath, reposPath)
				logger.Info("Enabled '" + reposPath.String() + "' on profile '" + profileName + "'")
			}
		}
	})
	if err != nil {
		return err
	}

	// Build ~/.vim/pack/volt dir
	err = builder.Build(false)
	if err != nil {
		return errors.New("could not build " + pathutil.VimVoltDir() + ": " + err.Error())
	}

	return nil
}

func (cmd *profileCmd) doRm(args []string) error {
	// Read lock.json
	lockJSON, err := lockjson.Read()
	if err != nil {
		return errors.New("failed to read lock.json: " + err.Error())
	}

	// Parse args
	profileName, reposPathList, err := cmd.parseAddArgs(lockJSON, "rm", args)
	if err != nil {
		return errors.New("failed to parse args: " + err.Error())
	}

	if profileName == "-current" {
		profileName = lockJSON.CurrentProfileName
	}

	// Read modified profile and write to lock.json
	lockJSON, err = cmd.transactProfile(lockJSON, profileName, func(profile *lockjson.Profile) {
		// Remove repositories from profile if the repository does not exist
		for _, reposPath := range reposPathList {
			index := profile.ReposPath.IndexOf(reposPath)
			if index >= 0 {
				// Remove profile.ReposPath[index]
				profile.ReposPath = append(profile.ReposPath[:index], profile.ReposPath[index+1:]...)
				logger.Info("Disabled '" + reposPath.String() + "' from profile '" + profileName + "'")
			} else {
				logger.Warn("repository '" + reposPath.String() + "' is already disabled")
			}
		}
	})
	if err != nil {
		return err
	}

	// Build ~/.vim/pack/volt dir
	err = builder.Build(false)
	if err != nil {
		return errors.New("could not build " + pathutil.VimVoltDir() + ": " + err.Error())
	}

	return nil
}

func (cmd *profileCmd) parseAddArgs(lockJSON *lockjson.LockJSON, subCmd string, args []string) (string, []pathutil.ReposPath, error) {
	if len(args) == 0 {
		cmd.FlagSet().Usage()
		logger.Errorf("'volt profile %s' receives profile name and one or more repositories.", subCmd)
		return "", nil, nil
	}

	profileName := args[0]
	reposPathList := make([]pathutil.ReposPath, 0, len(args)-1)
	for _, arg := range args[1:] {
		reposPath, err := pathutil.NormalizeRepos(arg)
		if err != nil {
			return "", nil, err
		}
		reposPathList = append(reposPathList, reposPath)
	}

	// Validate if all repositories exist in repos[]
	for i := range reposPathList {
		_, err := lockJSON.Repos.FindByPath(reposPathList[i])
		if err != nil {
			return "", nil, err
		}
	}

	return profileName, reposPathList, nil
}

// Run modifyProfile and write modified structure to lock.json
func (*profileCmd) transactProfile(lockJSON *lockjson.LockJSON, profileName string, modifyProfile func(*lockjson.Profile)) (*lockjson.LockJSON, error) {
	// Return error if profiles[]/name does not match profileName
	profile, err := lockJSON.Profiles.FindByName(profileName)
	if err != nil {
		return nil, err
	}

	// Begin transaction
	err = transaction.Create()
	if err != nil {
		return nil, err
	}
	defer transaction.Remove()

	modifyProfile(profile)

	// Write to lock.json
	err = lockJSON.Write()
	if err != nil {
		return nil, err
	}
	return lockJSON, nil
}
