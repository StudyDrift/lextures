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

- [01 — Adaptive Learning Core](../completed/01-adaptive-learning-core/) (completed plans)
- [02 — Assessment & Authoring](../completed/02-assessment-and-authoring/) (completed plans)
- [03 — Submissions, Grading & Academic Integrity](../completed/03-submissions-grading-integrity/) (completed plans)
- [04 — Identity, SSO & Provisioning](../completed/04-identity-sso-provisioning/) (completed plans)
- [05 — Multi-tenancy, Org Hierarchy & Roles](../completed/05-multi-tenancy-org-roles/) (completed plans)
- [06 — Communication & Collaboration](../completed/06-communication-collaboration/) (completed plans)
- [07 — Mobile, Offline & Cross-Platform](../completed/07-mobile-offline-cross-platform/) (completed plans)
- [08 — Content, Media & File Handling](../completed/08-content-media-files/) (completed plans)
- [09 — Analytics, Reporting & Insights](../completed/09-analytics-reporting/) (completed plans)
- [10 — Compliance, Privacy & Security](../completed/10-compliance-privacy-security/) (completed plans)
- [11 — Internationalization & Localization](../completed/11-i18n-l10n/) (completed plans)
- [12 — Accessibility (WCAG 2.1 AA)](../completed/12-accessibility/) (completed plans)
- [13 — K-12 Specific](../completed/13-k12-specific/) (completed plans)
- [14 — Higher-Education Specific](../completed/14-higher-ed-specific/) (completed plans)
- [15 — Homeschool Specific (formerly Self-Learner)](../completed/15-self-learner-specific/) · [archive](../completed/15-self-learner-specific/)
- [16 — Integrations & Extensibility](16-integrations-extensibility/)
- [17 — Platform, Performance & Operability](17-platform-performance-operability/)
- [18 — Admin Experience](18-admin-experience/)
- [19 — AI-Specific Capabilities](19-ai-capabilities/)
- [20 — Documentation & Trust Surfaces](../completed/20-docs-trust/) (completed plans)
- [21 — Mobile, Offline & Cross-Platform](21-mobile-offline-cross-platform/)
- [LH — Lighthouse remediation](../completed/lighthouse/) (completed; audits from `docs/lighthouse/` reports)
- [S — Standards & Legal Hardening](standards/) — bullet-proofing FERPA + every jurisdiction's privacy/AI/accessibility law (S01–S21)
- [AP — AI Multi-Provider / BYOK](ai-providers/) — OpenRouter as one provider among many; platform + org credentials; call-site migration (AP.1–AP.9)
- [VC — Visual Collaboration Boards](visual-collaboration/) · [completed](../completed/visual-collaboration/) — in-house real-time collaboration board behind a per-course feature flag (like the Whiteboard app); posts, layouts, presence, moderation, sharing, templates, export (VC.1–VC.10)
- [IQ — Interactive Quizzes](interactive-quizzes/) · [completed](../completed/interactive-quizzes/) — in-house, game-based live quizzing behind a per-course feature flag; author quiz kits, host live games with join codes + leaderboards, plus team/student-paced/async-homework modes, reports/gradebook, sharing, moderation & accessibility, and AI generation (IQ.1–IQ.11)
- [AN — Motion & Animation Polish](animations/) · [completed](../completed/animations/) — one cross-platform motion language (web/desktop/iOS/Android): the signature "bubble" spring, shared tokens, launch→landing & navigation transitions, skeleton→content load choreography, list/overlay/control motion, and delight moments — all reduced-motion & performance-budgeted (AN.1–AN.7)
- [PP — Parent Portal (staff workflows)](../completed/parent-portal/) — permission-gated assign parents/guardians, invite-when-missing-account, activate-link pairing (PP.1+) on top of shipped 13.1 / W02
- [HS — Homeschool rebrand](homeschool/) · [completed](../completed/homeschool/) — product segment rebrand to Homeschool across marketing, clients, server, docs & e2e metadata (HS.1–HS.6)

## Standards & Legal Hardening (2026-07-06)

A dedicated [`standards/`](standards/) folder hardens the shipped compliance layer (`docs/completed/10-*`) and extends it to every market we sell into. It contains a full **coverage matrix** (every law → owning plan → status) plus 21 template-compliant plans: cross-cutting engines (DSAR orchestration, retention/deletion, breach notification, consent ledger, RoPA/data-map, DPIA/AIA, transfer & subprocessor governance, children/age-assurance), US (FERPA deep-hardening, PPRA, state-law expansion), EU/UK (GDPR accountability, **EU AI Act high-risk**), rest-of-world (Canada/Quebec, Australia/NZ, Brazil, India, China, APAC/Africa), accessibility law (ADA Title II/III, §508, EAA/EN 301 549, AODA), and continuous compliance-evidence monitoring. See [standards/README](standards/README.md).

## Newly identified gaps — adoption-blocker scan (2026-06-19)

Gaps found by scanning `docs/completed`, `docs/plan`, and the codebase (handlers, repos, migrations) for features absent from **both** the completed set and existing plans, scoped to what blocks adoption by Higher-Ed, K-12, and Homeschool. Each was verified to have **no** implementation in `server/internal/httpserver`, `server/internal/repos`, or `server/migrations`.

| ID | Plan | Severity | Markets | Why it blocks adoption |
|---|---|---|---|---|
| 3.15 | ~~Peer review & peer assessment~~ → [completed](../completed/03-submissions-grading-integrity/3.15-peer-review-assessment.md) | BLOCKER (HE) / MAJOR | HE · K12 · HS | Done |
| 2.15 | ~~Differentiated assignments (assign-to / multiple due dates)~~ → [completed](../completed/02-assessment-and-authoring/2.15-differentiated-assignments.md) | MAJOR | K12 · HE | Done |
| 3.16 | ~~What-if grades (student projection)~~ → [completed](../completed/03-submissions-grading-integrity/3.16-what-if-grades.md) | MAJOR | HE · K12 · HS | Done |
| 3.17 | ~~Grade curving & scaling~~ → [completed](../completed/03-submissions-grading-integrity/3.17-grade-curving-scaling.md) | MAJOR (HE) | HE · K12 · HS | Done |
| 15.13 | [Tax compliance (Stripe Tax / VAT / GST)](../completed/15-self-learner-specific/15.13-tax-compliance.md) | BLOCKER (global) | HS · HE (CE) | Deferred to "phase 2" in 15.3/16.8; legally required to sell paid courses in EU/UK and US-nexus states |

**Reviewed and judged already-covered / out-of-scope:** IEP/504 *document* management (correctly delegated to special-ed software; enforcement settings exist in [12.10](../completed/12-accessibility/12.10-accommodations-engine.md)), native calendar (built), waitlists ([5.4](../completed/05-multi-tenancy-org-roles/5.4-sections.md)/[14.2](../completed/14-higher-ed-specific/14.2-course-catalog-registration.md)), coupons/subscriptions/refunds ([15.3](../completed/15-self-learner-specific/15.3-billing-stripe.md)), group assignments ([6.6](../completed/06-communication-collaboration/6.6-group-spaces.md)).