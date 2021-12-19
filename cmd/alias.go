package cmd

import (
	_ "embed"
	"flag"
	"fmt"
	"github.com/fatih/color"
	"github.com/mitchellh/cli"
	"github.com/posener/complete"
	"github.com/roots/trellis-cli/trellis"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type AliasCommand struct {
	UI                cli.Ui
	flags             *flag.FlagSet
	Trellis           *trellis.Trellis
	local             string
	aliasPlaybook     PlaybookRunner
	aliasCopyPlaybook PlaybookRunner
}

//go:embed files/playbooks/alias.yml
var aliasYml string

const aliasYmlJ2 = `
@{{ env }}:
  ssh: "{{ web_user }}@{{ ansible_host }}:{{ ansible_port | default('22') }}"
  path: "{{ project_root | default(www_root + '/' + item.key) | regex_replace('^~\/','') }}/{{ item.current_path | default('current') }}/web/wp"
`

//go:embed files/playbooks/alias_copy.yml
var aliasCopyYml string

func NewAliasCommand(ui cli.Ui, trellis *trellis.Trellis) *AliasCommand {
	aliasPlaybook := &AdHocPlaybook{
		files: map[string]string{
			"alias.yml":    aliasYml,
			"alias.yml.j2": strings.TrimSpace(aliasYmlJ2) + "\n",
		},
		Playbook: Playbook{
			ui: ui,
		},
	}

	aliasCopyPlaybook := &AdHocPlaybook{
		files: map[string]string{
			"alias-copy.yml": aliasCopyYml,
		},
		Playbook: Playbook{
			ui: ui,
		},
	}

	c := &AliasCommand{UI: ui, Trellis: trellis, aliasPlaybook: aliasPlaybook, aliasCopyPlaybook: aliasCopyPlaybook}
	c.init()
	return c
}

func (c *AliasCommand) init() {
	c.flags = flag.NewFlagSet("", flag.ContinueOnError)
	c.flags.Usage = func() { c.UI.Info(c.Help()) }
	c.flags.StringVar(&c.local, "local", "development", "local environment name, default: development")
}

func (c *AliasCommand) Run(args []string) int {
	if err := c.Trellis.LoadProject(); err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	if err := c.flags.Parse(args); err != nil {
		return 1
	}

	args = c.flags.Args()

	commandArgumentValidator := &CommandArgumentValidator{required: 0, optional: 0}
	commandArgumentErr := commandArgumentValidator.validate(args)
	if commandArgumentErr != nil {
		c.UI.Error(commandArgumentErr.Error())
		c.UI.Output(c.Help())
		return 1
	}

	environments := c.Trellis.EnvironmentNames()
	var remoteEnvironments []string
	for _, environment := range environments {
		if environment != c.local {
			remoteEnvironments = append(remoteEnvironments, environment)
		}
	}

	tempDir, tempDirErr := ioutil.TempDir("", "trellis-alias-")
	if tempDirErr != nil {
		c.UI.Error(tempDirErr.Error())
		return 1
	}
	defer os.RemoveAll(tempDir)

	c.aliasPlaybook.SetRoot(c.Trellis.Path)

	for _, environment := range remoteEnvironments {
		args := []string{
			"-vvv",
			"-e", "env=" + environment,
			"-e", "trellis_alias_j2=alias.yml.j2",
			"-e", "trellis_alias_temp_dir=" + tempDir,
		}
		if err := c.aliasPlaybook.Run("alias.yml", args); err != nil {
			c.UI.Error(fmt.Sprintf("Error running ansible-playbook alias.yml: %s", err))
			return 1
		}
	}

	combined := ""
	for _, environment := range remoteEnvironments {
		part, err := ioutil.ReadFile(filepath.Join(tempDir, environment+".yml.part"))
		if err != nil {
			c.UI.Error(err.Error())
			return 1
		}
		combined = combined + string(part)
	}

	combinedYmlPath := filepath.Join(tempDir, "/combined.yml")
	writeFileErr := ioutil.WriteFile(combinedYmlPath, []byte(combined), 0644)
	if writeFileErr != nil {
		c.UI.Error(writeFileErr.Error())
		return 1
	}

	c.aliasCopyPlaybook.SetRoot(c.Trellis.Path)

	if err := c.aliasCopyPlaybook.Run("alias-copy.yml", []string{"-e", "env=" + c.local, "-e", "trellis_alias_combined=" + combinedYmlPath}); err != nil {
		c.UI.Error(fmt.Sprintf("Error running ansible-playbook alias-copy.yml: %s", err))
		return 1
	}

	c.UI.Info(color.GreenString("✓ wp-cli.trellis-alias.yml generated"))
	message := `
Action Required: Add these lines into wp-cli.yml or wp-cli.local.yml

_: 
  inherit: wp-cli.trellis-alias.yml
`
	c.UI.Info(strings.TrimSpace(message))

	return 0
}

func (c *AliasCommand) Synopsis() string {
	return "Generate WP CLI aliases for remote environments"
}

func (c *AliasCommand) Help() string {
	helpText := `
Usage: trellis alias [options]

Generate WP CLI aliases for remote environments

Options:
      --local (default: development) Local environment name
  -h, --help  show this help
`

	return strings.TrimSpace(helpText)
}

func (c *AliasCommand) AutocompleteFlags() complete.Flags {
	return complete.Flags{
		"--local": complete.PredictNothing,
	}
}
