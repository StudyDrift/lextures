# AGENTS.md

## Cursor Cloud specific instructions

### Architecture

Lextures is an LMS (Learning Management System) with two main services:

- **Go API** (`server/`): Go 1.25, Chi router, pgx for PostgreSQL, JWT auth. Runs on port 8080.
- **React SPA** (`clients/web/`): React 19, Vite 8, TypeScript 6, Tailwind CSS v4. Runs on port 5173.
- **PostgreSQL 16**: Primary data store (Docker container, port 5432). Credentials: `studydrift/studydrift`, database `studydrift`.

### Starting services

1. **Database**: `docker compose -f docker-compose.yml up -d postgres` (from repo root)
2. **RabbitMQ**: `docker compose -f docker-compose.yml up -d rabbitmq` (Canvas import queue; management UI http://localhost:15672)
3. **Go API**: `cd server && go run ./cmd/server` (requires env vars below)
4. **Web frontend**: `cd clients/web && npm run dev -- --host 0.0.0.0 --port 5173`

Required env vars for the Go API (copy from `server/.env.example` to `server/.env`):
- `DATABASE_URL=postgres://studydrift:studydrift@localhost:5432/studydrift?sslmode=disable`
- `JWT_SECRET=change-me-use-at-least-32-characters-for-production`
- `BOOTSTRAP_ADMIN_EMAIL` — optional; if set to your email, the **first** password signup on an empty human user table gets Global Admin. If unset, use `cd server && go run ./cmd/bootstrap-admin -email=you@example.com` after creating an account.
- `RUN_MIGRATIONS=true`
- `PORT=8080`
- `PUBLIC_WEB_ORIGIN=http://localhost:5173`
- `COURSE_FILES_ROOT=data/course-files`
- `RABBITMQ_URL=amqp://studydrift:studydrift@localhost:5672/` (optional locally — falls back to in-process queue when unset)

Frontend env: `VITE_API_URL=http://localhost:8080` (set when running `npm run dev`). Feature flags are loaded at runtime from `GET /api/v1/platform/features` (backed by Settings → Global platform), not `VITE_FEATURE_*` build vars.

### Commands reference

| Task | Command | Working Directory |
|------|---------|-------------------|
| Go build | `go build -o bin/server ./cmd/server` | `server/` |
| Grant Global Admin (CLI) | `go run ./cmd/bootstrap-admin -email=user@example.com` | `server/` (needs `DATABASE_URL`) |
| Go test (short, no DB) | `go test ./... -count=1 -short -timeout=1m` | `server/` |
| Go test (full, needs DB) | `make test` (needs `DATABASE_URL`) | `server/` |
| Go lint | `golangci-lint run ./...` | `server/` |
| Frontend lint | `npm run lint` (oxlint) | `clients/web/` |
| Marketing site lint | `npm run lint` (oxlint) | `www/` |
| Frontend typecheck | `npm run typecheck` | `clients/web/` |
| Frontend tests | `npm run test` | `clients/web/` |
| Frontend dev server | `npm run dev` | `clients/web/` |
| Lighthouse (dashboard, dark) | `npm run lighthouse:dashboard:dark` | `clients/web/` or `e2e/` (stack must be running) |
| E2E suite | `make e2e` | repo root |
| E2E (stack already up) | `make e2e-run` | repo root |

### Lighthouse harness (LH.1)

Reproducible Lighthouse audits for the signed-in global dashboard:

1. Start the stack (`make dev` or e2e-local).
2. Run `npm run lighthouse:dashboard:dark` from `clients/web/` or `e2e/`.
3. Inspect `docs/lighthouse/global-dashboard-darkmode.json`.

If the report shows `NO_FCP`, verify the auth token (`LH_TOKEN`), API availability, and that the browser is not backgrounded. Use `LH_REQUIRE_AUTH=1` without `LH_TOKEN` to confirm the harness fails fast instead of producing an invalid report.

### Gotchas

- The Go project uses Go 1.25, which requires a recent Go installation (not the Ubuntu default 1.22).
- `golangci-lint` must be built with Go >= 1.25 to lint this project. Use the latest version.
- The password-signup endpoint enforces HIBP (Have I Been Pwned) breach checking. Use long, random passwords for test accounts.
- The pre-commit hook (`.husky/pre-commit`) runs `lint-staged` (ESLint fix) and `tsc -b` from `clients/web/`. This runs automatically on commit if husky is installed.
- Docker daemon must be started manually before using `docker compose` (dependencies Postgres + RabbitMQ run as containers). `dockerd` needs root and a writable log path, so start it with `sudo bash -c 'dockerd >/var/log/dockerd.log 2>&1 &'` and wait ~10s for `docker info` to succeed.
- The `ubuntu` user is added to the `docker` group during setup, so plain `docker`/`docker compose` work in fresh login shells once the daemon is up; the shell that performed the `usermod` still needs `sg docker -c '<cmd>'` (or `sudo`) until a new login picks up the group.
- The fuse-overlayfs storage driver and iptables-legacy are required for Docker-in-Docker in this environment.
- Go 1.25 lives at `/usr/local/go/bin` (the system default `go` is 1.22). Setup appends this to `~/.bashrc`; if `go version` shows 1.22, prepend `/usr/local/go/bin` to `PATH`.
- Run the Go API and web SPA natively (`go run ./cmd/server`, `npm run dev`); only Postgres and RabbitMQ need Docker. The API reads `server/.env` (copy from `server/.env.example`) — it is gitignored, so recreate it if missing.
