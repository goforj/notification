#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
mkdir -p .artifacts
rm -rf .artifacts/coverdata
mkdir .artifacts/coverdata
python3 -m unittest discover -s scripts -p '*_test.py'
export GOWORK=off
export NOTIFICATION_EXAMPLE_COVERAGE="$PWD/.artifacts/coverdata"
index=0
while IFS= read -r module; do
  if ! go -C "$module" test -count=1 -json -covermode=atomic -coverpkg=github.com/goforj/notification/... -coverprofile="$PWD/.artifacts/unit-$index.out" ./... > ".artifacts/unit-$index.json"; then
    cat ".artifacts/unit-$index.json"
    exit 1
  fi
  index=$((index+1))
done < scripts/module-manifest.txt
go tool covdata textfmt -i=.artifacts/coverdata -o=.artifacts/examples.out
python3 scripts/report-unit.py
