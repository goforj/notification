#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
export GOWORK=off
while IFS= read -r module; do
  case "${1:-race}" in
    race) go -C "$module" test -count=1 -race ./... ;;
    vet) go -C "$module" vet ./... ;;
    tidy) go -C "$module" mod tidy ;;
    *) exit 2 ;;
  esac
done < scripts/module-manifest.txt
