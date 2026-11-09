#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
version="${1:-}"
if [[ ! "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo 'usage: scripts/release-plan.sh v0.1.0' >&2
  exit 2
fi
printf 'Root module: %s\n' "$version"
for adapter in mail apns sns webpush; do
  printf 'Optional module: driver/%s/%s\n' "$adapter" "$version"
done
echo 'Publish the root first, update each optional module requirement, then verify all five with GOWORK=off and no replacements.'
echo 'This command plans tags only. It does not tag or publish.'
