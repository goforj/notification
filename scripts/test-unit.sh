#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
mkdir -p .artifacts
rm -f .artifacts/unit-[0-9]*.out .artifacts/unit-[0-9]*.json
rm -rf .artifacts/coverdata
mkdir .artifacts/coverdata
export GOWORK=off
go -C docs run ./report -fingerprint > .artifacts/unit-source.txt
export NOTIFICATION_EXAMPLE_COVERAGE="$PWD/.artifacts/coverdata"
index=0
while IFS= read -r module; do
  if [[ "$module" == "docs" ]]; then
    if ! go -C docs test -count=1 -json ./... > ".artifacts/unit-$index.json"; then
      cat ".artifacts/unit-$index.json"
      exit 1
    fi
    index=$((index+1))
    continue
  fi
  if ! go -C "$module" test -count=1 -json -covermode=atomic -coverpkg=github.com/goforj/notification/... -coverprofile="$PWD/.artifacts/unit-$index.out" ./... > ".artifacts/unit-$index.json"; then
    cat ".artifacts/unit-$index.json"
    exit 1
  fi
  index=$((index+1))
done < scripts/module-manifest.txt
go tool covdata textfmt -i=.artifacts/coverdata -o=.artifacts/examples.out
go -C docs run ./report
