# Lextures — Architecture & Stack Blueprint

Use this as a reference to start a new project with a similar shape to [Lextures](https://github.com/StudyDrift/lextures): a **monorepo** with a **Go API**, **React SPA**, and **PostgreSQL**, optimized for local Docker dev and self-hosting.

---

## 1. What Lextures Is

**Lextures** is an open-source LMS (Learning Management System): courses, modules, grading, enrollments, integrations (LTI, SAML, OIDC, SCIM), optional AI via customer-chosen providers (BYOK; OpenRouter is one option). License: **AGPL-3.0**.

For a **greenfield** project you typically keep the **platform shape** (repo layout, layers, tooling) and drop domain-specific pieces (quizzes, LTI, 220+ migrations, compliance modules).

---

## 2. Monorepo Layout

```
lextures/
├── server/                 # Go API (module: github.com/lextures/lextures/server)
│   ├── cmd/
│   │   ├── server/         # Main HTTP entry
│   │   └── bootstrap-admin/  # One-off CLI tools
│   ├── migrations/         # Numbered SQL (001_*.sql … 225_*.sql)
│   ├── internal/
│   │   ├── app/            # Wiring: config, DB, migrations, HTTP server
│   │   ├── httpserver/     # Chi routes + HTTP handlers (large surface)
│   │   ├── service/        # Business logic
│   │   ├── repos/          # pgx data access
│   │   ├── models/         # Domain types
│   │   ├── auth/           # JWT, passwords (Argon2id)
│   │   ├── authz/          # RBAC permission matching
│   │   ├── apierr/         # JSON error envelope
│   │   ├── config/         # Env-based config
│   │   ├── migrate/        # Migration runner
│   │   └── background/     # Periodic jobs (goroutines today)
│   ├── Dockerfile
│   └── .env.example
├── clients/
│   ├── web/                # Primary React SPA
│   ├── cli/                # Go admin CLI
│   ├── android/            # Native mobile (optional)
│   └── ios/
├── www/                    # Marketing/docs site (separate Vite app)
├── e2e/                    # Playwright tests
├── iac/                    # Terraform modules (AWS/GCP/Azure)
├── docs/                   # ARCH.md, getting-started, ADRs
├── data/                   # Local file storage mount (course-files)
├── docker-compose.yml      # Base: postgres, server, web
├── docker-compose.dev.yml  # Dev: Vite HMR :5173
├── docker-compose.prod.yml # Prod: nginx static :3000
├── Makefile                # e2e orchestration
├── AGENTS.md               # Agent/dev environment notes
└── package.json            # Root (minimal; Playwright at repo root for e2e)
```

**Pattern:** one repo, multiple deployable artifacts; API and SPA versioned together; shared contracts evolving toward OpenAPI.

---

## 3. Tech Stack (Copy This Table)

| Layer | Choices |
|--------|---------|
| **API** | Go **1.25**, [chi v5](https://github.com/go-chi/chi), [pgx v5](https://github.com/jackc/pgx), `log/slog` |
| **Auth** | JWT access tokens (`golang-jwt/jwt/v5`), refresh tokens, Argon2id passwords, optional WebAuthn/TOTP/MFA |
| **DB** | PostgreSQL **16** (Docker Alpine image locally) |
| **Web app** | React **19**, Vite **8**, TypeScript **6**, Tailwind CSS **v4** (`@tailwindcss/vite`) |
| **Routing (web)** | React Router **v7** (`BrowserRouter` + `Routes`) |
| **Validation (web)** | Zod schemas + `parseApiResponse` helpers |
| **Rich text** | TipTap (domain-specific; optional for new apps) |
| **i18n** | i18next + react-i18next |
| **Tests (web)** | Vitest, Testing Library, MSW for mocks |
| **Tests (API)** | `go test` with Postgres service in CI |
| **E2E** | Playwright (`e2e/`, `make e2e`) |
| **Containers** | Docker Compose (dev/prod/e2e overlays) |
| **CI** | GitHub Actions: golangci-lint, go test + coverage floor, govulncheck, web lint/typecheck/test |
| **Pre-commit** | Husky + lint-staged (ESLint on `*.ts/tsx`) + `tsc -b` from `clients/web` |

**Ports (local dev):**

- Web (Vite): `5173`
- API: `8080`
- Postgres: `5432` (user/pass/db: `studydrift` / `studydrift` / `studydrift`)

---

## 4. Backend Architecture

### 4.1 Four-layer split

Documented in `docs/ARCH.md` as the intended mental model:

1. **`httpserver`** — HTTP: decode JSON, call services, write JSON/errors. Split by domain (`auth.go`, `course_*.go`, …). Chi router in `server.go`.
2. **`service`** — Business rules, orchestration, transactions.
3. **`repos`** — SQL via pgx; one package per aggregate/table group.
4. **`models`** — Structs and domain constants (often with `doc.go`).

**Entry flow:**

```
cmd/server/main.go
  → app.Run(ctx, embedded migrations FS)
    → config.Load()
    → db.NewPool(DATABASE_URL)
    → migrate.RunWithFS (if RUN_MIGRATIONS=true)
    → merge env + DB platform settings
    → filestorage.New(local or S3-compatible)
    → background.StartWithStorage(...)
    → httpserver.NewHandler(Deps) on http.Server
```

### 4.2 HTTP conventions

- **API prefix:** `/api/v1/...`
- **Health:** `GET /health`, `GET /health/ready` (DB ping)
- **OpenAPI:** `GET /api/openapi.json`, `GET /api/docs` (skeleton today; roadmap is OpenAPI-first)
- **Errors:** stable JSON shape via `apierr`:

  ```json
  { "error": { "code": "FORBIDDEN", "message": "Human-readable text." } }
  ```

- **Middleware (chi):** CORS, RequestID, RealIP, Recoverer, access logging
- **404:** explicit handler when no route matches (distinct from handler-level NOT_FOUND)

### 4.3 Auth & authorization

- **Signup/login:** `authservice` → Argon2id hash → JWT via `JWTSigner` (claims: `sub`, `email`, `org_id`, session version for revocation)
- **Password policy:** HIBP breach check on signup/password change
- **Bootstrap admin:** `BOOTSTRAP_ADMIN_EMAIL` — first human signup with matching email gets **Global Admin** (advisory lock + role assign in transaction)
- **RBAC:** permissions are four segments: `scope:area:function:action` with `*` wildcards (`authz.PermissionMatches`)
- **Handlers today:** often call `require_permission()` manually (ARCH.md recommends moving to chi middleware)

### 4.4 Configuration

- **Secrets & infra:** env vars (`server/.env` from `.env.example`)
  - Required: `DATABASE_URL`, `JWT_SECRET` (≥32 chars prod), `PORT`, `PUBLIC_WEB_ORIGIN`, `RUN_MIGRATIONS`
  - Files: `COURSE_FILES_ROOT` or S3-compatible storage vars
- **Feature flags:** stored in DB (`settings.platform_app_settings`), not `VITE_*` build flags. Web loads `GET /api/v1/platform/features` at runtime.

### 4.5 Migrations

- **~223** sequential SQL files in `server/migrations/`
- Embedded in Go module root, applied on startup when `RUN_MIGRATIONS=true`
- PostgreSQL-specific: schemas (`"user".users`), JSONB, advisory locks, `uuid`, etc. — **not portable to SQLite**
- Dev repair: `MIGRATE_REPAIR_CHECKSUMS` for idempotent migration edits

### 4.6 Background work

- `internal/background/` — periodic goroutines (e.g. quiz auto-submit, grade release every ~30s)
- Roadmap: Postgres-backed job queue (`FOR UPDATE SKIP LOCKED`)

---

## 5. Frontend Architecture

### 5.1 Bootstrap (`clients/web/src/main.tsx`)

Provider tree (simplified):

```
StrictMode
  BrowserRouter
    I18nProvider
      LocaleFormatProvider
        OrgBrandingProvider
          PermissionsProvider
            App + LmsToaster
```

PWA: `vite-plugin-pwa` with custom service worker `src/sw.ts`.

### 5.2 Routing

- Central `app.tsx` with many `<Route>` entries
- Pages under `src/pages/` (LMS under `pages/lms/`, admin, auth)
- Shared UI: `src/components/`, hooks in `src/hooks/`, contexts in `src/context/`

### 5.3 API client pattern (current)

- **`src/lib/api.ts`:** `apiBaseUrl()` from `VITE_API_URL`, `authorizedFetch` with JWT + refresh retry + 401 → clear session + `studydrift-auth-required` event
- **Per-domain modules:** `*-api.ts` (e.g. `courses-api.ts`) — hand-written fetch + **Zod** parse
- **Codegen (optional):** `npm run openapi:types` → `src/lib/generated/openapi-types.ts` from running server
- **No React Query yet** — many `fetch` + `useEffect` + `useState`; ARCH recommends TanStack Query for new work

### 5.4 Auth on the client

- Access token in `localStorage` (`studydrift_access_token`)
- Refresh token flow via `/api/v1/auth/refresh`
- Permissions context loaded after login for UI gating

### 5.5 Styling & a11y

- Tailwind v4 via Vite plugin
- ESLint: `eslint-plugin-jsx-a11y`, i18n key checks
- Scripts: contrast check, bundle size budget, locale parity

### 5.6 Testing

- Unit/component: Vitest + jsdom + Testing Library
- MSW handlers in `src/test/mocks/`
- E2E: separate `e2e/` package, `make e2e` spins ephemeral Postgres + stack

---

## 6. Docker & Local Dev

**Recommended dev:**

```bash
# .env at repo root
BOOTSTRAP_ADMIN_EMAIL=you@yourdomain.com

docker compose -f docker-compose.yml -f docker-compose.dev.yml up --build -d
# Web: http://localhost:5173  API: http://localhost:8080
```

**Without full Docker:**

```bash
docker compose -f docker-compose.yml up -d postgres
cp server/.env.example server/.env   # set DATABASE_URL, JWT_SECRET, BOOTSTRAP_ADMIN_EMAIL
cd server && go run ./cmd/server
cd clients/web && npm install && VITE_API_URL=http://localhost:8080 npm run dev
```

**Prod-style static web:**

```bash
docker compose -f docker-compose.yml -f docker-compose.prod.yml up --build
# Web nginx :3000, API :8080
```

---

## 7. CI & Quality Gates

From `.github/workflows/ci.yml` (on PRs to `main`):

| Job | What it does |
|-----|----------------|
| **server-go** | Postgres 16 service, golangci-lint v2.x (Go 1.25+), `go test ./internal/...` with coverage ≥ ~10.5%, govulncheck, build `cmd/server` |
| **client-cli** | Lint/test Go CLI |
| **web** | ESLint, `tsc -b`, Vitest, bundle checks |
| **e2e** | Playwright against compose/local Postgres |

**Local commands:**

| Task | Command | CWD |
|------|---------|-----|
| Go build | `go build -o bin/server ./cmd/server` | `server/` |
| Go test (short) | `go test ./... -short` | `server/` |
| Go lint | `golangci-lint run ./...` | `server/` |
| Web typecheck | `npm run typecheck` | `clients/web/` |
| Web test | `npm run test` | `clients/web/` |
| E2E | `make e2e` | repo root |

---

## 8. Cross-Cutting Concerns (Present in Lextures)

Worth knowing before copying — many are **optional** for a smaller greenfield app:

| Concern | Implementation |
|---------|----------------|
| **Multi-tenant / orgs** | Org branding, org units, terms, sections (schema evolving) |
| **SSO** | SAML 2.0, OIDC, Clever/ClassLink |
| **LTI 1.3** | Provider/consumer |
| **SCIM** | User provisioning |
| **File uploads** | TUS resumable uploads, local or MinIO/S3 |
| **Real-time** | WebSockets (`coder/websocket`), comm/notification hubs (partial) |
| **AI** | Multi-provider BYOK behind platform/org settings (`aiprovider` resolver; OpenRouter is one provider) |
| **Compliance** | FERPA, GDPR, COPPA, CCPA routes and audit logs |
| **Mobile** | Android/iOS clients + shared API |
| **Marketing site** | `www/` separate Vite app |
| **IaC** | `iac/` Terraform for self/production clouds |

---

## 9. Greenfield Starter Checklist

Use this to bootstrap a **new** project with Lextures-like architecture (minimal viable platform):

### Repo skeleton

- [ ] `server/` Go module with `cmd/server`, `internal/{app,httpserver,service,repos,models,auth,apierr,config,migrate,db}`
- [ ] `clients/web/` Vite + React + TS + Tailwind v4
- [ ] `docker-compose.yml` (postgres + server + web)
- [ ] `docker-compose.dev.yml` (Vite port 5173, hot reload)
- [ ] Root `AGENTS.md` or `README` with start commands
- [ ] `.github/workflows/ci.yml` (postgres service + go test + web lint/test)
- [ ] `.env.example` files (never commit real `.env`)

### Backend (day one)

- [ ] Chi router: `/health`, `/api/v1/auth/signup`, `/api/v1/auth/login`, `/api/v1/me`
- [ ] pgx pool + migrations `001_users.sql` (users table + roles)
- [ ] Argon2id + JWT + refresh tokens
- [ ] `apierr` JSON envelope
- [ ] CORS for `PUBLIC_WEB_ORIGIN`
- [ ] `RUN_MIGRATIONS=true` on boot

### Frontend (day one)

- [ ] `VITE_API_URL` → `lib/api.ts` + `authorizedFetch`
- [ ] Login/signup pages + token storage
- [ ] React Router + one protected dashboard route
- [ ] Zod for API responses (even before OpenAPI codegen)

### Defer until needed

- OpenAPI codegen + CI drift check (P0.1 in ARCH.md)
- TanStack Query (replace fetch/useEffect pattern)
- Auth middleware on router (vs per-handler checks)
- S3 file storage abstraction
- Job queue, Redis cache, WebSockets
- Husky (add when team grows)

### Conventions to adopt from Lextures

- TypeScript **exhaustive switch** with `never` in default case (`.cursor/rules`)
- **Imports at top of file** only (no inline imports)
- Permission strings `scope:area:function:action` if you need RBAC early
- Feature flags in **DB**, not build-time env (for operability)
- Separate **platform secrets** (env) from **product toggles** (DB)

---

## 10. Architecture Roadmap (From `docs/ARCH.md`)

Lextures’ own engineers prioritize these **before** scaling features — good guidance for what **not** to replicate blindly:

| Priority | Item |
|----------|------|
| **P0** | OpenAPI-first API contract + generated TS types |
| **P0** | Break up monolithic handler/page files (>500 LOC) |
| **P0** | Auth as chi middleware, not manual checks in every handler |
| **P0** | TanStack Query for server state |
| **P0** | DB query audit (N+1, indexes) |
| **P1** | Postgres job queue, structured JSON logs + metrics, S3 file store, Playwright E2E on PRs |

**Current snapshot (ARCH.md):** ~67K LOC Go, ~265 TSX files, hand-maintained API clients, seven React contexts, local filesystem storage, limited integration tests on backend.

---

## 11. Minimal `go.mod` Dependencies (Starter Set)

From Lextures’ core (trim the rest for a new app):

```go
require (
    github.com/go-chi/chi/v5
    github.com/jackc/pgx/v5
    github.com/golang-jwt/jwt/v5
    github.com/alexedwards/argon2id
    github.com/google/uuid
    golang.org/x/crypto
)
```

---

## 12. Minimal Web `package.json` Dependencies (Starter Set)

```json
{
  "dependencies": {
    "react": "^19",
    "react-dom": "^19",
    "react-router-dom": "^7",
    "zod": "^4"
  },
  "devDependencies": {
    "@vitejs/plugin-react": "^6",
    "@tailwindcss/vite": "^4",
    "tailwindcss": "^4",
    "typescript": "~6",
    "vite": "^8",
    "vitest": "^4",
    "@testing-library/react": "^16",
    "eslint": "^9",
    "typescript-eslint": "^8"
  }
}
```

---

## 13. Key Docs in the Repo

| Doc | Purpose |
|-----|---------|
| `README.md` | Product + stack summary |
| `docs/getting-started.md` | Docker, bootstrap admin, local dev |
| `docs/ARCH.md` | Deep architecture recommendations |
| `AGENTS.md` | Commands, env vars, gotchas for agents/CI VMs |
| `server/.env.example` | API configuration template |

---

## 14. One-Paragraph Summary

**Lextures** is a **monorepo LMS** built as a **stateless Go API** (chi + pgx + JWT + layered `httpserver → service → repos → models`) and a **React 19 SPA** (Vite + TS + Tailwind v4 + Zod-validated fetch clients), backed by **PostgreSQL 16** and **Docker Compose** for dev/prod. Configuration splits **secrets in env** and **feature toggles in the database**. The project is mature in domain features but intentionally documents gaps (OpenAPI coverage, React Query, auth middleware, job queue) so new work can improve the platform shape without copying 220 migrations or compliance modules wholesale.

For a new project: **copy the repo layout, stack versions, error/auth patterns, and Compose-based dev loop**; start with **one migration, a handful of API routes, and a thin React shell**; adopt ARCH.md **P0 items early** if you expect multiple clients or a growing team.
