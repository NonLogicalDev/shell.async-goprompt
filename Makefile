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

.PHONY: clean
clean:
	rm -rf dist
