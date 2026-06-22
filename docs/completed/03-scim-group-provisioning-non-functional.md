# 03 — SCIM Group provisioning is a non-functional placeholder

- **Category:** Feature not fully implemented
- **Severity:** P2 (advertised SCIM capability is partial)
- **Area:** Identity / SCIM 2.0 provisioning (plan 4.5)
- **Status:** Fixed (2026-06-22)

## Summary

`README.md` lists **SCIM** for enterprise identity. SCIM **User** provisioning works, but
the **Group** endpoints were placeholders: listing groups always returned an empty
collection, and creating a group returned a hard-coded `"placeholder"` resource. There was no
group → role mapping, so IdP-driven **group-based role assignment did not function**.

## Fix

Implemented full SCIM 2.0 Group provisioning backed by persistent tables and membership-driven
entitlement grants.

### Database (`server/migrations/310_scim_groups.sql`)

| Table | Purpose |
|-------|---------|
| `provisioning.scim_groups` | Group resources per institution (`display_name`, `external_id`) |
| `provisioning.scim_group_members` | Group membership (`group_id`, `user_id`) |
| `provisioning.scim_group_mappings` | Maps group display names → app roles, org roles, or course enrollments |

Audit log operations extended with `member_add` and `member_remove`.

### API (`server/internal/provisioning/scim/groups.go`, `scim_http.go`)

| Endpoint | Behaviour |
|----------|-----------|
| `GET /scim/v2/Groups` | Lists created groups with members; supports `filter=displayName eq "..."` |
| `POST /scim/v2/Groups` | Creates a group (optional initial members) |
| `GET /scim/v2/Groups/{id}` | Returns group with members |
| `PUT /scim/v2/Groups/{id}` | Full replace (display name, members) |
| `PATCH /scim/v2/Groups/{id}` | Partial update (`displayName`, `externalId`, add/remove `members`) |
| `DELETE /scim/v2/Groups/{id}` | Deletes group and revokes membership-driven grants |

### Group → entitlement mapping

When a user is added to a group, mappings are resolved and applied:

1. **`provisioning.scim_group_mappings`** rows for the institution + group `display_name`
   (kinds: `app_role`, `org_role`, `course_enrollment`).
2. **Fallback:** `user.provisioning_role_map` with `provider = 'scim'` and
   `external_role` matching the group display name (existing SCIM user-role pattern).

When a user is removed (or the group is deleted), grants are revoked only if no other group
membership still covers the same mapping.

### Configuration

IT admins configure group mappings by inserting rows into `provisioning.scim_group_mappings`
(or extending `provisioning_role_map` for app-role-only mappings). Example:

```sql
INSERT INTO provisioning.scim_group_mappings
  (institution_id, display_name, mapping_kind, app_role_name)
VALUES
  ('<institution-uuid>', 'Lextures Teachers', 'app_role', 'Teacher');
```

## Acceptance criteria

- [x] `GET /scim/v2/Groups` returns previously created groups with members.
- [x] Adding a user to a mapped SCIM group grants the mapped role; removing them revokes it.
- [ ] Round-trips validate against an IdP SCIM conformance test (Okta/Entra) — manual staging verification recommended.