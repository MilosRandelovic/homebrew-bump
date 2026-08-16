#!/usr/bin/env bash

set -euo pipefail

BINARY_PATH="${1:-./bump}"
BINARY_DIRECTORY="$(cd "$(dirname "$BINARY_PATH")" && pwd)"
BINARY_PATH="$BINARY_DIRECTORY/$(basename "$BINARY_PATH")"

VERSION_OUTPUT="$("$BINARY_PATH" --version)"
if [[ ! "$VERSION_OUTPUT" =~ ^bump\ version\ [0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z.-]+)?$ ]]; then
  echo "Unexpected version output: $VERSION_OUTPUT" >&2
  exit 1
fi

MINIMUM_AGE_VERSION_OUTPUT="$("$BINARY_PATH" -aV)"
if [ "$MINIMUM_AGE_VERSION_OUTPUT" != "$VERSION_OUTPUT" ]; then
  echo "Merged minimum-age shorthand produced unexpected version output: $MINIMUM_AGE_VERSION_OUTPUT" >&2
  exit 1
fi

HELP_OUTPUT="$("$BINARY_PATH" --help)"
if ! grep -Fq "Usage: bump [options]" <<<"$HELP_OUTPUT"; then
  echo "Help output does not contain the usage line" >&2
  exit 1
fi

if ! grep -Fq -- "--minimum-age, -a" <<<"$HELP_OUTPUT"; then
  echo "Help output does not document the minimum-age flag" >&2
  exit 1
fi

TEMPORARY_DIRECTORY="$(mktemp -d)"
trap 'rm -rf "$TEMPORARY_DIRECTORY"' EXIT

if (cd "$TEMPORARY_DIRECTORY" && "$BINARY_PATH" >stdout.log 2>stderr.log); then
  echo "Expected bump to fail when no dependency file exists" >&2
  exit 1
fi

if ! grep -Fq "no package.json or pubspec.yaml found" "$TEMPORARY_DIRECTORY/stderr.log"; then
  echo "Missing dependency-file error was not reported" >&2
  exit 1
fi
