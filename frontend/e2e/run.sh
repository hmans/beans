#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Make sure we have an embedded frontend built
if [ ! -f "$PROJECT_ROOT/internal/web/dist/index.html" ]; then
	echo "Building frontend..."
	mise run build:embed
fi

# Pre-build both binaries so tests don't pay compilation cost
echo "Building test binaries..."
export BEANS_BINARY="$PROJECT_ROOT/beans"
export BEANS_SERVE_BINARY="$PROJECT_ROOT/beans-serve"
mise exec -- go build -o "$BEANS_BINARY" "$PROJECT_ROOT/cmd/beans"
mise exec -- go build -o "$BEANS_SERVE_BINARY" "$PROJECT_ROOT/cmd/beans-serve"

cd "$SCRIPT_DIR/.."

# Run Playwright tests
npx playwright test "$@"
