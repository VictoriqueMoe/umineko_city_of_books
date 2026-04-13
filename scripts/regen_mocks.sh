#!/usr/bin/env bash
# Regenerate all mockery mocks declared in .mockery.yml.
# Usage: ./scripts/regen_mocks.sh
set -euo pipefail

cd "$(dirname "$0")/.."

if ! go tool mockery --version >/dev/null 2>&1; then
    echo "mockery is not installed as a go tool. Installing..."
    go get -tool github.com/vektra/mockery/v3@latest
fi

echo "Regenerating mocks..."
go tool mockery
echo "Done."
