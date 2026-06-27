# ADR 0001 — Background Job Queue Backend: Postgres `SKIP LOCKED`

- **Status:** Accepted
- **Date:** 2026-06-27
- **Plan:** [17.3 Background Job Queue](../completed/17-platform-performance-operability/17.3-background-job-queue.md)

## Context

Long-running and failure-prone work (email, webhooks, AI grading, transcoding)
must move off the HTTP request path with retry, dead-lettering, and monitoring.
We needed a durable queue. Two options were considered (plan §3): a
Postgres-backed queue using `SELECT … FOR UPDATE SKIP LOCKED`, or a Redis-backed
queue.

## Decision

Use a **Postgres-backed queue** as the source of truth, implemented as a thin
custom layer (`jobs.queue` table + `SKIP LOCKED` claim) rather than an
extension such as `pgmq`.

Rationale:

1. Postgres is already required; no new infrastructure or extension to provision
   (`pgmq` is not available in our managed Postgres image, ruling out the plan's
   tentative recommendation).
2. Jobs can be enqueued in the same transaction as the triggering write, avoiding
   the "sent before commit" dual-write problem.
3. Launch job volume is modest; `SKIP LOCKED` row-locking scales horizontally
   across stateless app instances (plan 17.2) with no central coordinator.
4. A thin custom layer keeps the surface small and matches the existing repo
   conventions in `server/internal/repos/`.

## Consequences

- At-least-once delivery: handlers must tolerate re-execution (idempotency).
  Documented in the `Handler` contract.
- The worker runs in-process in the main binary (FR-5). If a job type needs
  isolation or independent scaling, hot types can later migrate to a Redis
  worker tier while Postgres stays the source of truth (plan §3 / risk table).
- Queue depth/throughput are exposed via `GET /admin/jobs` stats today;
  Prometheus gauges are deferred to the observability work (17.7).
