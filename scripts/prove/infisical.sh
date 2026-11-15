#!/bin/sh
# Prove the repository is linked to Infisical and every name the app needs
# is present in an environment. Prints names only, never a value.
# Names come from .env.example; list extra required names (not in
# .env.example) in EXTRA, and names the app sets itself in OPTIONAL.
# Usage: sh scripts/prove/infisical.sh [env]   (default: dev)
# Needs: infisical (logged in), jq.
set -eu
cd "$(dirname "$0")/../.."
env_slug=${1:-dev}
export INFISICAL_DISABLE_UPDATE_CHECK=true

EXTRA=""
OPTIONAL=""

command -v infisical >/dev/null || { echo "FAIL: infisical CLI not installed"; exit 2; }
command -v jq >/dev/null || { echo "FAIL: jq not installed"; exit 2; }
{ [ -f .infisical.yaml ] || [ -f .infisical.json ]; } ||
	{ echo "FAIL: not linked (no .infisical.yaml or .infisical.json); run infisical init"; exit 2; }
if [ -f .infisical.yaml ] && [ -f .infisical.json ]; then
	y=$(sed -nE 's/^ *project-id: *//p' .infisical.yaml)
	j=$(jq -r .workspaceId .infisical.json)
	[ "$y" = "$j" ] || { echo "FAIL: .infisical.yaml ($y) and .infisical.json ($j) name different projects"; exit 1; }
fi

want=$( { sed -nE 's/^#? ?([A-Z][A-Z0-9_]*)=.*/\1/p' .env.example; for n in $EXTRA; do echo "$n"; done; } | sort -u)

# Names only: jq keeps secretKey and drops every value before anything prints.
if ! have=$(infisical secrets-manager secrets get --env="$env_slug" --silent -o json 2>/dev/null |
	jq -r '.[].secretKey' | sort -u); then
	echo "FAIL: could not read $env_slug (run: infisical login status)"
	exit 1
fi

missing=
for n in $want; do
	case " $OPTIONAL " in *" $n "*) continue ;; esac
	printf '%s\n' "$have" | grep -qx "$n" || missing="$missing $n"
done
[ -z "$missing" ] || { echo "FAIL: missing in $env_slug:$missing"; exit 1; }
echo "PASS: linked, $(printf '%s\n' "$have" | grep -c .) secrets in $env_slug, every needed name present"
