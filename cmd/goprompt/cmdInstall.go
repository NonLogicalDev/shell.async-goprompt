package main

import (
	"fmt"
	"github.com/spf13/cobra"
)

var (
	cmdInstall = &cobra.Command{
		Use:   "install",
		Short: "install the integration",
	}
)

func init() {
	cmdInstall.RunE = cmdInstallRun
}

// TODO: bundle in the plugin directory, and provide a way to extract it into users directory of choice.

func cmdInstallRun(command *cobra.Command, args []string) error {
	fmt.Println(`
# SETUP:
# ------------------------------------------------------------------------------
# Assuming GoPrompt is installed in $(USR_BIN_DIR)
# and zsh func in $(USR_ZSH_DIR)
# ------------------------------------------------------------------------------
# $$ make setup >> ~/.zshrc"
# ------------------------------------------------------------------------------
# Add this to your ~/.zshenv
# ------------------------------------------------------------------------------

# PROMPT_ASYNC_ZLE: ------------------------------------------------------------
path+=( "$(USR_BIN_DIR)" )
fpath+=( "$(USR_ZSH_DIR)" )
autoload -Uz promptinit
promptinit && prompt_asynczle_setup
# ------------------------------------------------------------------------------
	`)
	return nil
}
