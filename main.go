package main

import (
	"fmt"
	"github.com/roots/trellis-cli/cmd"
	"github.com/roots/trellis-cli/config"
	"github.com/roots/trellis-cli/github"
	"github.com/roots/trellis-cli/plugin"
	"github.com/roots/trellis-cli/trellis"
	"github.com/roots/trellis-cli/update"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/fatih/color"
	"github.com/mitchellh/cli"
)

// To be replaced by goreleaser build flags.
var version = "canary"
var updaterRepo = ""

func main() {
	cacheDir, _ := config.Scope.CacheDir()

	updateNotifier := &update.Notifier{
		CacheDir: cacheDir,
		Client:   github.Client,
		Repo:     updaterRepo,
		Version:  version,
	}

	updateMessageChan := make(chan *github.Release)
	go func() {
		release, _ := updateNotifier.CheckForUpdate()
		updateMessageChan <- release
	}()

	c := cli.NewCLI("trellis", version)
	c.Args = os.Args[1:]

	ui := &cli.ColoredUi{
		ErrorColor: cli.UiColorRed,
		WarnColor:  cli.UiColor{Code: int(color.FgYellow), Bold: false},
		Ui: &cli.BasicUi{
			Reader:      os.Stdin,
			Writer:      os.Stdout,
			ErrorWriter: os.Stderr,
		},
	}

	project := &trellis.Project{}
	trellis := trellis.NewTrellis(project)

	c.Commands = map[string]cli.CommandFactory{
		"alias": func() (cli.Command, error) {
			return cmd.NewAliasCommand(ui, trellis), nil
		},
		"check": func() (cli.Command, error) {
			return &cmd.CheckCommand{UI: ui, Trellis: trellis}, nil
		},
		"db": func() (cli.Command, error) {
			return &cmd.NamespaceCommand{
				HelpText:     "Usage: trellis db <subcommand> [<args>]",
				SynopsisText: "Commands for database management",
			}, nil
		},
		"db open": func() (cli.Command, error) {
			return cmd.NewDBOpenCommand(ui, trellis), nil
		},
		"deploy": func() (cli.Command, error) {
			return cmd.NewDeployCommand(ui, trellis), nil
		},
		"dotenv": func() (cli.Command, error) {
			return cmd.NewDotEnvCommand(ui, trellis), nil
		},
		"down": func() (cli.Command, error) {
			return &cmd.DownCommand{UI: ui, Trellis: trellis}, nil
		},
		"droplet": func() (cli.Command, error) {
			return &cmd.NamespaceCommand{
				HelpText:     "Usage: trellis droplet <subcommand> [<args>]",
				SynopsisText: "Commands for DigitalOcean Droplets",
			}, nil
		},
		"droplet create": func() (cli.Command, error) {
			return cmd.NewDropletCreateCommand(ui, trellis), nil
		},
		"exec": func() (cli.Command, error) {
			return &cmd.ExecCommand{UI: ui, Trellis: trellis}, nil
		},
		"galaxy": func() (cli.Command, error) {
			return &cmd.NamespaceCommand{
				HelpText:     "Usage: trellis galaxy <subcommand> [<args>]",
				SynopsisText: "Commands for Ansible Galaxy",
			}, nil
		},
		"galaxy install": func() (cli.Command, error) {
			return &cmd.GalaxyInstallCommand{UI: ui, Trellis: trellis}, nil
		},
		"info": func() (cli.Command, error) {
			return &cmd.InfoCommand{UI: ui, Trellis: trellis}, nil
		},
		"init": func() (cli.Command, error) {
			return &cmd.InitCommand{UI: ui, Trellis: trellis}, nil
		},
		"new": func() (cli.Command, error) {
			return cmd.NewNewCommand(ui, trellis, c.Version), nil
		},
		"provision": func() (cli.Command, error) {
			return cmd.NewProvisionCommand(ui, trellis), nil
		},
		"rollback": func() (cli.Command, error) {
			return cmd.NewRollbackCommand(ui, trellis), nil
		},
		"shell-init": func() (cli.Command, error) {
			return &cmd.ShellInitCommand{ui}, nil
		},
		"ssh": func() (cli.Command, error) {
			return &cmd.SshCommand{ui, trellis}, nil
		},
		"up": func() (cli.Command, error) {
			return cmd.NewUpCommand(ui, trellis), nil
		},
		"vault": func() (cli.Command, error) {
			return &cmd.NamespaceCommand{
				HelpText:     "Usage: trellis vault <subcommand> [<args>]",
				SynopsisText: "Commands for Ansible Vault",
			}, nil
		},
		"vault edit": func() (cli.Command, error) {
			return &cmd.VaultEditCommand{ui, trellis}, nil
		},
		"vault encrypt": func() (cli.Command, error) {
			return cmd.NewVaultEncryptCommand(ui, trellis), nil
		},
		"vault decrypt": func() (cli.Command, error) {
			return cmd.NewVaultDecryptCommand(ui, trellis), nil
		},
		"vault view": func() (cli.Command, error) {
			return cmd.NewVaultViewCommand(ui, trellis), nil
		},
		"valet": func() (cli.Command, error) {
			return &cmd.NamespaceCommand{
				HelpText:     "Usage: trellis valet <subcommand> [<args>]",
				SynopsisText: "Commands for Laravel Valet",
			}, nil
		},
		"valet link": func() (cli.Command, error) {
			return &cmd.ValetLinkCommand{UI: ui, Trellis: trellis}, nil
		},
		"venv hook": func() (cli.Command, error) {
			return &cmd.VenvHookCommand{UI: ui, Trellis: trellis}, nil
		},
	}

	c.HiddenCommands = []string{"venv", "venv hook"}

	if shouldSkipPlugins, _ := strconv.ParseBool(os.Getenv("TRELLIS_NO_PLUGINS")); !shouldSkipPlugins {
		pluginPaths := filepath.SplitList(os.Getenv("PATH"))
		plugin.Register(c, pluginPaths, []string{"trellis"})
	}

	exitStatus, err := c.Run()

	if err != nil {
		log.Println(err)
	}

	newRelease := <-updateMessageChan
	if newRelease != nil {
		msg := fmt.Sprintf(
			"\n%s %s → %s\n%s",
			color.YellowString("A new release of trellis-cli is available:"),
			color.CyanString(version),
			color.CyanString(newRelease.Version),
			color.YellowString(newRelease.URL),
		)

		ui.Info(msg)
	}

	os.Exit(exitStatus)
}
