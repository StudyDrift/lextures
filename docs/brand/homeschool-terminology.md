# Homeschool terminology & copy deck

> **Canonical source for the self-learner → Homeschool rebrand.**
> Consumed by [HS.2](../completed/homeschool/HS.2-www-marketing-site-rebrand.md) ·
> [HS.3](../completed/homeschool/HS.3-web-client-rebrand.md) ·
> [HS.4](../completed/homeschool/HS.4-mobile-clients-rebrand.md) ·
> [HS.5](../completed/homeschool/HS.5-server-copy-and-onboarding-program.md) ·
> [HS.6](../completed/homeschool/HS.6-docs-compliance-and-e2e-metadata.md).
> Plan record: [HS.1](../completed/homeschool/HS.1-terminology-copy-deck-and-guardrails.md).
>
> Guard: `scripts/check-homeschool-terminology.sh` · Allowlist: `scripts/homeschool-terminology-allow.txt`.

## 1. Canonical English terms (FR-1)

| Use | Term |
|---|---|
| Segment / adjective | **Homeschool** (one word; capital H at sentence start or as a label) |
| Persona, singular | **homeschooler** |
| Persona, plural / nav label | **Homeschoolers** (when a plural persona is needed) |
| Nav / segment label (preferred) | **Homeschool** (≤ 16 chars; used in www nav + footer) |
| Account phrasing | **Homeschool account** |
| Route slug | `/homeschool` |
| TS / Swift / Kotlin symbol (lowerCamel) | `homeschool` |
| TS / Go constant (SCREAMING_SNAKE) | `HOMESCHOOL` |
| Kotlin / Swift enum case | `Homeschool` / `homeschool` |
| Analytics `program` value (new events) | `homeschool` |

**Decision (OQ-1):** Product uses **Homeschool** (one word), not "Home school". Rationale: standard en-US form as noun/adjective/verb; clean slug and symbols.

**Decision (OQ-2):** Nav label is **Homeschool** (segment noun), not "Homeschoolers", for width (≤ 16 chars).

## 2. Banned terms (FR-2)

These strings must not appear in product code or new docs outside the allowlist:

| Banned form | Notes |
|---|---|
| `self-learner` / `self learner` / `Self-Learner` | Case-insensitive on hyphen/space forms |
| `self-learners` / `self-learning` | Plural and gerund forms |
| `selfLearner` / `SelfLearner` / `SELF_LEARNER` / `self_learner` | Symbol and constant forms |

### Not banned (FR-3)

These are unrelated uses of the word "self" and **must not** be flagged:

- `self-paced`, `self-host`, `self-hosting`, `self-service`
- Host origin `self.lextures.com` / `https://self.lextures.com`
- Feature concept **learning paths** (separate product surface)

## 3. Do-not-rename list (FR-4)

These values stay byte-identical after the rebrand. Changing them breaks installed clients or rewrites history.

| Value | Location | Rationale |
|---|---|---|
| `"selfLearner"` | iOS `EnvironmentStore.Kind` raw value (`UserDefaults`) | Renaming resets stored API base on every installed device |
| `"selfLearner"` | Android `EnvironmentStore.Kind.storageValue` (`SharedPreferences`) | Same as iOS |
| `'self-learner'` | `onboarding_events.program` historical / CHECK constraint values | Rewriting rewrites analytics history |
| `ffOnboardingFlow` | Platform feature key | Keys are stable wire IDs; only labels change |
| `ffStripeBilling` | Platform feature key | Keys are stable wire IDs; only labels change |
| `ffAiStudyBuddy` | Platform feature key | Keys are stable wire IDs; only labels change |
| `https://self.lextures.com` | Deploy DNS, TLS, mobile API base, `SITE_LINKS` value | Domain is an explicit non-goal |
| `docs/completed/15-self-learner-specific/**` | Shipped archive path | 51+ inbound refs; historical plan IDs `15.1`–`15.13` |

Each of these is allowlisted in `scripts/homeschool-terminology-allow.txt` with a `# DO NOT RENAME` comment where applicable.

## 4. English copy deck (user-visible strings)

Every string changed by HS.2–HS.5. Downstream plans must use these exact new English values.

