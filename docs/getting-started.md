# Getting started

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/) with Compose, **or**
- [Go](https://go.dev/dl/) 1.25+, [Node.js](https://nodejs.org/) (current LTS), and PostgreSQL if you run services locally.

## Run with Docker (recommended for development)

From the repository root:

1. **Set the first platform admin (before anyone signs up).**  
   The first password account does **not** automatically receive Global Admin. It only gets full platform permissions when `BOOTSTRAP_ADMIN_EMAIL` matches that signup email (trimmed and lowercased).  
   Create a `.env` file next to `docker-compose.yml` (or export the variable in your shell):

   ```bash
   BOOTSTRAP_ADMIN_EMAIL=you@yourdomain.com
   ```

   Optional: configure AI provider credentials (BYOK) under **Settings → Intelligence → Models** — see [ai-providers-byok.md](ai-providers-byok.md).

2. **Start the stack:**

   ```bash
   docker compose -f docker-compose.yml -f docker-compose.dev.yml up --build -d
   ```

3. **Open the app and sign up** at [http://localhost:5173](http://localhost:5173) using the **same email** as `BOOTSTRAP_ADMIN_EMAIL`.

- **Web (Vite, HMR)**: [http://localhost:5173](http://localhost:5173)
- **API**: [http://localhost:8080](http://localhost:8080)
- **PostgreSQL**: `localhost:5432` (defaults match `docker-compose.yml`)

### Already signed up without Global Admin?

Promote an existing account (replace the email):

```bash
docker compose -f docker-compose.yml -f docker-compose.dev.yml exec server \
  go run ./cmd/bootstrap-admin -email=you@yourdomain.com
```

Then refresh the web app (or sign out and back in if admin menus still look limited).

From the host without Docker exec:

```bash
cd server
DATABASE_URL='postgres://studydrift:studydrift@localhost:5432/studydrift?sslmode=disable' \
  go run ./cmd/bootstrap-admin -email=you@yourdomain.com
```

See also [`.env.example`](../.env.example) at the repo root and [`server/.env.example`](../server/.env.example) when running the API outside Compose.

## Production-style web (nginx + static build)

From the repository root, set `BOOTSTRAP_ADMIN_EMAIL` the same way, then:

```bash
docker compose -f docker-compose.yml -f docker-compose.prod.yml up --build
```

The web app is served on [http://localhost:3000](http://localhost:3000) (API still on `8080` from the base compose file).

## Local development without full Docker (outline)

1. Start PostgreSQL (e.g. `docker compose -f docker-compose.yml up -d postgres`).
2. Copy [`server/.env.example`](../server/.env.example) to `server/.env` and set `DATABASE_URL`, `JWT_SECRET`, and `BOOTSTRAP_ADMIN_EMAIL` (if you want the first signup to be Global Admin).
3. In `server/`: `go run ./cmd/server` (or use [Air](https://github.com/air-verse/air) with `Dockerfile.dev` via `docker compose -f docker-compose.yml -f docker-compose.dev.yml`).
4. In `clients/web/`: `npm install` then `npm run dev` (set `VITE_API_URL` if the API is not at `http://localhost:8080`).

For architecture notes, see [Architecture recommendations](ARCH.md).
