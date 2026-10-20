#!/bin/sh
# Prove the MCP route of plans/06-mcp-route.md: initialize a session with the
# official Penpot MCP server, list its tools and print PASS when
# execute_code is there. docs/external/penpot-official-mcp.md. The URL
# holds the user's MCP key, so it reaches curl through a config file on
# stdin, never its command line, and is never printed.
set -eu

[ -n "${PENPOT_MCP_URL:-}" ] || { echo "FAIL: PENPOT_MCP_URL is not set (see .env.example)" >&2; exit 2; }
dir=$(mktemp -d)
trap 'rm -rf "$dir"' EXIT

post() { # body, extra header (optional)
	printf 'url = "%s"\n' "$PENPOT_MCP_URL" | curl -sS -K - -D "$dir/headers" -o "$dir/body" -w '%{http_code}' -X POST \
		-H 'Content-Type: application/json' -H 'Accept: application/json, text/event-stream' \
		-H 'User-Agent: penpot-mcp/prove' ${2:+-H "$2"} --data "$1"
}

status=$(post '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"penpot-mcp-prove","version":"0"}}}')
[ "$status" = 200 ] || { echo "FAIL: initialize HTTP $status"; exit 1; }
sid=$(tr -d '\r' < "$dir/headers" | sed -n 's/^[Mm]cp-[Ss]ession-[Ii]d: *//p' | head -n 1)
server=$(sed -n 's/.*"serverInfo":{\([^}]*\)}.*/\1/p' "$dir/body" | head -n 1)

post '{"jsonrpc":"2.0","method":"notifications/initialized"}' "${sid:+Mcp-Session-Id: $sid}" >/dev/null
status=$(post '{"jsonrpc":"2.0","id":2,"method":"tools/list"}' "${sid:+Mcp-Session-Id: $sid}")
[ "$status" = 200 ] || { echo "FAIL: tools/list HTTP $status"; exit 1; }
tools=$(grep -o '"name":"[a-z_]*"' "$dir/body" | sed 's/"name":"\(.*\)"/\1/' | sort -u | tr '\n' ' ')
case " $tools " in
*" execute_code "*) echo "PASS: $server offers $tools" ;;
*) echo "FAIL: execute_code is not among: $tools"; exit 1 ;;
esac
