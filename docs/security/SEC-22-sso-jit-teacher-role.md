# SEC-22 — SSO JIT provisioning trusts a self-asserted Teacher role

- **Severity:** Low
- **Status:** Confirmed present
- **Area:** Server / SSO
- **File:** [server/internal/browsersaml/acs.go](../../server/internal/browsersaml/acs.go) (`guessTeacherFromAssertion` ~L314, JIT assignment ~L239)

## Problem

On SAML just-in-time provisioning, the role is derived from a fuzzy substring scan of assertion attributes:

```go
roleish := strings.Contains(strings.ToLower(att.Name), "role")
// ...
if strings.Contains(t, "instructor") || strings.Contains(t, "teacher") || strings.Contains(t, "faculty") {
```

Any attribute whose *name* contains "role" and whose *value* contains "teacher"/"instructor"/"faculty" anywhere promotes the new account to Teacher. The default for password signup is correctly `Student`, but the SAML path elevates based on loosely-matched, IdP-supplied (and in misconfigured IdPs, potentially user-influenced) attribute text.

## Risk

Teacher is a broad role here (course creation, `item:create`, roster access). Honoring a self-asserted teacher claim from an external IdP without exact-match mapping or admin review is a privilege-assignment weakness. The blast radius depends on the IdP's attribute hygiene.

## Fix

1. Require an **exact-match** mapping from a specific, configured attribute name to a specific value, configured per-IdP — not a substring scan across all attributes.
2. Default JIT accounts to `Student` and require an explicit promotion path (admin action, invite token, or a strict role-mapping table).
3. Log every JIT role assignment (ties into SEC-16).

## Verification

- A SAML assertion with `someRandomRole=teacher` in an unmapped attribute does **not** grant Teacher.
- Only the configured `(attributeName, value)` pair grants elevated roles.
