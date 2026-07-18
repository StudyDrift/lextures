.PHONY: dev desktop e2e e2e-run e2e-teardown e2e-coverage-check lighthouse-dashboard-dark lint lint-server lint-web lint-cli lint-www intro-course-validate marketplace-courses-validate iac-check mobile mobile-android mobile-ios mobile-lint-android mobile-test-android mobile-lint-ios mobile-build-ios-test mobile-test-ios mobile-test-ios-fast ios-xcodebuild android ios server web cli www

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
server web:
	@:

# Start the marketing site Vite dev server.
www:
ifneq ($(filter lint,$(MAKECMDGOALS)),)
	@:
else
	cd www && npm run dev
endif

# Lint and test native mobile apps, or pass a platform: `make mobile ios`, `make mobile android`.
MOBILE_APPS := android ios

mobile:
	@apps="$(filter $(MOBILE_APPS),$(MAKECMDGOALS))"; \
	if [ -z "$$apps" ]; then apps="$(MOBILE_APPS)"; fi; \
	for app in $$apps; do \
		echo "==> mobile $$app"; \
		$(MAKE) mobile-$$app || exit 1; \
	done

# Swallow platform names when passed as goals alongside `mobile`.
android ios:
	@:

mobile-android: mobile-lint-android mobile-test-android

mobile-ios: mobile-lint-ios mobile-test-ios

mobile-lint-android:
	cd clients/android && ./gradlew lint --no-daemon

mobile-test-android:
	cd clients/android && ./gradlew test --no-daemon

mobile-lint-ios:
	cd clients/ios && swiftlint lint

# iOS xcodebuild tuning (optional env overrides):
#   IOS_DERIVED_DATA_PATH=.cache/DerivedData  — reuse compile artifacts (default)
#   IOS_SIMULATOR="iPhone 17"                 — force a simulator (else booted, then 17 → 17 Pro → 16 → 15)
#   IOS_TEST_JOBS=4                           — cap concurrent compile tasks
#   IOS_TEST_PARALLEL=NO                      — disable parallel XCTest workers (lower RAM)
#   IOS_TEST_PARALLEL=2                       — cap parallel XCTest workers
#   IOS_ONLY_TESTING=LexturesTests/QuizLogicTests — run one test class
IOS_DERIVED_DATA_PATH ?= .cache/DerivedData
IOS_SIMULATOR ?=
IOS_TEST_JOBS ?=
IOS_TEST_PARALLEL ?=
IOS_ONLY_TESTING ?=

mobile-build-ios-test:
	@$(MAKE) ios-xcodebuild ACTION=build-for-testing

mobile-test-ios-fast:
	@$(MAKE) ios-xcodebuild ACTION=test-without-building

mobile-test-ios:
	@$(MAKE) ios-xcodebuild ACTION=test

ios-xcodebuild:
	@set -e; \
	dest=''; \
	if [ -n "$(IOS_SIMULATOR)" ]; then \
		dest="platform=iOS Simulator,name=$(IOS_SIMULATOR)"; \
	else \
		booted=$$(xcrun simctl list devices booted 2>/dev/null | grep -E '^\s+iPhone' | head -1 | sed -E 's/^[[:space:]]+([^()]+)[[:space:]]*\(.*/\1/' | xargs); \
		if [ -n "$$booted" ]; then \
			dest="platform=iOS Simulator,name=$$booted"; \
		else \
			for sim in "iPhone 17" "iPhone 17 Pro" "iPhone 16" "iPhone 15"; do \
				if xcrun simctl list devices available 2>/dev/null | grep -qF "    $$sim ("; then \
					dest="platform=iOS Simulator,name=$$sim"; \
					break; \
				fi; \
			done; \
		fi; \
	fi; \
	if [ -z "$$dest" ]; then \
		echo "No iOS Simulator found. Install one in Xcode → Settings → Platforms."; \
		exit 1; \
	fi; \
	mkdir -p "$(IOS_DERIVED_DATA_PATH)"; \
	parallel_args=''; \
	if [ "$(IOS_TEST_PARALLEL)" = "NO" ]; then \
		parallel_args='-parallel-testing-enabled NO'; \
	elif [ -n "$(IOS_TEST_PARALLEL)" ]; then \
		parallel_args="-maximum-parallel-testing-workers $(IOS_TEST_PARALLEL)"; \
	fi; \
	jobs_args=''; \
	if [ -n "$(IOS_TEST_JOBS)" ]; then \
		jobs_args="-jobs $(IOS_TEST_JOBS)"; \
	fi; \
	only_args=''; \
	if [ -n "$(IOS_ONLY_TESTING)" ]; then \
		only_args="-only-testing:$(IOS_ONLY_TESTING)"; \
	fi; \
	echo "==> iOS $(ACTION) ($$dest, derivedData=$(IOS_DERIVED_DATA_PATH))"; \
	cd clients/ios && xcodebuild $(ACTION) \
		-project Lextures.xcodeproj \
		-scheme Lextures \
		-destination "$$dest" \
		-configuration Debug \
		-derivedDataPath "$(CURDIR)/$(IOS_DERIVED_DATA_PATH)" \
		$$jobs_args \
		$$parallel_args \
		$$only_args \
		CODE_SIGNING_ALLOWED=NO \
		COMPILER_INDEX_STORE_ENABLE=NO \
		ONLY_ACTIVE_ARCH=YES

intro-course-validate:
	$(MAKE) -C server intro-course-validate

marketplace-courses-validate:
	$(MAKE) -C server marketplace-courses-validate

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

# Build the Tauri desktop app, install it locally, and launch it.
desktop:
	bash scripts/desktop.sh

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

# E2E.4 — completed-feature coverage gate (no browsers / no stack).
e2e-coverage-check:
	cd e2e && npm ci --prefer-offline --quiet && npm run e2e:coverage:test && npm run e2e:coverage:check && npm run e2e:coverage:report

# Run Lighthouse on the signed-in global dashboard in dark mode (LH.1).
# Requires API + web client already running (e.g. `make dev`).
lighthouse-dashboard-dark:
	cd e2e && npm run lighthouse:dashboard:dark

# Terraform fmt + validate for iac/self-aws (no cloud credentials).
iac-check:
	bash iac/scripts/terraform-check.sh

cli:
ifneq ($(filter lint,$(MAKECMDGOALS)),)
	@:
else
	cd clients/cli && go build -o lextures main.go && mkdir -p ~/.local/bin && mv lextures ~/.local/bin/lextures
endif
