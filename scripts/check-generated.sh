#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
first=$(mktemp)
second=$(mktemp)
trap 'rm -f "$first" "$second"' EXIT
python3 scripts/generate-docs.py --check
python3 scripts/generate-docs.py
find README.md docs/assets/badges -type f -exec shasum -a 256 {} \; | sort > "$first"
python3 scripts/generate-docs.py
find README.md docs/assets/badges -type f -exec shasum -a 256 {} \; | sort > "$second"
cmp "$first" "$second"
