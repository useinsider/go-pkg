#!/usr/bin/env bash
# scripts/coverage.sh — merged, mock-filtered, statement-weighted coverage for the
# whole multi-module repo (there is no root go.mod; every module is tested from
# its own directory).
#
# Output: a single merged profile (default: ./coverage.out, override with
# COVERAGE_OUT=<path>) with one 'mode: set' header, with the 7 generated mock
# files filtered out BY FILENAME. Hand-written mock helpers
# (insgorm/gorm_mock.go, inssql/sql_mock.go) are intentionally KEPT.
#
# Note the module-path quirk: the insrequester v2 module path is
# github.com/useinsider/go-pkg/insrequester/v2, so its mock appears in profiles
# as .../insrequester/v2/requester_mock.go even though the file on disk lives at
# insrequester/requester_mock.go.
#
# Exits non-zero if any module's tests fail.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="${COVERAGE_OUT:-$ROOT/coverage.out}"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

# 1. Discover modules: every directory that holds a go.mod.
MODULES="$(find "$ROOT" -name go.mod -not -path '*/vendor/*' -exec dirname {} \; | sort)"

# 2. Run tests per module, each with its own coverprofile.
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

# 3. Merge into a single profile with one 'mode: set' header.
{
  echo "mode: set"
  for p in "$WORK"/profile_*.out; do
    [ -s "$p" ] && grep -h -v '^mode:' "$p"
  done
} > "$WORK/merged.out"

# 4. Filter OUT exactly the 7 generated mock files, by filename, then normalize
#    the insrequester v2 module-path segment. The insrequester module declares
#    path .../insrequester/v2 but its source lives on disk at insrequester/
#    (there is no insrequester/v2/ dir — unlike insrequester/v3/), so the raw
#    profile keys those lines under insrequester/v2/*. Rewrite v2 -> (no vN) so
#    coverus resolves them to the real on-disk files after stripping the module
#    prefix. `|| true` keeps set -e from aborting when the inverse-match is empty
#    (degenerate all-mock/empty input); the awk guard below reports that case.
{ grep -v -E 'github\.com/useinsider/go-pkg/(insredis/redis_mock\.go|insrequester/v2/requester_mock\.go|insrequester/v3/requester_mock\.go|inskinesis/kinesis_mock\.go|inskinesis/inskinesis_mock\.go|inssqs/sqs/sqs_mock\.go|inssqs/inssqs_mock\.go):' \
  "$WORK/merged.out" || true; } \
  | sed -E 's#(github\.com/useinsider/go-pkg/insrequester)/v2/#\1/#' > "$OUT"

# 5. Statement-weighted coverage percentage.
awk 'NR > 1 { total += $2; if ($3 > 0) covered += $2 } END {
  if (total == 0) { print "no statements found"; exit 1 }
  printf "statement-weighted coverage: %d/%d = %.1f%%\n", covered, total, (covered / total) * 100
}' "$OUT"
