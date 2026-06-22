# Incomplete / Inaccessible / Buggy Features

This folder documents features that are **not fully implemented**, **implemented but
have no UI/UX path to reach them**, or **contain bugs**. Each finding has its own
markdown file with concrete evidence (`file:line`), expected vs. actual behaviour, and a
suggested fix so the dev team can pick it up directly.

> Audit method: static code review of `server/` (Go API) and `clients/web/` (React SPA)
> cross-referenced against the feature specs in `docs/completed/`. Signals used: `5xx`/
> `NotImplemented` handlers, "not yet ported"/"deferred"/MVP comments in live code paths,
> feature-flag toggles vs. the admin settings UI, and spec FRs vs. enforcement points.
> Date: 2026-06-22. Branch: `main` @ `f3424ff4`.

## Severity legend

- **P1 — Broken / security-or-integrity impact**: feature is reachable but does the wrong
  thing, or a documented MUST is not enforced.
- **P2 — Missing capability**: advertised/spec'd capability is stubbed or only partially
  implemented.
- **P3 — Inaccessible**: capability is implemented but cannot be reached/enabled through
  the product UI.
- **P4 — Doc / cosmetic**: stale documentation or low-impact gap.

## Findings

| # | Title | Category | Severity |
|---|-------|----------|----------|
| [01](01-lti-deep-linking-and-platform-launch-stubbed.md) | LTI 1.3 Deep Linking 2.0 + platform-initiated launch return 501 | Not fully implemented | P2 |
| [02](02-saml-single-logout-not-implemented.md) | SAML Single Logout (SLO) endpoint returns 501 | Not fully implemented | P2 |
| [03](03-scim-group-provisioning-non-functional.md) | SCIM Group provisioning is a non-functional placeholder | Not fully implemented | P2 |
| [04](04-conditional-release-not-enforced-on-quizzes-assignments.md) | Conditional release not enforced server-side for quizzes & assignment submissions | Bug / integrity | P1 |
| [05](05-originality-show-after-grading-not-enforced.md) | Originality "show after grading" visibility policy not enforced | Bug | P1 |
| [06](06-quiz-autosubmit-mastery-not-updated.md) | Quiz auto-submit does not update non-adaptive mastery | Not fully implemented | P2 |
| [09](09-platform-feature-flags-without-admin-toggle.md) | ~16 features gated behind platform flags that have no admin UI toggle | Implemented, no UI access | P3 |

## Quick triage

- **Fix first (P1):** [04](04-conditional-release-not-enforced-on-quizzes-assignments.md),
  [05](05-originality-show-after-grading-not-enforced.md). These are reachable today and
  violate a documented requirement / configured policy.
- **Unblocks whole features (P3):** [09](09-platform-feature-flags-without-admin-toggle.md)
  is the highest-leverage fix — it makes a large set of already-built features reachable by
  adding rows to one file.
- **Advertised-but-missing (P2):** [01](01-lti-deep-linking-and-platform-launch-stubbed.md),
  [02](02-saml-single-logout-not-implemented.md),
  [03](03-scim-group-provisioning-non-functional.md) all back claims in `README.md`
  ("LTI 1.3 provider/consumer", "SCIM").
