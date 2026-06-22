# Lextures Implementation Plans

One plan per feature gap identified in `[docs/MISSING_FEATURES.md](../MISSING_FEATURES.md)`. Every plan follows the structure in `[_TEMPLATE.md](_TEMPLATE.md)`.

## Conventions

- File naming: `{section}.{number}-{kebab-slug}.md` (e.g. `1.1-learner-model-knowledge-state.md`).
- Folder per top-level section of the gap analysis.
- A plan is "ready" when every section in the template is filled (no `…` placeholders).
- Cross-references between plans are encouraged — use relative markdown links.

## Severity legend

- **BLOCKER** — cannot sell to the listed market without it.
- **MAJOR** — RFP-losing gap.
- **MINOR** — nice-to-have / parity.

## Sections

- [01 — Adaptive Learning Core](01-adaptive-learning-core/) (open plans) · [completed](../completed/01-adaptive-learning-core/)
- [02 — Assessment & Authoring](02-assessment-and-authoring/) (open plans) · [completed](../completed/02-assessment-and-authoring/)
- [03 — Submissions, Grading & Academic Integrity](03-submissions-grading-integrity/) (open plans) · [completed](../completed/03-submissions-grading-integrity/)
- [04 — Identity, SSO & Provisioning](../completed/04-identity-sso-provisioning/) (completed plans)
- [05 — Multi-tenancy, Org Hierarchy & Roles](../completed/05-multi-tenancy-org-roles/) (completed plans)
- [06 — Communication & Collaboration](../completed/06-communication-collaboration/) (completed plans)
- [07 — Mobile, Offline & Cross-Platform](../completed/07-mobile-offline-cross-platform/) (completed plans)
- [08 — Content, Media & File Handling](../completed/08-content-media-files/) (completed plans)
- [09 — Analytics, Reporting & Insights](../completed/09-analytics-reporting/) (completed plans)
- [10 — Compliance, Privacy & Security](../completed/10-compliance-privacy-security/) (completed plans)
- [11 — Internationalization & Localization](../completed/11-i18n-l10n/) (completed plans)
- [12 — Accessibility (WCAG 2.1 AA)](../completed/12-accessibility/) (completed plans)
- [13 — K-12 Specific](13-k12-specific/)
- [14 — Higher-Education Specific](14-higher-ed-specific/)
- [15 — Self-Learner Specific](15-self-learner-specific/) · [completed](../completed/15-self-learner-specific/)
- [16 — Integrations & Extensibility](16-integrations-extensibility/)
- [17 — Platform, Performance & Operability](17-platform-performance-operability/)
- [18 — Admin Experience](18-admin-experience/)
- [19 — AI-Specific Capabilities](19-ai-capabilities/)
- [20 — Documentation & Trust Surfaces](20-docs-trust/)
- [21 — Mobile, Offline & Cross-Platform](21-mobile-offline-cross-platform/)
- [LH — Lighthouse remediation](lighthouse/) (audits from `docs/lighthouse/` reports)

## Newly identified gaps — adoption-blocker scan (2026-06-19)

Gaps found by scanning `docs/completed`, `docs/plan`, and the codebase (handlers, repos, migrations) for features absent from **both** the completed set and existing plans, scoped to what blocks adoption by Higher-Ed, K-12, and Self-Learners. Each was verified to have **no** implementation in `server/internal/httpserver`, `server/internal/repos`, or `server/migrations`.

| ID | Plan | Severity | Markets | Why it blocks adoption |
|---|---|---|---|---|
| 3.15 | [Peer review & peer assessment](03-submissions-grading-integrity/3.15-peer-review-assessment.md) | BLOCKER (HE) / MAJOR | HE · K12 · SL | Explicitly deferred 3× (3.3/3.4/3.13); required by writing programs, large-lecture peer grading, MOOC cohorts |
| 2.15 | [Differentiated assignments (assign-to / multiple due dates)](02-assessment-and-authoring/2.15-differentiated-assignments.md) | MAJOR | K12 · HE | Only quiz time/attempt overrides exist; no per-section/group/student targeting — core differentiation & multi-section workflow |
| 3.16 | [What-if grades (student projection)](03-submissions-grading-integrity/3.16-what-if-grades.md) | MAJOR | HE · K12 · SL | Baseline student gradebook expectation (Canvas parity); absent entirely |
| 3.17 | [Grade curving & scaling](03-submissions-grading-integrity/3.17-grade-curving-scaling.md) | MAJOR (HE) | HE · K12 · SL | Faculty cannot curve/scale grades; forces error-prone spreadsheet round-trips |
| 15.13 | [Tax compliance (Stripe Tax / VAT / GST)](15-self-learner-specific/15.13-tax-compliance.md) | BLOCKER (global) | SL · HE (CE) | Deferred to "phase 2" in 15.3/16.8; legally required to sell paid courses in EU/UK and US-nexus states |

**Reviewed and judged already-covered / out-of-scope:** IEP/504 *document* management (correctly delegated to special-ed software; enforcement settings exist in [12.10](../completed/12-accessibility/12.10-accommodations-engine.md)), native calendar (built), waitlists ([5.4](../completed/05-multi-tenancy-org-roles/5.4-sections.md)/[14.2](../completed/14-higher-ed-specific/14.2-course-catalog-registration.md)), coupons/subscriptions/refunds ([15.3](../completed/15-self-learner-specific/15.3-billing-stripe.md)), group assignments ([6.6](../completed/06-communication-collaboration/6.6-group-spaces.md)).