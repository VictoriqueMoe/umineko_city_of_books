#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

export CGO_ENABLED=0

if ! go tool sqlc version >/dev/null 2>&1; then
    echo "sqlc is not installed as a go tool. Installing..."
    go get -tool github.com/sqlc-dev/sqlc/cmd/sqlc@latest
fi

echo "Expanding query templates..."
go run ./internal/dao/queries/expand

echo "Regenerating sqlc code..."
rm -rf internal/dao/sqlcgen
go tool sqlc generate
echo "Done."