| Where | Key / file | Current | New | Plan |
|---|---|---|---|---|
| www nav + footer audience link | `header.tsx` / `site-footer.tsx` | `Self-learners` | `Homeschool` | HS.2 |
| www route | `app.tsx` | `/self-learner` | `/homeschool` (+ permanent redirect from old path) | HS.2 |
| www audience page eyebrow | `self-learner-page.tsx` → `homeschool-page.tsx` | `Self-learner` | `Homeschool` | HS.2 |
| www get-started card title | `get-started-page.tsx` | `Self-learner` | `Homeschool` | HS.2 |
| www get-started card body | `get-started-page.tsx` | `I'm studying independently, for a certification, or on my own schedule.` | `I'm homeschooling, studying for a certification, or learning on my own schedule.` | HS.2 |
| www pricing card label | `pricing-page.tsx` | `Self-learner` | `Homeschool` | HS.2 |
| www pricing card body | `pricing-page.tsx` | `For certification prep, language study, and independent learners who want adaptive practice without running their own server.` | `For homeschool families, certification prep, and language study — adaptive practice without running your own server.` | HS.2 |
| www pricing FAQ question | `pricing-page.tsx` | `How do self-learner accounts work?` | `How do homeschool accounts work?` | HS.2 |
| www pricing FAQ answer (capabilities) | `pricing-page.tsx` | `…K–12, higher-ed, and self-learner capabilities…` | `…K–12, higher-ed, and homeschool capabilities…` | HS.2 |
| www pricing hero tail | pricing CTAs | `…or sign up as an independent learner at self.lextures.com.` | `…or sign up for a homeschool account at self.lextures.com.` | HS.2 |
| mobile get-started card title | `auth.getStarted.homeschoolTitle` (was `selfLearnerTitle`) | `Self-learner` | `Homeschool` | HS.4 |
| mobile get-started card body | `auth.getStarted.homeschoolDescription` | `I'm studying independently, for a certification, or on my own schedule.` | `I'm homeschooling, studying for a certification, or learning on my own schedule.` | HS.4 |
| mobile login footer link | `auth.getStarted.changeEnvironment` | `Change school or learning path` | `Change school or homeschool account` | HS.4 |
| admin feature pack label | `platform-settings-panel.tsx` | `Marketplace & self-learner` | `Marketplace & homeschool` | HS.3 |
| admin flag label | `platform-feature-definitions.ts` / e2e matrix | `Self-learner onboarding` | `Homeschool onboarding` | HS.3 / HS.5 |
| admin flag label | `platform-feature-definitions.ts` / e2e matrix | `Stripe billing (self-learner)` | `Stripe billing (homeschool)` | HS.3 / HS.5 |
| admin flag description | `platform-feature-definitions.ts` | `Persistent self-learner AI companion…` | `Persistent homeschool AI companion…` | HS.3 / HS.5 |
| AI disclosure description | `server/internal/aidisclosure/disclosure.go` | `Standalone study companion for self-learners.` | `Standalone study companion for homeschoolers.` | HS.5 |
| Study-buddy system prompt | `server/internal/service/studybuddy/prompt.go` | `…help self-learners understand…` | `…help learners understand…` (segment-neutral; study buddy also serves school learners) | HS.5 |
| Analytics program (new) | www get-started / onboarding API | `self-learner` | `homeschool` (accept both during dual-write window) | HS.2 / HS.5 |
| ISMS scope label | `docs/isms/scope-statement.md` | `Hosted self-learner app` | `Hosted homeschool app` (host column unchanged) | HS.6 |

## 5. Translations (draft — locale review required)

Status: **draft** until a native-speaker reviewer marks each language **reviewed**. HS.4 must not merge with any `draft` cell remaining.

| Key | en | es | fr | ar | Status |
|---|---|---|---|---|---|
| `auth.getStarted.homeschoolTitle` | Homeschool | Educación en casa | École à la maison | التعليم المنزلي | draft |
| `auth.getStarted.homeschoolDescription` | I'm homeschooling, studying for a certification, or learning on my own schedule. | Estudio en casa, para una certificación o a mi propio ritmo. | J'étudie à la maison, pour une certification ou à mon rythme. | أدرس في المنزل، للحصول على شهادة، أو وفق جدولي الخاص. | draft |
| `auth.getStarted.changeEnvironment` | Change school or homeschool account | Cambiar de escuela o de cuenta de educación en casa | Changer d'école ou de compte école à la maison | تغيير المدرسة أو حساب التعليم المنزلي | draft |

### en-XA (pseudo-locale)

When keys rename, regenerate `en-XA` via `scripts/sync-mobile-locales.py` so pseudo strings wrap the new English:

