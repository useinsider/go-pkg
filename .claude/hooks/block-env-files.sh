#!/bin/sh
# PreToolUse hook: refuse edits/writes that touch env files or secret-ish paths.
# Exits 2 to block the tool call. Belt-and-suspenders with .gitignore.
set -eu

file=$(jq -r '.tool_input.file_path // empty')

if [ -z "$file" ]; then
  exit 0
fi

case "$file" in
  *.env|*.env.*|\
  *.pem|*.key|\
  *credentials|*credentials.*|\
  *service-account*.json|\
  *id_rsa|*id_rsa.*)
    echo "BLOCK: refusing to edit secret-ish path: $file" >&2
    echo "If this is a false positive, bypass manually." >&2
    exit 2
    ;;
esac

exit 0
