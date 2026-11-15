#!/bin/sh
# Run a command with the secrets loaded: from Infisical (`dev`) when this
# checkout is linked and the CLI is installed, from .env otherwise, then
# .env.local on top for machine-only values. Prints no value.
#   sh scripts/dev.sh make live
#   sh scripts/dev.sh sh scripts/prove/penpot-api.sh
set -eu
cd "$(dirname "$0")/.."
[ $# -gt 0 ] || { echo "usage: sh scripts/dev.sh <command>..." >&2; exit 2; }

if [ -z "${PENPOT_SECRETS_LOADED:-}" ] && { [ -f .infisical.yaml ] || [ -f .infisical.json ]; } &&
	command -v infisical >/dev/null 2>&1; then
	export PENPOT_SECRETS_LOADED=infisical INFISICAL_DISABLE_UPDATE_CHECK=true
	exec infisical run --env="${INFISICAL_ENV:-dev}" --silent -- sh "$0" "$@"
fi
if [ -z "${PENPOT_SECRETS_LOADED:-}" ] && [ -f .env ]; then
	set -a; . ./.env; set +a
fi
if [ -f .env.local ]; then
	set -a; . ./.env.local; set +a
fi
exec "$@"
