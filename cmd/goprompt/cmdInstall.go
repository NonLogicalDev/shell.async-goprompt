package main

import (
	"fmt"

	goprompt "github.com/NonLogicalDev/shell.async-goprompt"
	"github.com/spf13/cobra"
)

var installDesc = `
To install run the following:

	$ goprompt install zsh.zshrc >> .zshrc

FILE options:

	* zsh.zshrc
	* zsh.plugin
`

var (
	cmdInstall = &cobra.Command{
		Use:   "install [FILE]",
		Short: "install the integration",
		Long: trim(installDesc),
		Args: cobra.MinimumNArgs(1),
	}
)

const (
	_zshRc = "zshrc"
	_zshPlugin = "zshplugin"
)

func init() {
	cmdInstall.RunE = cmdInstallRun
}

// TODO: bundle in the plugin directory, and provide a way to extract it into users directory of choice.

func cmdInstallRun(command *cobra.Command, args []string) error {
	var content string
	switch args[0] {
	case "zsh.zshrc":
		f, _ := goprompt.ZSHPluginFiles.ReadFile("plugin/zsh/prompt_install.zsh")
		content = string(f)
	case "zsh.plugin":
		f, _ := goprompt.ZSHPluginFiles.ReadFile("plugin/zsh/prompt_asynczle_setup.zsh")
		content = string(f)
	}
	fmt.Println(content)
	return nil
}
