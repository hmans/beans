#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Create a temp directory for test beans data
BEANS_E2E_PATH=$(mktemp -d)
export BEANS_E2E_PATH

echo "Using temp beans path: $BEANS_E2E_PATH"

# Initialize the beans directory
mise exec -- go run "$PROJECT_ROOT/cmd/beans" --beans-path "$BEANS_E2E_PATH" init >/dev/null 2>&1

# Make sure we have an embedded frontend built
if [ ! -f "$PROJECT_ROOT/internal/web/dist/index.html" ]; then
	echo "Building frontend..."
	mise run build:embed
fi

cleanup() {
	rm -rf "$BEANS_E2E_PATH"
	echo "Cleaned up $BEANS_E2E_PATH"
}
trap cleanup EXIT

cd "$SCRIPT_DIR/.."

# Run Playwright tests
npx playwright test "$@"
