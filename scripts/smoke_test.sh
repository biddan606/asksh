#!/usr/bin/env bash
# Smoke test: build the binary and verify basic invocations.
# Usage: ./scripts/smoke_test.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BIN=$(mktemp)
trap 'rm -f "$BIN"' EXIT

echo "Building binary..."
go build -o "$BIN" "$ROOT"

pass() { echo "  PASS: $1"; }
fail() { echo "  FAIL: $1"; exit 1; }

echo ""
echo "=== smoke: --version ==="
OUT=$("$BIN" --version 2>&1)
echo "$OUT"
echo "$OUT" | grep -qE "0\.[0-9]" || fail "version string missing"
pass "--version exits 0 with version string"

echo ""
echo "=== smoke: --help ==="
"$BIN" --help >/dev/null
pass "--help exits 0"

echo ""
echo "=== smoke: --dry-run safe query ==="
OUT=$("$BIN" --dry-run "list files" 2>&1)
echo "$OUT"
echo "$OUT" | grep -q "query: list files" || fail "missing 'query:' line"
echo "$OUT" | grep -qi "SAFE"            || fail "missing SAFE verdict"
pass "--dry-run safe exits 0 with SAFE verdict"

echo ""
echo "=== smoke: --dry-run blocked query ==="
set +e
OUT=$("$BIN" --dry-run "rm -rf /" 2>&1)
RC=$?
set -e
echo "$OUT"
[ "$RC" -ne 0 ]          || fail "blocked command should exit non-zero (got 0)"
echo "$OUT" | grep -qi "BLOCKED" || fail "missing BLOCKED in output"
pass "--dry-run blocked exits non-zero with BLOCKED"

echo ""
echo "=== smoke: no args ==="
set +e
"$BIN" >/dev/null 2>&1
RC=$?
set -e
[ "$RC" -ne 0 ] || fail "no-args should exit non-zero"
pass "no-args exits non-zero"

echo ""
echo "=== smoke: --ollama and --openai mutually exclusive ==="
set +e
"$BIN" --ollama --openai --dry-run "x" >/dev/null 2>&1
RC=$?
set -e
[ "$RC" -ne 0 ] || fail "--ollama --openai together should exit non-zero"
pass "--ollama --openai mutual exclusion enforced"

echo ""
echo "All smoke tests passed."
