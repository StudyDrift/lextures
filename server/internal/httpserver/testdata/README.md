# HTTP contract golden files (TD.1)

This directory is the **refactoring safety net** for `internal/httpserver`. It records what the API *is* so refactors cannot silently drop routes or change auth posture / response shapes.

## Files

| Path | Purpose |
|---|---|
| `route_inventory.golden` | Sorted `METHOD\tPATTERN\tAUTH` for every registered route |
| `characterization/*.golden` | Per-endpoint status, Content-Type, and recursive JSON key sets |

### Auth posture (`AUTH` column)

Derived by **probing** (unauthenticated request), not by annotations:

| Value | Meaning |
|---|---|
| `anonymous` | Request without credentials did **not** receive `401` |
| `session` | Request without credentials received `401` |

Elevated (admin-only) scope is covered by characterization fixtures and existing RBAC tests. The inventory intentionally stays DB-free and side-effect-free so it can run on every PR in under 5s.

### Characterization snapshots

Each file records:

```text
status: <http status>
content-type: <normalized media type>
keys:
  <sorted JSON key paths>
```

Only **key paths** are pinned — never volatile values (IDs, timestamps, emails). Arrays appear as `field[*]` for element shapes.

## Commands

From repo root:

```bash
make route-inventory          # print live inventory to stdout
make route-inventory-update   # regenerate route_inventory.golden
```

From `server/`:

```bash
# Inventory (no DB)
go test ./internal/httpserver/ -run TestRouteInventory -count=1

# Regenerate inventory
UPDATE_GOLDEN=1 go test ./internal/httpserver/ -run TestRouteInventory -count=1

# Characterization (needs DATABASE_URL + migrations)
go test ./internal/httpserver/ -run TestCharacterization -count=1

# Regenerate characterization goldens
UPDATE_GOLDEN=1 go test ./internal/httpserver/ -run TestCharacterization -count=1
```

## Reviewing a golden diff

1. **Added routes** (`+ METHOD path`) — expected for new features; confirm auth class is correct.
2. **Removed routes** (`- METHOD path`) — treat as a **breaking API change** unless the route was never shipped.
3. **Auth change** (`session -> anonymous`) — **highest severity**; requires explicit security review (AC-6).
4. **Characterization key adds** (`+ key foo.bar`) — response shape grew; confirm clients tolerate it.
5. **Characterization key removals** — breaking for clients; require changelog / client update.

A PR that only moves handlers between files **must not** change these goldens (AC-3).

## Scope (what this does *not* cover)

- Job workers, WebSocket message payloads, and non-HTTP side effects
- Full behavioural coverage of all routes (use existing unit/e2e tests)
- OpenAPI document correctness (TD.3)
