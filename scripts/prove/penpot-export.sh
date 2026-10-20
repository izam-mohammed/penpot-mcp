#!/bin/sh
# Prove the exporter route of plans/05-export.md: log in with email and
# password for the auth-token cookie, export the first board of the test file
# to PNG through /api/export (transit body), download it and check the PNG
# signature. Then try the share-link route without a cookie and report it.
# docs/penpot-api-reference.md section 10.12.
set -eu

for v in PENPOT_ACCESS_TOKEN PENPOT_EMAIL PENPOT_PASSWORD PENPOT_TEST_FILE_ID; do
	eval "val=\${$v:-}"
	[ -n "$val" ] || { echo "FAIL: $v is not set (see .env.example)" >&2; exit 2; }
done
base="${PENPOT_BASE_URL:-https://design.penpot.app}"
file="$PENPOT_TEST_FILE_ID"
dir=$(mktemp -d)
trap 'rm -rf "$dir"' EXIT
jar="$dir/jar"

rpc() { # command, body
	curl -sS -o "$dir/out" -w '%{http_code}' -X POST \
		-H "Authorization: Token $PENPOT_ACCESS_TOKEN" -H 'Accept: application/json' \
		-H 'Content-Type: application/json' -H 'X-Client: penpot-mcp/prove' \
		--data "$2" "$base/api/main/methods/$1"
}

# The password goes to curl on stdin, never on its command line.
status=$(printf '{"email":"%s","password":"%s"}' "$PENPOT_EMAIL" "$PENPOT_PASSWORD" |
	curl -sS -o "$dir/login" -w '%{http_code}' -c "$jar" -X POST \
		-H 'Content-Type: application/json' -H 'Accept: application/json' \
		--data-binary @- "$base/api/main/methods/login-with-password")
[ "$status" = 200 ] || { echo "FAIL: login HTTP $status"; exit 1; }
grep -q 'auth-token' "$jar" || { echo "FAIL: login set no auth-token cookie"; exit 1; }
profile=$(sed -n 's/.*"id":"\([0-9a-f-]*\)".*/\1/p' "$dir/login" | head -n 1)

rpc get-file "{\"id\":\"$file\"}" >/dev/null
page=$(sed -n 's/.*"pages":\["\([0-9a-f-]*\)".*/\1/p' "$dir/out" | head -n 1)
rpc get-page "{\"fileId\":\"$file\",\"pageId\":\"$page\"}" >/dev/null
board=$(tr ',' '\n' < "$dir/out" | sed -n 's/.*"shapes":\["\([0-9a-f-]*\)".*/\1/p' | head -n 1)
[ -n "$board" ] || { echo "FAIL: no board on the test file's first page"; exit 1; }

body="[\"^ \",\"~:cmd\",\"~:export-shapes\",\"~:profile-id\",\"~u$profile\",\"~:wait\",true,\"~:exports\",[[\"^ \",\"~:file-id\",\"~u$file\",\"~:page-id\",\"~u$page\",\"~:object-id\",\"~u$board\",\"~:type\",\"~:png\",\"~:scale\",1,\"~:suffix\",\"\",\"~:name\",\"prove\"]]]"
status=$(curl -sS -o "$dir/res" -w '%{http_code}' -b "$jar" -X POST \
	-H 'Content-Type: application/transit+json' --data "$body" "$base/api/export")
[ "$status" = 200 ] || { echo "FAIL: export HTTP $status $(head -c 200 "$dir/res")"; exit 1; }
uri=$(sed -n 's/.*"~:uri","~r\([^"]*\)".*/\1/p' "$dir/res")
[ -n "$uri" ] || { echo "FAIL: no uri in $(head -c 200 "$dir/res")"; exit 1; }
curl -sS -o "$dir/board.png" "$uri"
sig=$(head -c 8 "$dir/board.png" | od -An -tx1 | tr -d ' \n')
[ "$sig" = 89504e470d0a1a0a ] || { echo "FAIL: download is not a PNG ($sig)"; exit 1; }
echo "PASS: cookie route exported board $board ($(wc -c < "$dir/board.png" | tr -d ' ') bytes)"

# Share link: no cookie, share-id in the export item. Only reported.
rpc create-share-link "{\"fileId\":\"$file\",\"whoComment\":\"all\",\"whoInspect\":\"all\",\"pages\":[\"$page\"]}" >/dev/null
share=$(sed -n 's/.*"id":"\([0-9a-f-]*\)".*/\1/p' "$dir/out" | head -n 1)
body2=$(printf '%s' "$body" | sed "s/\"~:name\",\"prove\"/\"~:name\",\"prove\",\"~:share-id\",\"~u$share\"/")
status=$(curl -sS -o "$dir/res2" -w '%{http_code}' -X POST \
	-H 'Content-Type: application/transit+json' --data "$body2" "$base/api/export")
echo "share-link route: HTTP $status $(head -c 120 "$dir/res2")"
rpc delete-share-link "{\"id\":\"$share\"}" >/dev/null || true
