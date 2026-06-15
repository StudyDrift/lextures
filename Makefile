.PHONY: dev e2e e2e-run e2e-teardown lighthouse-dashboard-dark lint lint-server lint-web lint-cli lint-www server web cli www

# Lint all apps, or pass one or more app names: `make lint server`, `make lint web www`.
LINT_APPS := server web cli www

lint:
	@apps="$(filter $(LINT_APPS),$(MAKECMDGOALS))"; \
	if [ -z "$$apps" ]; then apps="$(LINT_APPS)"; fi; \
	for app in $$apps; do \
		echo "==> lint $$app"; \
		$(MAKE) lint-$$app || exit 1; \
	done

# Swallow app names when passed as goals alongside `lint` (e.g. `make lint server`).
server web www:
	@:

lint-server:
	$(MAKE) -C server lint

lint-web:
	cd clients/web && npm run lint

lint-cli:
	cd clients/cli && golangci-lint run ./...

lint-www:
	cd www && npm run lint

# Start the development stack (Postgres, RabbitMQ, Air-backed API, Vite) in detached mode.
dev:
	docker compose -f docker-compose.yml -f docker-compose.dev.yml up --build -d

# Run the full e2e suite — automatically picks a strategy:
#
#   Docker running  →  e2e/scripts/e2e-docker.sh
#                      Ephemeral Postgres on tmpfs inside Docker.  All data gone after `down -v`.
#
#   Docker absent   →  e2e/scripts/e2e-local.sh
#                      Ephemeral Postgres cluster in a temp directory via system PG binaries
#                      (brew install postgresql@16 / apt install postgresql).  Go server via
#                      `go run`, Vite dev server for the web client.  Everything cleaned up on exit.
#
# Force a strategy: E2E_USE_DOCKER=1 (always Docker) or E2E_USE_DOCKER=0 (always local).
# GitHub Actions CI sets E2E_USE_DOCKER=0 and uses e2e/scripts/e2e-local.sh with a Postgres service.
#
# Why not SQLite?
#   The server uses jackc/pgx v5 with 653 call sites across 73+ files, plus 140+ migration
#   files that use PostgreSQL-specific syntax (JSONB, advisory locks, uuid_generate_v4,
#   pg schemas, etc.).  Both strategies above achieve "zero data persists after the run"
#   without modifying the server or rewriting all migrations.
e2e:
	@if [ "$${E2E_USE_DOCKER:-}" = "1" ]; then \
	    bash e2e/scripts/e2e-docker.sh; \
	elif [ "$${E2E_USE_DOCKER:-}" = "0" ]; then \
	    bash e2e/scripts/e2e-local.sh; \
	elif docker info > /dev/null 2>&1; then \
	    echo "==> Docker detected."; \
	    bash e2e/scripts/e2e-docker.sh; \
	else \
	    echo "==> Docker not running — switching to local Postgres stack."; \
	    bash e2e/scripts/e2e-local.sh; \
	fi

# Run Playwright tests against an already-running stack (no service management).
# Useful during active development — start the app once and iterate on tests quickly.
# Override base URL / API URL with E2E_BASE_URL / E2E_API_URL if needed.
e2e-run:
	cd e2e && npm ci --prefer-offline --quiet && npx playwright install --with-deps chromium && npx playwright test

# Force-remove the Docker e2e stack and ephemeral volumes.
e2e-teardown:
	docker compose -f docker-compose.e2e.yml down -v

# Run Lighthouse on the signed-in global dashboard in dark mode (LH.1).
# Requires API + web client already running (e.g. `make dev`).
lighthouse-dashboard-dark:
	cd e2e && npm run lighthouse:dashboard:dark

cli:
ifneq ($(filter lint,$(MAKECMDGOALS)),)
	@:
else
	cd clients/cli && go build -o lextures main.go && mkdir -p ~/.local/bin && mv lextures ~/.local/bin/lextures
endif
