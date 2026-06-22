# 03 — SCIM Group provisioning is a non-functional placeholder

- **Category:** Feature not fully implemented
- **Severity:** P2 (advertised SCIM capability is partial)
- **Area:** Identity / SCIM 2.0 provisioning (plan 4.5)

## Summary

`README.md` lists **SCIM** for enterprise identity. SCIM **User** provisioning works, but
the **Group** endpoints are placeholders: listing groups always returns an empty
collection, and creating a group returns a hard-coded `"placeholder"` resource. There is no
group → role mapping, so IdP-driven **group-based role assignment does not function**.

## Evidence

`server/internal/provisioning/scim/groups.go`:

```go
// WriteGroupList returns an empty SCIM Group collection (group→role mapping deferred).
func WriteGroupList(w http.ResponseWriter) {
    _ = json.NewEncoder(w).Encode(groupListResponse{
        Schemas:      []string{"urn:ietf:params:scim:api:messages:2.0:ListResponse"},
        TotalResults: 0,
        StartIndex:   1,
        ItemsPerPage: 0,
        Resources:    []*groupResource{},
    })
}

// WriteGroupCreated returns a placeholder Group so IdP probes can succeed without mapping yet.
func WriteGroupCreated(w http.ResponseWriter, baseURL string, id string) {
    // ...
    gr := groupResource{
        Schemas:     []string{"urn:ietf:params:scim:schemas:core:2.0:Group"},
        ID:          id,
        DisplayName: "placeholder",
    }
    _ = json.NewEncoder(w).Encode(gr)
}
```

The comments themselves state "group→role mapping deferred" and "placeholder Group so IdP
probes can succeed without mapping yet."

## Impact

- Customers who provision roles/cohorts via IdP groups (Okta, Entra ID, etc.) cannot use
  SCIM groups to drive Lextures roles or enrollments — group pushes are accepted but
  silently dropped.
- SCIM group reads return empty even after the IdP believes it created groups, which can
  confuse provisioning reconciliation on the IdP side.

## Suggested fix

1. Implement Group CRUD backed by a real table, with membership (`members`) and a mapping
   from SCIM group → Lextures role / org-role / enrollment cohort.
2. Apply membership changes (add/remove members) to the corresponding role grants /
   enrollments, with audit entries.
3. Until implemented, document SCIM as "User provisioning only" in the SCIM/self-hosting
   docs so customers do not design around group-based roles.

## Acceptance criteria

- `GET /scim/v2/Groups` returns previously created groups with members.
- Adding a user to a mapped SCIM group grants the mapped role; removing them revokes it.
- Round-trips validate against an IdP SCIM conformance test (Okta/Entra).
