#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
mkdir -p .artifacts
export GOWORK=off
go -C docs run ./report -fingerprint > .artifacts/integration-source.txt
if ! go -C integration test -tags=integration -count=1 -json ./... > .artifacts/integration.json; then
  cat .artifacts/integration.json
  exit 1
fi
go -C docs run ./report -integration "$PWD/.artifacts/integration.json"
