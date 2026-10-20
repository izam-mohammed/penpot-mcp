#!/bin/sh
# Check how Penpot handles moving a shape between boards: send mov-objects as
# JSON, then as transit, and report which one moved it, per
# docs/penpot-api-reference.md section 5.5. Writes to the file named in
# PENPOT_TEST_FILE_ID (two boards and a rect, left in place).
set -eu

for v in PENPOT_ACCESS_TOKEN PENPOT_TEST_FILE_ID; do
	eval "val=\${$v:-}"
	[ -n "$val" ] || { echo "FAIL: $v is not set (see .env.example)" >&2; exit 2; }
done
base="${PENPOT_BASE_URL:-https://design.penpot.app}"
file="$PENPOT_TEST_FILE_ID"
root=00000000-0000-0000-0000-000000000000
id() { uuidgen | tr 'A-Z' 'a-z'; }
tmp=$(mktemp)
trap 'rm -f "$tmp"' EXIT

rpc() { # command, content type, body
	curl -sS -o "$tmp" -w '%{http_code}' -X POST \
		-H "Authorization: Token $PENPOT_ACCESS_TOKEN" -H 'Accept: application/json' \
		-H "Content-Type: $2" -H 'X-Client: penpot-mcp/prove' \
		--data "$3" "$base/api/main/methods/$1"
}
field() { sed -n "s/.*\"$1\":\([0-9]*\).*/\1/p" "$tmp" | head -n 1; }

rpc get-file application/json "{\"id\":\"$file\"}" >/dev/null
revn=$(field revn); vern=$(field vern)
page=$(sed -n 's/.*"pages":\["\([0-9a-f-]*\)".*/\1/p' "$tmp" | head -n 1)
sid=$(id)

box() { # id type name parent x
	printf '{"id":"%s","type":"%s","name":"%s","x":%s,"y":900,"width":100,"height":100,"rotation":0,"selrect":{"x":%s,"y":900,"width":100,"height":100,"x1":%s,"y1":900,"x2":%s,"y2":1000},"points":[{"x":%s,"y":900},{"x":%s,"y":900},{"x":%s,"y":1000},{"x":%s,"y":1000}],"transform":{"a":1,"b":0,"c":0,"d":1,"e":0,"f":0},"transformInverse":{"a":1,"b":0,"c":0,"d":1,"e":0,"f":0},"parentId":"%s","frameId":"%s","fills":[],"strokes":[]%s}' \
		"$1" "$2" "$3" "$5" "$5" "$5" "$(($5 + 100))" "$5" "$(($5 + 100))" "$(($5 + 100))" "$5" "$4" "$4" "$( [ "$2" = frame ] && echo ',"shapes":[]' )"
}
a=$(id); b=$(id); c=$(id); r=$(id)
add() { printf '{"type":"add-obj","id":"%s","pageId":"%s","frameId":"%s","parentId":"%s","obj":%s}' "$1" "$page" "$2" "$2" "$3"; }
status=$(rpc update-file application/json "{\"id\":\"$file\",\"sessionId\":\"$sid\",\"revn\":$revn,\"vern\":$vern,\"changes\":[$(add "$a" $root "$(box "$a" frame "move A" $root 0)"),$(add "$b" $root "$(box "$b" frame "move B" $root 200)"),$(add "$c" $root "$(box "$c" frame "move C" $root 400)"),$(add "$r" "$a" "$(box "$r" rect "move rect" "$a" 0)")]}")
[ "$status" = 200 ] || { echo "FAIL: setup HTTP $status $(head -c 200 "$tmp")"; exit 1; }
revn=$((revn + 1))

parent_of_rect() {
	rpc get-page application/json "{\"fileId\":\"$file\",\"pageId\":\"$page\",\"objectId\":\"$r\"}" >/dev/null
	python3 -I -c 'import json,sys; print(json.load(open(sys.argv[1]))["objects"][sys.argv[2]]["parentId"])' "$tmp" "$r"
}

# 1. JSON mov-objects: rect from A to B.
status=$(rpc update-file application/json "{\"id\":\"$file\",\"sessionId\":\"$sid\",\"revn\":$revn,\"vern\":$vern,\"changes\":[{\"type\":\"mov-objects\",\"pageId\":\"$page\",\"parentId\":\"$b\",\"shapes\":[\"$r\"]}]}")
revn=$((revn + 1))
json_parent=$(parent_of_rect)
json_result="HTTP $status, rect parent after: $( [ "$json_parent" = "$b" ] && echo moved || echo 'not moved' )"

# 2. Transit mov-objects: rect to C.
body=$(printf '["^ ","~:id","~u%s","~:session-id","~u%s","~:revn",%s,"~:vern",%s,"~:changes",[["^ ","~:type","~:mov-objects","~:page-id","~u%s","~:parent-id","~u%s","~:shapes",["~u%s"]]]]' "$file" "$sid" "$revn" "$vern" "$page" "$c" "$r")
status=$(rpc update-file application/transit+json "$body")
transit_parent=$(parent_of_rect)
transit_result="HTTP $status, rect parent after: $( [ "$transit_parent" = "$c" ] && echo moved || echo 'not moved' )"

echo "json mov-objects: $json_result"
echo "transit mov-objects: $transit_result"
if [ "$transit_parent" = "$c" ] || [ "$json_parent" = "$b" ]; then
	echo "PASS: at least one encoding moves shapes"
else
	echo "FAIL: neither encoding moved the rect"
	exit 1
fi
