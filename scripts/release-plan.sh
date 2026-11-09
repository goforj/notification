#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
version="${1:-}"
if [[ ! "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo 'usage: scripts/release-plan.sh v0.1.0' >&2
  exit 2
fi
printf 'Root module: %s\nMail adapter module: driver/mail/%s\n' "$version" "$version"
echo 'Publish the root first, update the adapter requirement, then verify both with GOWORK=off and no replacements.'
echo 'This command plans tags only. It does not tag or publish.'
