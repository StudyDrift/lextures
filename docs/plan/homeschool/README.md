# HS — Self-learner → Homeschool rebrand (active)

Rename the **self-learner** audience/segment to **Homeschool** across every user-visible surface:
the marketing site, the web client auth + admin screens, the iOS and Android auth screens, and every
remaining place the product says "self-learner" / "self-learning".

The hosted app domain **`self.lextures.com` does not change** — that is an explicit non-goal for this
work stream (see [HS.1 §3](HS.1-terminology-copy-deck-and-guardrails.md#3-non-goals)).

| ID | Plan | Severity | Effort | Status |
|---|---|---|---|---|
| HS.1 | [Terminology, copy deck & rename guardrails](HS.1-terminology-copy-deck-and-guardrails.md) | MINOR | XS (≤1d) | **PLANNED** |
| HS.2 | [www marketing site rebrand](HS.2-www-marketing-site-rebrand.md) | MAJOR | S (1w) | **PLANNED** |
| HS.3 | [Web client auth, onboarding & admin labels](HS.3-web-client-rebrand.md) | MINOR | XS (≤1d) | **PLANNED** |
| HS.4 | [iOS & Android auth screens + mobile locales](HS.4-mobile-clients-rebrand.md) | MINOR | S (1w) | **PLANNED** |
| HS.5 | [Server copy, flag labels & onboarding program value](HS.5-server-copy-and-onboarding-program.md) | MINOR | S (1w) | **PLANNED** |
| HS.6 | [Docs, ISMS & e2e metadata](HS.6-docs-compliance-and-e2e-metadata.md) | MINOR | XS (≤1d) | **PLANNED** |

## Sequencing

```
HS.1 (copy deck + guardrail script)
  ├─→ HS.2  www            (independent deploy — GitHub Pages)
  ├─→ HS.3  clients/web    ─┐
  ├─→ HS.4  clients/ios+android │ HS.3/HS.5 must land in the same PR for the
  ├─→ HS.5  server         ─┘  platform-feature label parity e2e test
  └─→ HS.6  docs/e2e/ISMS  (last — flips the guardrail from warn to fail)
```

HS.2 ships first and alone: it is the only surface with SEO/redirect risk and it deploys via its own
workflow (`.github/workflows/pages-www.yml`, path-filtered on `www/**`).

## Naming

- Folder: `docs/plan/homeschool/` (active) · `docs/completed/homeschool/` (once shipped).
- Files: `HS.{n}-{kebab-slug}.md`.
- Every plan follows [`../_TEMPLATE.md`](../_TEMPLATE.md).
- The `Markets` axis token `SL` is retired in favour of `HS` — see
  [HS.6](HS.6-docs-compliance-and-e2e-metadata.md).

## What this rebrand is **not**

- Not a domain change (`self.lextures.com` stays).
- Not a data model change to account types — there is no `account_type = 'self_learner'` in the
  schema today; the "self-learner" path is a *marketing segment* plus a mobile *environment
  selection*, nothing more.
- Not a feature-flag key rename (`ffOnboardingFlow`, `ffStripeBilling`, `ffAiStudyBuddy` keep their
  keys; only their human-readable labels change).
- Not a rename of the shipped archive folder `docs/completed/15-self-learner-specific/` — 51
  inbound references, and the folder records historical plan IDs.
