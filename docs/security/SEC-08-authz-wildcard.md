# SEC-08 — Permission wildcard matches the *required* side

- **Severity:** High (architectural footgun; not directly exploitable today)
- **Status:** Confirmed present
- **Area:** Server / authorization
- **File:** [server/internal/authz/authz.go](../../server/internal/authz/authz.go) (`segmentMatches`)

## Problem

Permission strings are `scope:area:function:action`. The segment matcher treats `*` as a wildcard on **either** side:

```go
func segmentMatches(g, r string) bool {
    return g == "*" || r == "*" || g == r
}
```

The second clause (`r == "*"`) means a *required* permission string containing `*` matches **any** grant the user holds. Required strings are frequently built by concatenating a URL path parameter, e.g. `"course:" + courseCode + ":item:create"` in `course_sections.go` and ~20 similar call sites. If any code path lets a caller influence `courseCode` to be `*` (or an empty segment), the required string collapses to a wildcard and authorizes the caller against an unrelated grant.

Today this is partially blunted because `requireCourseAccess` first calls `enrollment.UserHasAccess(courseCode, viewer)`, and a literal `*` won't match a real enrollment row. But the wildcard-on-required semantics are latent: any future handler that builds a permission string from user input *without* that enrollment pre-check inherits a privilege-escalation bug.

## Risk

Privilege escalation / cross-tenant authorization bypass if a permission string is ever built from attacker-influenced input. Course codes are normally random (`C-XXXXXX`), but Canvas import, QTI import, and LTI deep-link mappings accept externally-derived identifiers, widening the input surface.

## Fix

1. Allow `*` only on the **granted** side:
   ```go
   func segmentMatches(g, r string) bool {
       return g == "*" || g == r
   }
   ```
   A required permission should always be a concrete string; a `*` in the required slot is a bug, not a match.
2. Validate `course_code` (and similar interpolated identifiers) at the route boundary: `^[A-Za-z0-9_-]{1,32}$`. Reject `*`, `..`, whitespace, and non-ASCII before the value ever reaches a permission string or a filesystem path (see SEC-10).

## Verification

- Unit test: `PermissionMatches("course:C-1:roster:read", "course:*:item:create")` returns `false`.
- A request with `course_code=*` or an empty course code is rejected at the boundary with 400, never reaching an authz check.
