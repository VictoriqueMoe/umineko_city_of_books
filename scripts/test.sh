#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

./scripts/regen_mocks.sh

echo "Running go vet..."
go vet ./...

if ! go tool staticcheck --version >/dev/null 2>&1; then
    echo "staticcheck is not installed as a go tool. Installing..."
    go get -tool honnef.co/go/tools/cmd/staticcheck@latest
fi

echo "Running staticcheck..."
go tool staticcheck ./...

echo "Running tests..."
go test ./... -count=1 "$@"
