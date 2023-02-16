package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	goprompt "github.com/NonLogicalDev/shell.async-goprompt"
	"github.com/kballard/go-shellquote"
	"github.com/spf13/cobra"
)

var cmdInstallHelpLong = `
# To install run the following:

	$ goprompt install zsh >> .zshrc

# [FILE] options:

	* zsh
	* zsh.plugin
`

const defaultContent = `
# To try for just this session run the following:

	$ eval "$({{goprompt}} install zsh)"

# To install run the following:

	$ {{goprompt}} install zsh >> .zshrc
`

var (
	cmdInstall = &cobra.Command{
		Use:   "install [FILE]",
		Short: "print out the shell script to setup prompt",
		Long: trim(cmdInstallHelpLong),
	}
)

func init() {
	cmdInstall.RunE = cmdInstallRun
}

func replacePlaceholders(content string) string {
	goPromptExec := os.Args[0]
	if strings.Contains(goPromptExec, "/") {
		if fullPath, err := filepath.Abs(goPromptExec); err == nil {
			goPromptExec = fullPath
		}
	}
	goPromptExec = shellquote.Join(goPromptExec)
	content = strings.ReplaceAll(content, "{{goprompt}}", goPromptExec)
	content = strings.ReplaceAll(content, "${GOPROMPT}", goPromptExec)
	return content
}

func cmdInstallRun(command *cobra.Command, args []string) error {
	argFile := ""
	if len(args) > 0 {
		argFile = args[0]
	}

	var content string
	switch argFile {
	case "zsh":
		f, _ := goprompt.ZSHPluginFiles.ReadFile("plugin/zsh/prompt_install.zsh")
		content = replacePlaceholders(string(f))
	case "zsh.plugin":
		f, _ := goprompt.ZSHPluginFiles.ReadFile("plugin/zsh/prompt_asynczle_setup.zsh")
		content = replacePlaceholders(string(f))
	default:
		content = replacePlaceholders(defaultContent)
	}
	fmt.Println(content)
	return nil
}
