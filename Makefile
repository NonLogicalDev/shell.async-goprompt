MKFILE_PATH := $(abspath $(lastword $(MAKEFILE_LIST)))
CURRENT_DIR := $(patsubst %/,%,$(dir $(MKFILE_PATH)))

ZSH_PROMPT_SETUP_SCRIPT := $(CURRENT_DIR)/zsh/prompt_goprompt_setup.zsh

install:
	go install ./cmd/goprompt

prompt.source:
	@echo ". $(ZSH_PROMPT_SETUP_SCRIPT)"

try: install
	ZSH_DISABLE_PROMPT=Y ZSH_EXTRA_SOURCE="$(ZSH_PROMPT_SETUP_SCRIPT)" zsh
