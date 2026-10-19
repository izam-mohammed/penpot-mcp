#!/bin/sh
# Prove the Penpot backend API accepts the access token: call get-profile and
# expect a non-zero profile id, per docs/external/penpot-backend-api.md.
# Read-only: changes nothing in Penpot.
set -eu

if [ -z "${PENPOT_ACCESS_TOKEN:-}" ]; then
	echo "FAIL: PENPOT_ACCESS_TOKEN is not set (see .env.example)" >&2
	exit 2
fi
base="${PENPOT_BASE_URL:-https://design.penpot.app}"

body_file=$(mktemp)
trap 'rm -f "$body_file"' EXIT

status=$(curl -sS -o "$body_file" -w '%{http_code}' -X POST \
	-H "Authorization: Token $PENPOT_ACCESS_TOKEN" \
	-H 'Accept: application/json' \
	-H 'Content-Type: application/json' \
	-H 'X-Client: penpot-mcp/prove' \
	--data '{}' \
	"$base/api/main/methods/get-profile")

id=$(sed -n 's/.*"id":"\([0-9a-f-]*\)".*/\1/p' "$body_file" | head -n 1)
zero=00000000-0000-0000-0000-000000000000

if [ "$status" = 200 ] && [ -n "$id" ] && [ "$id" != "$zero" ]; then
	echo "PASS: token accepted by get-profile at $base"
elif [ "$status" = 200 ]; then
	echo "FAIL: HTTP 200 but anonymous profile, token not accepted"
	exit 1
else
	echo "FAIL: HTTP $status $(head -c 200 "$body_file")"
	exit 1
fi
