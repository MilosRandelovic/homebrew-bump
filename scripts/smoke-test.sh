#!/usr/bin/env bash
# Verify the locally built Bump CLI and MCP binaries without contacting a package registry.
# Bash is required for regular-expression matching and the script runs on macOS/Homebrew CI.
# Usage: smoke-test.sh [bump-binary] [bump-mcp-binary]
# Options: --help  Show this help text.

set -euo pipefail

if [ "${1:-}" = "--help" ]; then
  sed -n '2,5s/^# //p' "$0"
  exit 0
fi

SCRIPT_DIRECTORY="$(cd "$(dirname "$0")" && pwd)"
REPOSITORY_ROOT="$(cd "$SCRIPT_DIRECTORY/.." && pwd)"

BINARY_PATH="${1:-$REPOSITORY_ROOT/bump}"
BINARY_DIRECTORY="$(cd "$(dirname "$BINARY_PATH")" && pwd)"
BINARY_PATH="$BINARY_DIRECTORY/$(basename "$BINARY_PATH")"
MCP_BINARY_PATH="${2:-$REPOSITORY_ROOT/bump-mcp}"
MCP_BINARY_DIRECTORY="$(cd "$(dirname "$MCP_BINARY_PATH")" && pwd)"
MCP_BINARY_PATH="$MCP_BINARY_DIRECTORY/$(basename "$MCP_BINARY_PATH")"

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

MCP_VERSION_OUTPUT="$("$MCP_BINARY_PATH" --version)"
if [[ ! "$MCP_VERSION_OUTPUT" =~ ^bump-mcp\ version\ [0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z.-]+)?$ ]]; then
  echo "Unexpected MCP version output: $MCP_VERSION_OUTPUT" >&2
  exit 1
fi

if [ "${MCP_VERSION_OUTPUT##* }" != "${VERSION_OUTPUT##* }" ]; then
  echo "Bump and bump-mcp versions do not match: $VERSION_OUTPUT; $MCP_VERSION_OUTPUT" >&2
  exit 1
fi

TEMPORARY_DIRECTORY="$(mktemp -d)"
trap 'rm -rf "$TEMPORARY_DIRECTORY"' EXIT
ruby -rjson -ropen3 -rtimeout -e '
  request = {
    jsonrpc: "2.0",
    id: 1,
    method: "initialize",
    params: {
      protocolVersion: "2025-03-26",
      capabilities: {},
      clientInfo: { name: "bump-smoke", version: "1.0.0" }
    }
  }
  input = output = process = nil
  begin
    Timeout.timeout(5) do
      input, output, process = Open3.popen2(ARGV.fetch(0))
      input.puts(JSON.generate(request))
      input.close
      line = output.gets
      raise "no response" if line.nil?
      response = JSON.parse(line)
      result = response.is_a?(Hash) ? response["result"] : nil
      server_info = result.is_a?(Hash) ? result["serverInfo"] : nil
      valid = response.is_a?(Hash) && response["jsonrpc"] == "2.0" && response["id"] == 1 &&
        result.is_a?(Hash) && result["protocolVersion"].is_a?(String) &&
        server_info.is_a?(Hash) && server_info["name"].is_a?(String)
      raise "invalid response" unless valid
      raise "server exited unsuccessfully" unless process.value.success?
    end
  rescue Timeout::Error
    warn "MCP initialize timed out"
    exit 1
  rescue StandardError => error
    warn "MCP initialize failed: #{error.message}"
    exit 1
  ensure
    input.close if input && !input.closed?
    output.close if output && !output.closed?
    if process && process.alive?
      begin
        Process.kill("TERM", process.pid)
      rescue Errno::ESRCH
      end
      unless process.join(1)
        begin
          Process.kill("KILL", process.pid)
        rescue Errno::ESRCH
        end
        process.join
      end
    end
  end
' "$MCP_BINARY_PATH"

SCRIPT_HELP_OUTPUT="$("$SCRIPT_DIRECTORY/smoke-test.sh" --help)"
if ! grep -Fq 'Usage: smoke-test.sh [bump-binary] [bump-mcp-binary]' <<<"$SCRIPT_HELP_OUTPUT"; then
  echo "Smoke script help does not contain its usage" >&2
  exit 1
fi
if ! grep -Fq 'Options: --help  Show this help text.' <<<"$SCRIPT_HELP_OUTPUT"; then
  echo "Smoke script help does not contain its options" >&2
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

if (cd "$TEMPORARY_DIRECTORY" && "$BINARY_PATH" >stdout.log 2>stderr.log); then
  echo "Expected bump to fail when no dependency file exists" >&2
  exit 1
fi

if ! grep -Fq "no package.json or pubspec.yaml found" "$TEMPORARY_DIRECTORY/stderr.log"; then
  echo "Missing dependency-file error was not reported" >&2
  exit 1
fi
