export prefix?=$(HOME)/.local
export bindir?=$(prefix)/bin

MKFILE_PATH := $(abspath $(lastword $(MAKEFILE_LIST)))
CURRENT_DIR := $(patsubst %/,%,$(dir $(MKFILE_PATH)))

ZSH_PROMPT_SETUP_SCRIPT := $(CURRENT_DIR)/plugin/zsh/prompt_asynczle_setup.zsh

USR_BIN_DIR := $(HOME)/bin
USR_ZSH_DIR := $(HOME)/.local/share/zsh-funcs

.PHONY: publish
publish:
	goreleaser release --rm-dist

.PHONY: release
release:
	goreleaser release --rm-dist --snapshot --skip-publish

.PHONY: build
build:
	goreleaser build --rm-dist --snapshot --single-target --output dist/goprompt

.PHONY: install
install: build
	mkdir -p "$(USR_BIN_DIR)"
	cp dist/goprompt "$(USR_BIN_DIR)/goprompt"
	mkdir -p "$(USR_ZSH_DIR)"
	cp "$(ZSH_PROMPT_SETUP_SCRIPT)" "$(USR_ZSH_DIR)/prompt_asynczle_setup"
	$(MAKE) setup

.PHONY: setup
setup:
	@echo '# SETUP:' >&2
	@echo '# ------------------------------------------------------------------------------' >&2
	@echo '# Assuming GoPrompt installed in $(USR_BIN_DIR)' >&2
	@echo '# and zsh func in $(USR_ZSH_DIR)' >&2
	@echo '# ------------------------------------------------------------------------------' >&2
	@echo "# $$ make setup >> ~/.zshrc" >&2
	@echo '# ------------------------------------------------------------------------------' >&2
	@echo "# Add this to your ~/.zshenv" >&2
	@echo '# ------------------------------------------------------------------------------' >&2
	@echo ''
	@echo '# PROMPT_ASYNC_ZLE: ------------------------------------------------------------'
	@echo 'path+=( "$(USR_BIN_DIR)" )'
	@echo 'fpath+=( "$(USR_ZSH_DIR)" )'
	@echo 'autoload -Uz promptinit'
	@echo 'promptinit && prompt_asynczle_setup'
	@echo '# ------------------------------------------------------------------------------'

.PHONY: try
try: install
	@echo '>> THIS NEEDS EXTRA CONFIG <<'
	@echo '>> FOR DEVELOPMENT ONLY <<'
	ZSH_DISABLE_PROMPT=Y ZSH_EXTRA_SOURCE="$(ZSH_PROMPT_SETUP_SCRIPT)" zsh

.PHONY: clean
clean:
	rm -rf dist
