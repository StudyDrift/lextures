---
title: Self-hosting Lextures
date: 2026-06-01
description: Install Lextures with Docker Compose, create the first Global Admin account, and open the local web app.
author: Lextures Team
---

Lextures is AGPL-3.0 licensed and runs on your own infrastructure with Docker Compose. This guide covers a **development** install (hot reload for the API and web app). For production-style nginx hosting, use the same bootstrap steps with `docker-compose.prod.yml` (see the [Getting started](https://github.com/StudyDrift/lextures/blob/main/docs/getting-started.md) doc in the repository).

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/) with Compose
- A clone of the [Lextures repository](https://github.com/StudyDrift/lextures)

## 1. Configure the first platform admin

The **first person to sign up is not automatically a Global Admin**. That behavior was removed for security: otherwise anyone who reached your instance first could take over the platform.

To make **your** account the platform administrator:

1. In the repository root, create a file named `.env` (next to `docker-compose.yml`).
2. Set your email address — it must match the email you will use on the signup form (comparison is case-insensitive):

```bash
BOOTSTRAP_ADMIN_EMAIL=you@yourdomain.com
```

Optional, for AI features: after signing in as a global admin, open **Settings → Intelligence → Models** and add credentials for one or more AI providers (OpenRouter, Anthropic, OpenAI, Azure OpenAI, Bedrock, or Vertex — bring-your-own-key). Secrets are stored in the platform database.

You can copy from [`.env.example`](https://github.com/StudyDrift/lextures/blob/main/.env.example) in the repo.

> **Important:** Set `BOOTSTRAP_ADMIN_EMAIL` **before** the first human user signs up. If you already created an account without it, skip to [Promote an existing user](#promote-an-existing-user) below.

## 2. Start the stack

From the repository root:

```bash
docker compose -f docker-compose.yml -f docker-compose.dev.yml up --build -d
```

Wait until Postgres and the API are healthy (API on port **8080**).

## 3. Create your account

1. Open [http://localhost:5173](http://localhost:5173) in your browser.
2. Go to **Sign up** and register with the **same email** as `BOOTSTRAP_ADMIN_EMAIL`.
3. You should have Global Admin access (platform settings, RBAC, org admin tools, etc.). Refresh the page if menus look stale.

| Service    | URL |
| ---------- | --- |
| Web (dev)  | http://localhost:5173 |
| API        | http://localhost:8080 |
| PostgreSQL | localhost:5432 (user/password/db: `studydrift`) |

## Promote an existing user

If you signed up before setting `BOOTSTRAP_ADMIN_EMAIL`, grant Global Admin to your email:

```bash
docker compose -f docker-compose.yml -f docker-compose.dev.yml exec server \
  go run ./cmd/bootstrap-admin -email=you@yourdomain.com
```

Then refresh the app or sign out and back in.

## Production-style web (optional)

With `BOOTSTRAP_ADMIN_EMAIL` set in `.env`:

```bash
docker compose -f docker-compose.yml -f docker-compose.prod.yml up --build
```

The app is served at [http://localhost:3000](http://localhost:3000); the API remains on port **8080**.

## What Global Admin includes

Global Admin is the platform superuser role (RBAC name `Global Admin`). It can manage global platform settings, organizations, and other admin-only APIs. A normal **Teacher** account from first signup can teach and create courses but cannot access those platform controls.

## More detail

- Full getting started (local Go/Node without full Docker): [docs/getting-started.md](https://github.com/StudyDrift/lextures/blob/main/docs/getting-started.md)
- Security context: [docs/SECURITY_ISSUES.md — C2](https://github.com/StudyDrift/lextures/blob/main/docs/SECURITY_ISSUES.md)
