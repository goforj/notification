#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
snapshot="$(mktemp -d "${TMPDIR:-/tmp}/notification-generated.XXXXXX")"
trap 'rm -rf "$snapshot"' EXIT
cp README.md "$snapshot/README.md"
cp docs/advanced.md "$snapshot/advanced.md"
cp -R examples "$snapshot/examples"
cp -R docs/assets/badges "$snapshot/badges"
GOWORK=off go -C docs run ./readme -check
GOWORK=off go -C docs run ./readme
GOWORK=off go -C docs run ./readme
diff -u "$snapshot/README.md" README.md
diff -u "$snapshot/advanced.md" docs/advanced.md
diff -ru "$snapshot/examples" examples
diff -ru "$snapshot/badges" docs/assets/badges
