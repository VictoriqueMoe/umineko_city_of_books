#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"

if ! command -v k6 >/dev/null 2>&1; then
    echo "k6 is not installed. See https://grafana.com/docs/k6/latest/set-up/install-k6/" >&2
    exit 1
fi

export BASE_URL="${BASE_URL:-http://localhost:4323}"
export INTERNAL_URL="${INTERNAL_URL:-http://localhost:9464}"
export LOADTEST_USER="${LOADTEST_USER:-}"
export LOADTEST_PASS="${LOADTEST_PASS:-}"

export K6_WEB_DASHBOARD="${K6_WEB_DASHBOARD:-true}"
export K6_WEB_DASHBOARD_OPEN="${K6_WEB_DASHBOARD_OPEN:-false}"

if ! curl -fs -o /dev/null --max-time 5 "${BASE_URL}/livez"; then
    echo "No server responding at ${BASE_URL}/livez" >&2
    echo "Start it with 'go run .' (binds :4323) or 'docker compose up' (host :2312)." >&2
    exit 1
fi

script="read-endpoints.js"
if [ "${SCENARIO:-read}" = "users" ]; then
    if [ ! -f users.json ]; then
        echo "No scripts/loadtest/users.json. Seed the load test database first: go run ./scripts/loadtest/seed -users ${USERS:-200}" >&2
        exit 1
    fi

    export USERS="${USERS:-200}"
    export RAMP="${RAMP:-2m}"
    export HOLD="${HOLD:-5m}"
    script="user-behaviour.js"
fi

echo "Load testing ${BASE_URL} with ${script}..."
exec k6 run "$@" "${script}"
