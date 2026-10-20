#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

gosec_excluded_everywhere="G115,G118"

gosec_excluded_by_path=(
    'internal[/\\]media[/\\].*:G204,G301'
    'internal[/\\]storage[/\\]engines[/\\]local\.go:G301,G302'
    'internal[/\\]upload[/\\]service\.go:G301'
    'internal[/\\]game[/\\]minesweeper[/\\]handler\.go:G404'
    'internal[/\\]game[/\\]pong[/\\]handler\.go:G404'
    'scripts[/\\]archgraph[/\\].*:G204,G301,G306'
)

run_govulncheck() {
    echo "Running govulncheck..."
    go tool govulncheck ./...
}

run_gosec() {
    local rules
    rules="$(IFS=';'; echo "${gosec_excluded_by_path[*]}")"

    echo "Running gosec..."
    go tool gosec -exclude-generated -exclude="$gosec_excluded_everywhere" -exclude-rules="$rules" ./...
}

run_gitleaks() {
    echo "Running gitleaks..."
    go tool gitleaks git --redact -v
}

checks=("$@")
if [ "${#checks[@]}" -eq 0 ]; then
    checks=(govulncheck gosec gitleaks)
fi

for check in "${checks[@]}"; do
    case "$check" in
        govulncheck | gosec | gitleaks)
            "run_${check}"
            ;;
        *)
            echo "unknown check: ${check} (expected govulncheck, gosec or gitleaks)" >&2
            exit 2
            ;;
    esac
done
