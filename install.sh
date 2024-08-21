#!/usr/bin/env sh
set -e

PACKAGE=NonLogicalDev/shell.async-goprompt
FILE_BASENAME="goprompt"
FILE_EXEC="goprompt"

RELEASES_URL="https://github.com/$PACKAGE/releases"
LATEST=$(curl "https://api.github.com/repos/$PACKAGE/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
LATEST_MARK=$(printf "%s" "$LATEST" | sed 's/^v//' )

test -z "$VERSION" && VERSION="$LATEST"

test -z "$VERSION" && {
	echo "Unable to get goprompt version." >&2
	exit 1
}

TMP_DIR="$(mktemp -d)"
# shellcheck disable=SC2064 # intentionally expands here
trap "rm -rf \"$TMP_DIR\"" EXIT INT TERM

OS="$( uname -s | tr '[:upper:]' '[:lower:]' )"
ARCH="$(uname -m | tr '[:upper:]' '[:lower:]' )"
test "$ARCH" = "aarch64" && ARCH="arm64"
FULL_EXEC_FILE="${FILE_BASENAME}_${LATEST_MARK}_${OS}_${ARCH}"

(
	cd "$TMP_DIR"

	echo "Downloading $VERSION..."
	curl -sfLO "$RELEASES_URL/download/$VERSION/$FULL_EXEC_FILE"
	echo "OK: download"

	echo "Verifying checksums..."
    if which sha256sum 2>/dev/null 1>/dev/null; then
	    curl -sfLO "$RELEASES_URL/download/$VERSION/checksums.txt"
	    sha256sum --ignore-missing --quiet --check checksums.txt
	    echo "OK: checksum"
    else
        echo "Can not validate checksums sha256sum command missing..."
    fi

    chmod u+x "$FULL_EXEC_FILE"
)

mkdir -p ~/.local/bin
mv "$TMP_DIR/$FULL_EXEC_FILE" ~/.local/bin/"$FILE_EXEC"
echo "DONE: $FILE_EXEC is available at ~/.local/bin/$FILE_EXEC"
