# Build, test, dev and release tasks; `just --list` shows them.
# Requires go, node and corepack on PATH; pnpm is run through corepack.
# Node 25+ no longer bundles corepack: install it with `npm install -g corepack`.
# wgo and svu are run with `go run`, so they need no install either.

set positional-arguments

# pnpm 11+ ignores `onlyBuiltDependencies` in frontend/pnpm-workspace.yaml and refuses to install.
pnpm := "corepack pnpm@10"

# Download pnpm without asking, so recipes never stop at an interactive prompt.
export COREPACK_ENABLE_DOWNLOAD_PROMPT := "0"

# Pinned here rather than as `tool` in go.mod, so they cannot change the
# dependencies of the beans binary. These are the newest versions that build
# with the Go version go.mod requires.
wgo := "go run github.com/bokwoon95/wgo@v0.6.2"
svu := "go run github.com/caarlos0/svu/v3@v3.2.4"

ldflags := "-X github.com/hmans/beans/internal/version.Version=" + `git describe --tags --always --dirty 2>/dev/null || echo dev` + " -X github.com/hmans/beans/internal/version.Commit=" + `git rev-parse --short HEAD 2>/dev/null || echo unknown` + " -X github.com/hmans/beans/internal/version.Date=" + `date -u +%Y-%m-%dT%H:%M:%SZ`

default: build

# Install dependencies and generate code
setup: deps codegen

deps: deps-backend deps-frontend

deps-backend:
    go mod download

deps-frontend:
    cd frontend && {{pnpm}} install

codegen:
    go generate ./...
    cd frontend && {{pnpm}} codegen

build-frontend:
    cd frontend && {{pnpm}} build

# Copy the frontend build into the directory embedded by the Go binary
build-embed: build-frontend
    rm -rf internal/web/dist
    mkdir -p internal/web/dist
    cp -r frontend/build/* internal/web/dist/
    touch internal/web/dist/.gitkeep

build: codegen build-embed
    go build -ldflags "{{ldflags}}" -o beans ./cmd/beans
    go build -ldflags "{{ldflags}}" -o beans-serve ./cmd/beans-serve
    go build -ldflags "{{ldflags}}" -o beans-tui ./cmd/beans-tui

install: build
    mkdir -p ~/.local/bin
    cp beans ~/.local/bin/beans
    cp beans-serve ~/.local/bin/beans-serve
    cp beans-tui ~/.local/bin/beans-tui

test: codegen test-e2e
    go test ./...

test-e2e: build-embed
    cd frontend && {{pnpm}} test:e2e

# Build and run the beans CLI, e.g. `just beans list --json`
beans *args:
    go run ./cmd/beans "$@"

beans-serve port=env("BEANS_PORT", "22880"):
    go build -o /tmp/beans-serve-dev ./cmd/beans-serve && /tmp/beans-serve-dev --port {{port}}

beans-tui *args:
    go run ./cmd/beans-tui "$@"

dev-frontend port=env("VITE_PORT", "5173"):
    cd frontend && {{pnpm}} dev --port {{port}} --strictPort

# Run beans-serve and the frontend dev server with live reload; Ctrl-C stops both
dev: setup
    #!/usr/bin/env bash
    set -uo pipefail
    # Ports: BEANS_PORT and VITE_PORT, else the workspace port that beans-serve
    # assigns to a Run session (frontend on it, so its Open link works; backend
    # one above, the allocator leaves 10 ports per workspace), else 22880/5173.
    ws=${BEANS_WORKSPACE_PORT:-}
    export BEANS_PORT=${BEANS_PORT:-${ws:+$((ws + 1))}}
    export BEANS_PORT=${BEANS_PORT:-22880}
    export VITE_PORT=${VITE_PORT:-${ws:-5173}}
    # Each job gets its own process group, so the trap stops both servers with
    # one signal each and never signals the caller's group. HUP arrives when the
    # terminal of a Run session closes.
    set -m
    trap 'trap - EXIT HUP INT TERM; kill -- -$serve -$front 2>/dev/null; wait' EXIT HUP INT TERM
    "{{just_executable()}}" dev-serve & serve=$!
    "{{just_executable()}}" dev-frontend & front=$!
    wait -n

# Rebuild and restart beans-serve when a non-test .go file changes
dev-serve port=env("BEANS_PORT", "22880"):
    {{wgo}} -file '\.go$' -xfile '_test\.go$' -xdir '^frontend$' \
        go build -o /tmp/beans-serve-dev-{{port}} ./cmd/beans-serve \
        :: /tmp/beans-serve-dev-{{port}} --port {{port}}

# Kill the dev servers of every checkout and worktree (not an installed beans-serve)
dev-kill:
    #!/usr/bin/env bash
    # A shebang recipe runs from a script file, so pkill cannot match itself.
    pkill -f 'wgo(@[^ ]*)? .*/tmp/beans-serve-dev-' && echo "Killed wgo watchers" || echo "No wgo watchers"
    pkill -f '^/tmp/beans-serve-dev(-[0-9]+)?( |$)' && echo "Killed beans-serve processes" || echo "No beans-serve processes"

# Create a patch version tag (e.g., v0.1.4 → v0.1.5)
release-patch: (_release "patch")

# Create a minor version tag (e.g., v0.1.4 → v0.2.0)
release-minor: (_release "minor")

# Create a major version tag (e.g., v0.1.4 → v1.0.0)
release-major: (_release "major")

_release bump:
    #!/usr/bin/env bash
    set -euo pipefail
    version=$({{svu}} {{bump}})
    git tag "$version"
    echo "Created tag: $version"
    echo "To release, run: git push && git push --tags"
