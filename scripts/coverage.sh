#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="${COVERAGE_OUT:-$ROOT/coverage.out}"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

MODULES="$(find "$ROOT" -name go.mod -not -path '*/vendor/*' -exec dirname {} \; | sort)"

i=0
FAILED=0
for mod in $MODULES; do
  i=$((i + 1))
  profile="$WORK/profile_$i.out"
  echo ">>> go test $(basename "$(dirname "$mod")")/$(basename "$mod")"
  if ! (cd "$mod" && go test ./... -count=1 -coverprofile="$profile"); then
    FAILED=1
  fi
done

if [ "$FAILED" -ne 0 ]; then
  echo "coverage.sh: one or more modules failed tests" >&2
  exit 1
fi

{
  echo "mode: set"
  for p in "$WORK"/profile_*.out; do
    [ -s "$p" ] && grep -h -v '^mode:' "$p"
  done
} > "$WORK/merged.out"

{ grep -v -E 'github\.com/useinsider/go-pkg/(insredis/redis_mock\.go|insrequester/v2/requester_mock\.go|insrequester/v3/requester_mock\.go|inskinesis/kinesis_mock\.go|inskinesis/inskinesis_mock\.go|inssqs/sqs/sqs_mock\.go|inssqs/inssqs_mock\.go):' \
  "$WORK/merged.out" || true; } \
  | sed -E 's#(github\.com/useinsider/go-pkg/insrequester)/v2/#\1/#' > "$OUT"

awk 'NR > 1 { total += $2; if ($3 > 0) covered += $2 } END {
  if (total == 0) { print "no statements found"; exit 1 }
  printf "statement-weighted coverage: %d/%d = %.1f%%\n", covered, total, (covered / total) * 100
}' "$OUT"
