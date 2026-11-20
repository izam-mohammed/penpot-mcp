#!/bin/sh
# Install the latest penpot-mcp release from GitHub Releases.
#   curl -fsSL https://raw.githubusercontent.com/izam-mohammed/penpot-mcp/main/scripts/install.sh | sh
# Env: PENPOT_MCP_VERSION (e.g. 0.2.0, default latest), PENPOT_MCP_INSTALL_DIR
# (default ~/.local/bin), PENPOT_MCP_DRY_RUN=1 (resolve and print, change nothing).
# Needs curl, tar and sha256sum or shasum. Windows: download the zip by hand.
set -eu

repo=izam-mohammed/penpot-mcp
dir=${PENPOT_MCP_INSTALL_DIR:-$HOME/.local/bin}

die() { printf 'install: %s\n' "$*" >&2; exit 1; }

case $(uname -s) in
Linux) os=linux ;;
Darwin) os=darwin ;;
*) die "unsupported os $(uname -s), grab the binary from https://github.com/$repo/releases" ;;
esac
case $(uname -m) in
x86_64 | amd64) arch=amd64 ;;
aarch64 | arm64) arch=arm64 ;;
*) die "unsupported cpu $(uname -m)" ;;
esac

version=${PENPOT_MCP_VERSION:-}
if [ -z "$version" ]; then
	url=$(curl -fsSL -o /dev/null -w '%{url_effective}' "https://github.com/$repo/releases/latest") ||
		die "could not find the latest release"
	version=${url##*/}
fi
version=${version#v}
[ -n "$version" ] || die "empty version"

name=penpot-mcp_${version}_${os}_${arch}.tar.gz
base=https://github.com/$repo/releases/download/v$version

if [ "${PENPOT_MCP_DRY_RUN:-}" = 1 ]; then
	printf 'would download %s/%s\nwould install to %s/penpot-mcp\n' "$base" "$name" "$dir"
	exit 0
fi

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT INT TERM

curl -fsSL -o "$tmp/$name" "$base/$name" || die "download failed: $base/$name"
curl -fsSL -o "$tmp/checksums.txt" "$base/checksums.txt" || die "checksums.txt download failed"

want=$(awk -v f="$name" '$2 == f { print $1 }' "$tmp/checksums.txt")
[ -n "$want" ] || die "$name not in checksums.txt"
if command -v sha256sum >/dev/null 2>&1; then
	got=$(sha256sum "$tmp/$name" | awk '{ print $1 }')
elif command -v shasum >/dev/null 2>&1; then
	got=$(shasum -a 256 "$tmp/$name" | awk '{ print $1 }')
else
	die "need sha256sum or shasum to verify the download"
fi
[ "$got" = "$want" ] || die "checksum mismatch for $name, not installing"

tar -xzf "$tmp/$name" -C "$tmp" penpot-mcp
mkdir -p "$dir"
mv "$tmp/penpot-mcp" "$dir/penpot-mcp"
chmod +x "$dir/penpot-mcp"
printf 'installed penpot-mcp %s to %s/penpot-mcp\n' "$version" "$dir"

case ":$PATH:" in
*":$dir:"*) ;;
*) printf 'note: %s is not on your PATH, add it\n' "$dir" ;;
esac
