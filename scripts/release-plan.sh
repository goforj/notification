#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
version="${1:-}"
if [[ ! "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo 'usage: scripts/release-plan.sh v0.1.0' >&2
  exit 2
fi
printf 'Root module: %s\n' "$version"
while IFS= read -r module; do
  if [[ "$module" == driver/* ]]; then
    printf 'Optional module: %s/%s\n' "$module" "$version"
  fi
done < scripts/module-manifest.txt
echo 'Verify the pushed commit with make verify-consumer REVISION=<commit> before tagging the root.'
echo 'Publish the root first, then pin optional modules to that published version, preserve development replacements, and verify the next pushed commit before tagging them.'
echo 'After tags are published, run make verify-consumer REVISION= followed by the selected version.'
echo 'This command plans tags only. It does not tag or publish.'
