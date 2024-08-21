#!/bin/sh
set -euxo pipefail

readonly GORELEASER_TAR_URL=$1
readonly TARGET_BIN_PATH=$2
readonly TMPDIR=$(mktemp -d)

on_exit() {
    rm -rf "$TMPDIR"
}
trap on_exit EXIT

curl -sfL "${GORELEASER_TAR_URL}" -o "${TMPDIR}/goreleaser.tar.gz"
tar -xf "${TMPDIR}/goreleaser.tar.gz" -C "$TMPDIR"
cp "${TMPDIR}/goreleaser" "${TARGET_BIN_PATH}"
