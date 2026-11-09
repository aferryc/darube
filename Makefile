# Darube Makefile
# ─────────────────────────────────────────────────────────────────────────────

.PHONY: help dev test test-fe test-be build-engine build-engine-win build-engine-linux build build-win build-linux electron-build

# Extra electron-builder options, for example --publish never in CI.
ELECTRON_BUILDER_ARGS ?=

# Default target
help:
	@echo ""
	@echo "  Darube – available targets"
	@echo "  ──────────────────────────────────────────"
	@echo "  make dev            Start frontend in dev mode (Vite)"
	@echo "  make test-fe        Run frontend tests (Vitest)"
	@echo "  make test-be        Run backend tests (go test)"
	@echo "  make test           Run both FE and BE tests"
	@echo "  make build-engine   Compile the Go engine binary"
	@echo "  make build          Compile engine + package full Electron app"
	@echo "  make build-win      Build Windows package (engine + Electron)"
	@echo "  make build-linux    Build Linux package (engine + Electron)"
	@echo "  make clean          Remove compiled artifacts"
	@echo ""

# ── Frontend dev server ──────────────────────────────────────────────────────
dev:
	npm run dev

# ── Frontend tests (Vitest) ──────────────────────────────────────────────────
test-fe:
	npm run test:run

# ── Backend tests (Go) ───────────────────────────────────────────────────────
test-be:
	cd engine && go test ./... -v

# ── Run both test suites sequentially ────────────────────────────────────────
test: test-fe test-be

# ── Build the Go engine binary ───────────────────────────────────────────────
build-engine:
	cd engine && CGO_ENABLED=0 go build -o bin/engine .

# ── Build the Go engine binary for Windows ───────────────────────────────────
build-engine-win:
	cd engine && CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o bin/engine.exe .

# ── Build the Go engine binary for Linux ─────────────────────────────────────
build-engine-linux:
	cd engine && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/engine .

# ── Full production build (engine → Vite → electron-builder) ─────────────────
# electron-builder copies engine/bin via extraResources so it must exist first.
build: build-engine
	npm run build -- $(ELECTRON_BUILDER_ARGS)
	@echo ""
	@echo "  ✅ App packaged to ./release/"
	@echo ""

# ── Windows build ────────────────────────────────────────────────────────────
build-win: build-engine-win
	npm run build -- --win $(ELECTRON_BUILDER_ARGS)

# ── Linux build ──────────────────────────────────────────────────────────────
build-linux: build-engine-linux
	npm run build -- --linux $(ELECTRON_BUILDER_ARGS)

# ── Remove compiled artifacts ────────────────────────────────────────────────
clean:
	rm -f engine/engine
	rm -rf dist release
