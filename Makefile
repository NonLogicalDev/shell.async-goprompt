#? https://www.gnu.org/prep/standards/html_node/DESTDIR.html
export DESTDIR ?=
export PREFIX  ?= $(HOME)/.local

.default: info

# ------------------------------------------------------------------------------

MKFILE_PATH := $(abspath $(lastword $(MAKEFILE_LIST)))
CURRENT_DIR := $(patsubst %/,%,$(dir $(MKFILE_PATH)))
HOST_OS     := $(shell uname -s)
HOST_ARCH   := $(shell uname -m)

GORELEASER_VERSION := v2.0.1
GORELEASER_TAR_URL := https://github.com/goreleaser/goreleaser/releases/download/$(GORELEASER_VERSION)/goreleaser_$(HOST_OS)_$(HOST_ARCH).tar.gz

GORELEASER := ./.goreleaser

GO_FILES         = $(sort $(shell find . -type f -name '*.go') go.mod go.sum)
STATIC_FILES     = $(sort $(shell find ./plugin -type f))
GORELEASER_FILES = $(sort .goreleaser .goreleaser.yaml)

# ------------------------------------------------------------------------------

BUILD_DIST           ?= dist
BUILD_GORELEASER_BIN ?= $(BUILD_DIST)/goreleaser

INSTALL_BIN_DIR       ?= $(DESTDIR)$(PREFIX)/bin
INSTALL_ZSH_FUNCS_DIR ?= $(DESTDIR)$(PREFIX)/share/zsh-funcs

# ------------------------------------------------------------------------------

.PNONY: info
info:
	@echo "OS:   $(HOST_OS)"
	@echo "ARCH: $(HOST_ARCH)"
	@echo
	@echo "PREFIX:                $(PREFIX)"
	@echo "INSTALL_BIN_DIR:       $(INSTALL_BIN_DIR)"
	@echo "INSTALL_ZSH_FUNCS_DIR: $(INSTALL_ZSH_FUNCS_DIR)"

$(GORELEASER):
	sh scripts/fetch-goreleaser.sh "$(GORELEASER_TAR_URL)" "$(GORELEASER)"

$(INSTALL_BIN_DIR) $(INSTALL_ZSH_FUNCS_DIR):
	mkdir -p $@

# ------------------------------------------------------------------------------

.PHONY: clean build install release publish

clean:
	rm -rf dist

build: $(BUILD_GORELEASER_BIN)

install: $(BUILD_GORELEASER_BIN) $(INSTALL_BIN_DIR) $(INSTALL_ZSH_FUNCS_DIR)
	cp "$(BUILD_GORELEASER_BIN)" "$(INSTALL_BIN_DIR)/goprompt"

$(BUILD_GORELEASER_BIN): $(GO_FILES) $(STATIC_FILES) $(GORELEASER_FILES)
	$(GORELEASER) build --clean --snapshot --single-target \
		--id goreleaser \
		--output "$(BUILD_GORELEASER_BIN)"

release:
	$(GORELEASER) release --clean --auto-snapshot --skip=publish

publish:
	$(GORELEASER) release --clean
