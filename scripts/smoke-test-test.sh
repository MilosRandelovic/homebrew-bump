#!/usr/bin/env bash
# Exercise smoke-test failure guards with local fixtures instead of registry access.
# Usage: smoke-test-test.sh
# Options: --help  Show this help text.

set -euo pipefail

if [ "${1:-}" = "--help" ]; then
  sed -n '2,4s/^# //p' "$0"
  exit 0
fi

SCRIPT_DIRECTORY="$(cd "$(dirname "$0")" && pwd)"
TEMPORARY_DIRECTORY="$(mktemp -d)"
trap 'rm -rf "$TEMPORARY_DIRECTORY"' EXIT

cat >"$TEMPORARY_DIRECTORY/bump" <<'SH'
#!/bin/sh
case "${1:-}" in
  --version|-aV) echo 'bump version 1.2.3' ;;
  --help) printf 'Usage: bump [options]\n  --minimum-age, -a\n' ;;
  *) echo 'no package.json or pubspec.yaml found' >&2; exit 1 ;;
esac
SH

cat >"$TEMPORARY_DIRECTORY/bump-mcp" <<'SH'
#!/bin/sh
if [ "${1:-}" = '--version' ]; then
  if [ "$SMOKE_SCENARIO" = 'version-mismatch' ]; then
    echo 'bump-mcp version 1.2.4'
  else
    echo 'bump-mcp version 1.2.3'
  fi
  exit 0
fi
echo "$$" >"$SMOKE_PID_FILE"
case "$SMOKE_SCENARIO" in
  invalid-response) echo '{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":42,"serverInfo":{"name":"fixture"}}}' ;;
  no-start) exit 1 ;;
  stalled) exec sleep 30 ;;
  does-not-exit)
    echo '{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2025-03-26","serverInfo":{"name":"fixture"}}}'
    exec sleep 30
    ;;
esac
SH

chmod +x "$TEMPORARY_DIRECTORY/bump" "$TEMPORARY_DIRECTORY/bump-mcp"

run_failure_case() {
  local scenario="$1"
  local expected="$2"
  local started_at=$SECONDS
  local process_id

  rm -f "$TEMPORARY_DIRECTORY/mcp.pid"
  if SMOKE_SCENARIO="$scenario" SMOKE_PID_FILE="$TEMPORARY_DIRECTORY/mcp.pid" \
    "$SCRIPT_DIRECTORY/smoke-test.sh" "$TEMPORARY_DIRECTORY/bump" "$TEMPORARY_DIRECTORY/bump-mcp" \
    >"$TEMPORARY_DIRECTORY/stdout.log" 2>"$TEMPORARY_DIRECTORY/stderr.log"; then
    echo "Smoke test unexpectedly accepted $scenario" >&2
    exit 1
  fi
  if ! grep -Fq "$expected" "$TEMPORARY_DIRECTORY/stderr.log"; then
    echo "Smoke test did not report $scenario: $(<"$TEMPORARY_DIRECTORY/stderr.log")" >&2
    exit 1
  fi
  if (( SECONDS - started_at > 8 )); then
    echo "Smoke test did not fail promptly for $scenario" >&2
    exit 1
  fi
  if [ -f "$TEMPORARY_DIRECTORY/mcp.pid" ]; then
    process_id="$(<"$TEMPORARY_DIRECTORY/mcp.pid")"
    if kill -0 "$process_id" 2>/dev/null; then
      echo "Smoke test left MCP process $process_id running for $scenario" >&2
      exit 1
    fi
  fi
}

run_failure_case version-mismatch 'versions do not match'
run_failure_case invalid-response 'invalid response'
run_failure_case no-start 'no response'
run_failure_case stalled 'MCP initialize timed out'
run_failure_case does-not-exit 'MCP initialize timed out'
