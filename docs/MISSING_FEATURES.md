# Lextures — Missing Features & Gap Analysis (Index)

> This file is the canonical entry point for the feature-gap analysis. The original single
> gap-analysis document has been **decomposed into one implementation plan per gap**. Each plan
> follows [`plan/_TEMPLATE.md`](plan/_TEMPLATE.md) and is cross-referenced back here as its
> `Source:` header (e.g. *"Source: docs/MISSING_FEATURES.md §19"*).

## Where the plans live

| Location | Meaning |
|---|---|
| [`docs/plan/`](plan/) | **Open** gaps — features not yet implemented (or only partially). One file per gap. |
| [`docs/completed/`](completed/) | Gaps that have **shipped**. A plan moves here (status flipped to `COMPLETE`) once the feature lands. |

Start at [`docs/plan/README.md`](plan/README.md) for the section map, severity legend, and the
adoption-blocker scan.

## Sections (§ numbers used by `Source:` headers)

| § | Section | Status |
|---|---|---|
| 01 | Adaptive Learning Core | [completed](completed/01-adaptive-learning-core/) |
| 02 | Assessment & Authoring | [completed](completed/02-assessment-and-authoring/) |
| 03 | Submissions, Grading & Academic Integrity | [completed](completed/03-submissions-grading-integrity/) |
| 04 | Identity, SSO & Provisioning | [completed](completed/04-identity-sso-provisioning/) |
| 05 | Multi-tenancy, Org Hierarchy & Roles | [completed](completed/05-multi-tenancy-org-roles/) |
| 06 | Communication & Collaboration | [completed](completed/06-communication-collaboration/) |
| 07 | Mobile, Offline & Cross-Platform | [completed](completed/07-mobile-offline-cross-platform/) |
| 08 | Content, Media & File Handling | [completed](completed/08-content-media-files/) |
| 09 | Analytics, Reporting & Insights | [completed](completed/09-analytics-reporting/) |
| 10 | Compliance, Privacy & Security | [completed](completed/10-compliance-privacy-security/) |
| 11 | Internationalization & Localization | [completed](completed/11-i18n-l10n/) |
| 12 | Accessibility (WCAG 2.1 AA) | [completed](completed/12-accessibility/) |
| 13 | K-12 Specific | [completed](completed/13-k12-specific/) |
| 14 | Higher-Education Specific | [completed](completed/14-higher-ed-specific/) |
| 15 | Self-Learner Specific | [open](plan/15-self-learner-specific/) · [completed](completed/15-self-learner-specific/) |
| 16 | Integrations & Extensibility | [open](plan/16-integrations-extensibility/) · [completed](completed/16-integrations-extensibility/) |
| 17 | Platform, Performance & Operability | [open](plan/17-platform-performance-operability/) |
| 18 | Admin Experience | [open](plan/18-admin-experience/) |
| 19 | AI-Specific Capabilities | [open](plan/19-ai-capabilities/) · grading-agent: [auto-grader](completed/auto-grader-agent.md) · [canvas](completed/grader-agent-workflow-canvas.md) · [nodes](completed/agent-grader/) |
| 20 | Documentation & Trust Surfaces | [completed](completed/20-docs-trust/) |
| 21 | Mobile, Offline & Cross-Platform | [open](plan/21-mobile-offline-cross-platform/) |

Section numbers in `Source:` headers are stable identifiers; a header such as *"Source:
docs/MISSING_FEATURES.md §17.6"* refers to gap 17.6, whose plan is
[`plan/17-platform-performance-operability/17.6-rate-limiting.md`](plan/17-platform-performance-operability/17.6-rate-limiting.md).
