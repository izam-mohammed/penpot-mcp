#!/bin/sh
# usage: cover-report.sh <go test output> <cover profile> <floor file>
# Prints each package's coverage lowest first, then the total, and exits 1
# when the total is under the number in the floor file.
set -eu
log=$1
prof=$2
floorfile=$3
echo "coverage per package, lowest first:"
sed -n 's/^[a-z]*[[:space:]]*\([^[:space:]]*\)[[:space:]].*coverage: \([0-9.]*\)% of statements.*/\2 \1/p' "$log" |
	sort -n | awk '{printf "  %5.1f%%  %s\n", $1, $2}'
total=$(go tool cover -func="$prof" | awk '/^total:/ {sub(/%/, "", $3); print $3}')
floor=$(tr -d '[:space:]' <"$floorfile")
echo "total ${total}% (floor ${floor}%, target 95%)"
if awk -v t="$total" -v f="$floor" 'BEGIN { exit !(t+0 < f+0) }'; then
	echo "coverage ${total}% is under the floor ${floor}%" >&2
	exit 1
fi
