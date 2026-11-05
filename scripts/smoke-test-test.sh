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
  no-start) IFS= read -r _; exit 1 ;;
  stalled) exec sleep 30 ;;
esac
IFS= read -r _
protocol_field='"protocolVersion":"2025-03-26"'
jsonrpc_field='"jsonrpc":"2.0"'
id_field='"id":1'
capabilities_field='"capabilities":{}'
name_field='"name":"fixture"'
version_field='"version":"1.0.0"'
case "$SMOKE_SCENARIO" in
  invalid-response) echo '{"jsonrpc":"2.0","id":1,"result":{}}'; exit 0 ;;
  jsonrpc-absent) jsonrpc_field='"other":"value"' ;;
  jsonrpc-wrong) jsonrpc_field='"jsonrpc":"1.0"' ;;
  id-absent) id_field='"otherId":1' ;;
  id-wrong) id_field='"id":2' ;;
  unsupported-protocol) protocol_field='"protocolVersion":"2024-11-05"' ;;
  protocol-wrong-type) protocol_field='"protocolVersion":42' ;;
  protocol-absent) protocol_field='"other":"value"' ;;
  capabilities-wrong-type) capabilities_field='"capabilities":[]' ;;
  capabilities-absent) capabilities_field='"other":"value"' ;;
  name-empty) name_field='"name":""' ;;
  name-wrong-type) name_field='"name":42' ;;
  name-absent) name_field='"other":"value"' ;;
  version-empty) version_field='"version":""' ;;
  version-wrong-type) version_field='"version":42' ;;
  version-absent) version_field='"other":"value"' ;;
esac
printf '{%s,%s,"result":{%s,%s,"serverInfo":{%s,%s}}}\n' \
  "$jsonrpc_field" "$id_field" "$protocol_field" "$capabilities_field" "$name_field" "$version_field"
if [ "$SMOKE_SCENARIO" = 'does-not-exit' ]; then
  exec sleep 30
fi
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
run_failure_case jsonrpc-absent 'invalid response'
run_failure_case jsonrpc-wrong 'invalid response'
run_failure_case id-absent 'invalid response'
run_failure_case id-wrong 'invalid response'
run_failure_case unsupported-protocol 'invalid response'
run_failure_case protocol-wrong-type 'invalid response'
run_failure_case protocol-absent 'invalid response'
run_failure_case capabilities-wrong-type 'invalid response'
run_failure_case capabilities-absent 'invalid response'
run_failure_case name-empty 'invalid response'
run_failure_case name-wrong-type 'invalid response'
run_failure_case name-absent 'invalid response'
run_failure_case version-empty 'invalid response'
run_failure_case version-wrong-type 'invalid response'
run_failure_case version-absent 'invalid response'
run_failure_case no-start 'no response'
run_failure_case stalled 'MCP initialize timed out'
run_failure_case does-not-exit 'MCP initialize timed out'

rm -f "$TEMPORARY_DIRECTORY/mcp.pid"
if ! SMOKE_SCENARIO=valid SMOKE_PID_FILE="$TEMPORARY_DIRECTORY/mcp.pid" \
  "$SCRIPT_DIRECTORY/smoke-test.sh" "$TEMPORARY_DIRECTORY/bump" "$TEMPORARY_DIRECTORY/bump-mcp" \
  >"$TEMPORARY_DIRECTORY/stdout.log" 2>"$TEMPORARY_DIRECTORY/stderr.log"; then
  echo "Smoke test rejected the complete valid fixture: $(<"$TEMPORARY_DIRECTORY/stderr.log")" >&2
  exit 1
fi
if [ ! -f "$TEMPORARY_DIRECTORY/mcp.pid" ] || kill -0 "$(<"$TEMPORARY_DIRECTORY/mcp.pid")" 2>/dev/null; then
  echo "Smoke test did not reap the valid fixture" >&2
  exit 1
fi
