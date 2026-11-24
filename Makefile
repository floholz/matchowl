.PHONY: dev-reset dev-snapshot help install dev-frontend dev-admin dev-backend build-frontend build-admin build run docker clean test tailscale tailscale-off design-pack design-unpack design-status design-hooks

# Load .env (if present) and export its variables to every recipe, so
# `make run` / `make dev-backend` see the same config as docker-compose.
-include .env
export

# Port the local app listens on (matches `make run` / `make dev-backend`).
TS_PORT ?= 8090

help: ## Show this help
	@grep -hE '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  %-16s %s\n", $$1, $$2}'

install: ## Install frontend + admin deps
	cd frontend && npm install
	cd admin && npm install

dev-frontend: ## Run the app's SvelteKit dev server on :5173 (proxies /api to :8090)
	cd frontend && npm run dev

dev-admin: ## Run the admin app's dev server on :5174 (proxies /api to :8090)
	cd admin && npm run dev

dev-backend: ## Run PocketBase backend on :8090
	go run . serve --http=127.0.0.1:8090 --dir=./pb_data

build-frontend: ## Build the app SPA into internal/web/build (cleaned first)
	rm -rf internal/web/build && mkdir -p internal/web/build
	cd frontend && npm run build
	touch internal/web/build/.gitkeep

build-admin: ## Build the admin SPA into internal/web/admin (cleaned first)
	rm -rf internal/web/admin && mkdir -p internal/web/admin
	cd admin && npm run build
	touch internal/web/admin/.gitkeep

# Version stamped into the binary (Home footer, /api/appconfig): the release
# tag when on one, else the nearest tag + commit, "-dirty" with local changes.
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS = -s -w -X github.com/floholz/matchowl/internal/version.Version=$(VERSION)

build: build-frontend build-admin ## Build the single binary (both SPAs embedded)
	CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)" -o matchowl .

run: build ## Build then run the single binary
	./matchowl serve --http=127.0.0.1:8090 --dir=./pb_data

test: ## Run Go tests
	go test ./...

tailscale: ## Expose the local app over Tailscale HTTPS for mobile dev (needs the app running, e.g. `make run`)
	@command -v tailscale >/dev/null || { echo "tailscale not found — install it first"; exit 1; }
	@# serve needs operator/root. Tries as-is, falls back to sudo. To drop the
	@# sudo prompt for good, run once: sudo tailscale set --operator=$$USER
	tailscale serve --bg $(TS_PORT) || sudo tailscale serve --bg $(TS_PORT)
	@echo
	@echo "Serving 127.0.0.1:$(TS_PORT) over HTTPS on your tailnet. Open this on your phone:"
	@tailscale serve status
	@echo "Stop with: make tailscale-off"

tailscale-off: ## Stop exposing the local app over Tailscale
	tailscale serve reset || sudo tailscale serve reset
	@echo "Tailscale serve cleared."

docker: ## Build the production Docker image
	docker build -t matchowl:latest .

reset: ## Wipe the local dev database (pb_data is disposable — re-seeded on boot)
	rm -rf pb_data
	@echo "pb_data removed — next 'make run'/'make dev-backend' re-seeds a fresh DB."

dev-reset: ## Wipe pb_data and fill it with the dev seed (sports snapshot + demo people, password matchowl-dev)
	rm -rf pb_data
	MATCHOWL_DEV=1 go run . dev-seed --dir=./pb_data --dev=false

dev-snapshot: ## Refresh the dev seed's sports snapshot from a COPY of a real database: make dev-snapshot FROM=path/to/pb_data
	@test -n "$(FROM)" || { echo "usage: make dev-snapshot FROM=path/to/a/copy/of/pb_data"; exit 1; }
	MATCHOWL_DEV=1 go run . dev-snapshot --dir=$(FROM) --dev=false

clean: ## Remove build artifacts (keeps the embed .gitkeep so go build works)
	rm -f matchowl
	rm -rf frontend/.svelte-kit frontend/build admin/.svelte-kit admin/build
	find internal/web/build -mindepth 1 ! -name .gitkeep -exec rm -rf {} + 2>/dev/null || true
	find internal/web/admin -mindepth 1 ! -name .gitkeep -exec rm -rf {} + 2>/dev/null || true

design-pack: ## Build design/matchowl.figcraft from design/matchowl/ (open it in FigCraft)
	python3 design/tools/figcraft_sync.py pack

design-unpack: ## Write FigCraft edits from design/matchowl.figcraft back to design/matchowl/
	python3 design/tools/figcraft_sync.py unpack

design-status: ## Show whether the .figcraft and design/matchowl/ are in sync
	@python3 design/tools/figcraft_sync.py status || true

design-hooks: ## Enable the git hooks that keep the .figcraft in sync (once per clone)
	git config core.hooksPath .githooks
	@echo "Hooks enabled. Run 'make design-pack' once if design/matchowl.figcraft is missing."
