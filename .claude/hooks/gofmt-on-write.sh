#!/bin/sh
# PostToolUse hook: gofmt -w Go files edited by Claude Code.
# Informational — never blocks. Reads the tool-call JSON on stdin.
set -eu

file=$(jq -r '.tool_input.file_path // empty')

if [ -z "$file" ]; then
  exit 0
fi

case "$file" in
  *.go)
    if command -v gofmt >/dev/null 2>&1; then
      gofmt -s -w "$file" 2>/dev/null || true
    fi
    if command -v goimports >/dev/null 2>&1; then
      goimports -w "$file" 2>/dev/null || true
    fi
    ;;
esac

exit 0
