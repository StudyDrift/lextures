# ADR 0001: Remove MongoDB from the stack

**Status:** Accepted  
**Date:** 2026-05-29

## Context

`docker-compose.yml` included a MongoDB 7 service. The Go API never connected to it: all application data lives in PostgreSQL via `pgx`. Mongo added ~200 MB RAM on dev machines, an extra health-check dependency for `server`, and confusion for new contributors.

## Decision

Remove MongoDB from Docker Compose, documentation, and operational references. **PostgreSQL is the sole application database.**

## Consequences

- **Positive:** Simpler local setup (`docker compose up -d postgres` only), smaller attack surface, no unused container.
- **Negative:** None for current features (no data or code depended on Mongo).
- **Neutral:** Document-store needs must be evaluated against Postgres JSONB before adding a second database.

## Re-evaluation criteria

Re-introduce a document store only when:

1. A concrete feature plan exists with load/latency requirements.
2. Postgres JSONB (or a dedicated analytics pipeline) is benchmarked and rejected with evidence.
3. The change includes IaC, migrations, backups, and security review—not only a Compose service.

## References

- Removed `mongo` service from `docker-compose.yml`.
- Plan superseded: `docs/plan/17-platform-performance-operability/17.15-mongodb-usage-decision.md` (deleted after implementation).