| Key | en-XA |
|---|---|
| `auth.getStarted.homeschoolTitle` | `[Homeschool]` |
| `auth.getStarted.homeschoolDescription` | `[I'm homeschooling, studying for a certification, or learning on my own schedule.]` |
| `auth.getStarted.changeEnvironment` | `[Change school or homeschool account]` |

### Legacy translations to replace

Existing `es` / `fr` / `ar` values for the get-started keys are literal translations of *self-taught* and must all be replaced — they do not carry the homeschool meaning:

| Locale | Old title (wrong sense) | Old description (wrong sense) |
|---|---|---|
| es | `Autoaprendizaje` | `Estudio de forma independiente…` |
| fr | `Apprenant autonome` | `J'apprends de façon autonome…` |
| ar | `متعلم ذاتي` | `أتعلم بشكل مستقل…` |

### Locale review owners (OQ-4)

| Locale | Owner | Status |
|---|---|---|
| es | *unassigned* | draft |
| fr | *unassigned* | draft |
| ar | *unassigned* | draft |

Arabic entries are RTL-clean (no embedded Latin punctuation at string edges).

## 6. Symbol rename map (for engineers)

| Layer | Old | New | Notes |
|---|---|---|---|
| www route | `/self-learner` | `/homeschool` | Keep permanent redirect stub at old path |
| www page component | `SelfLearnerPage` | `HomeschoolPage` | File rename |
| www constant | `SELF_LEARNER_ORIGIN` | `HOMESCHOOL_ORIGIN` | **Value** stays `https://self.lextures.com` |
| www link key | `SITE_LINKS.selfLearner` | `SITE_LINKS.homeschool` | Value unchanged |
| www path union | `'self-learner'` | `'homeschool'` | Dual-write analytics if needed |
| iOS env kind case | `case selfLearner` | `case homeschool = "selfLearner"` | **Raw value pinned** |
| iOS API base | `selfLearnerAPIBase` | `homeschoolAPIBase` | Value unchanged |
| Android kind | `SelfLearner("selfLearner")` | `Homeschool("selfLearner")` | **storageValue pinned** |
| Android API base | `SELF_LEARNER_API_BASE` | `HOMESCHOOL_API_BASE` | Value unchanged |
| Mobile locale keys | `auth.getStarted.selfLearner*` | `auth.getStarted.homeschool*` | All four locales + en-XA |
| Onboarding program (new) | `self-learner` | `homeschool` | Historical rows keep old value |

## 7. Guardrail usage

```bash
# Fail on non-allowlisted hits (required in CI as of HS.6)
scripts/check-homeschool-terminology.sh

# Optional inventory without failing (local use only)
scripts/check-homeschool-terminology.sh --warn

# Fixture self-test
scripts/check-homeschool-terminology.sh --self-test
```

### Permanent allowlist classes (HS.6 FR-11)

- `docs/completed/**` — historical plan prose (not rewritten)
- `server/migrations/**` — applied, immutable
- EnvironmentStore persisted raw values (`"selfLearner"`)
- Historical `'self-learner'` program literals that dual-read old events
- `e2e/coverage/**` — generated paths containing the archive folder name
- Bootstrap map key derived from the frozen section-15 archive path
- Plan index labels/hrefs into the frozen section-15 archive (`docs/plan/README.md`)
- Permanent `/self-learner` redirect stub sources under `www/`
- This file (`docs/brand/homeschool-terminology.md`) — defines banned terms
- Guard script + allowlist + fixtures under `scripts/`

## 8. Reading-level note (a11y)

Replacement sentences keep comparable length to today's copy. Nav labels stay ≤ 16 characters (`Homeschool` = 10). No increase in Flesch–Kincaid grade is intended for the six primary replacement sentences in §4.

## 9. Open product questions (carried from HS.1)

1. **OQ-3** — Keep certification/language use-case cards on the audience page, or lead with homeschool-specific cards? Affects HS.2 scope.
2. **OQ-5** — Announce the rename to existing hosted accounts, or land silently? No account identifier changes, so silent is defensible.

## 10. Changelog

| Date | Change |
|---|---|
| 2026-07-22 | HS.1 initial deck: canonical terms, banned list, do-not-rename, EN/ES/FR/AR draft copy, guard script |
| 2026-07-22 | HS.6: guard required in CI (fail mode); permanent allowlist pruned; `SL` → `HS` in plan/e2e metadata |
