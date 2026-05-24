.PHONY: dev test build clean help daemon web tui release release-snapshot install ensure-web-dist doc-quality markdownlint frontmatter audit-harness-verify

# Default target
help:
	@echo "Gastown Viewer Intent - Build Targets"
	@echo ""
	@echo "  make dev      - Run daemon + web dev server (parallel)"
	@echo "  make daemon   - Run daemon only (localhost:7070)"
	@echo "  make web      - Run web dev server only (localhost:5173)"
	@echo "  make tui      - Run TUI client"
	@echo "  make test     - Run all tests"
	@echo "  make build    - Build all binaries"
	@echo "  make clean    - Remove build artifacts"
	@echo ""

# Development - runs daemon and web in parallel
dev:
	@echo "Starting Gastown Viewer Intent..."
	@echo "  Daemon: http://localhost:7070"
	@echo "  Web UI: http://localhost:5173"
	@echo ""
	@$(MAKE) -j2 daemon web

# Ensure web_dist stub exists for go:embed (no-op if already present)
ensure-web-dist:
	@mkdir -p internal/api/web_dist
	@test -f internal/api/web_dist/index.html || echo '<!DOCTYPE html><html><body><p>Run <code>make build</code> to bundle the web UI.</p></body></html>' > internal/api/web_dist/index.html

# Run daemon
daemon: ensure-web-dist
	go run ./cmd/gvid

# Run web dev server
web:
	cd web && npm run dev

# Run TUI
tui:
	go run ./cmd/gvi-tui

# Run tests
test: ensure-web-dist
	@echo "=== Go Tests ==="
	go test -v ./...
	@echo ""
	@echo "=== Web Lint ==="
	cd web && npm run lint 2>/dev/null || echo "Web lint not configured yet"

# Build all (web first — Go embed needs web/dist/)
build:
	@echo "=== Building Web ==="
	cd web && npm run build
	@echo ""
	@echo "=== Copying web dist for embed ==="
	rm -rf internal/api/web_dist
	cp -r web/dist internal/api/web_dist
	@echo ""
	@echo "=== Building Go binaries ==="
	go build -o bin/gvid ./cmd/gvid
	go build -o bin/gvi-tui ./cmd/gvi-tui
	@echo ""
	@echo "Build complete. Binaries in ./bin/"

# Clean
clean:
	rm -rf bin/
	rm -rf dist/
	rm -rf web/dist/
	rm -rf internal/api/web_dist/
	go clean ./...

# Release with goreleaser
release:
	goreleaser release --clean

# Snapshot release (no publish)
release-snapshot:
	goreleaser release --snapshot --clean

# Install locally
install: build
	@chmod +x deploy/install.sh
	@./deploy/install.sh

# Doc-quality gates (run in CI via .github/workflows/doc-quality.yml).
# Local invocation: `make doc-quality` runs the two that have no external binary;
# Vale + lychee are CI-only by default (one-line install instructions in README).
doc-quality: markdownlint frontmatter
	@echo "doc-quality: markdownlint + frontmatter validator OK (Vale + lychee run in CI)"

markdownlint:
	@web/node_modules/.bin/markdownlint-cli2 "**/*.md" "#node_modules" "#web/node_modules"

frontmatter:
	@python3 scripts/validate-frontmatter.py 000-docs

audit-harness-verify:
	@web/node_modules/.bin/audit-harness verify
