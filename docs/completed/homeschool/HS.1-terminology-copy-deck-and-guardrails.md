# HS.1 — Terminology, copy deck & rename guardrails

> Implementation plan. Source: product rebrand of the **self-learner** segment to **Homeschool**.
> Foundation plan for [HS.2](HS.2-www-marketing-site-rebrand.md) · [HS.3](HS.3-web-client-rebrand.md) ·
> [HS.4](HS.4-mobile-clients-rebrand.md) · [HS.5](HS.5-server-copy-and-onboarding-program.md) ·
> [HS.6](HS.6-docs-compliance-and-e2e-metadata.md).
> Code references: `www/src/lib/site-links.ts`, `clients/mobile/locales/*.json`,
> `clients/web/src/components/settings/platform-feature-definitions.ts`, `scripts/`.
>
> **Shipped:** terminology deck, allowlist, CI warn-mode guard. See
> [`docs/brand/homeschool-terminology.md`](../../brand/homeschool-terminology.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | HS.1 |
| **Section** | Brand & terminology |
| **Severity** | MINOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | **DONE** — deck at `docs/brand/homeschool-terminology.md`; guard `scripts/check-homeschool-terminology.sh` in CI `--warn` until HS.6 |
| **Estimated effort** | XS (≤1d) |
| **Owner (proposed)** | Product/Brand + Platform |
| **Depends on** | — |
| **Unblocks** | HS.2, HS.3, HS.4, HS.5, HS.6 |

---

## 1. Problem Statement

"Self-learner" is the label for our third audience segment (alongside K–12 and Higher Ed), and it
appears in ~60 places across the marketing site, three clients, the Go API, migrations, e2e
fixtures, and docs. Product is rebranding that segment to **Homeschool**. Without a single
authoritative copy deck and a mechanical guard, five parallel rename PRs will drift — different
capitalisation, different Spanish/French/Arabic wording, some symbols renamed and some not — and the
old term will silently creep back in the next feature. This plan produces the one artefact every
other HS plan consumes: the canonical term list, the translated copy deck, the explicit
**do-not-rename** list (persisted values and wire formats), and a CI check that fails the build when
a banned term reappears.

## 2. Goals

- Define one canonical English term set (noun, adjective, persona plural, route slug, symbol case)
  and publish it at `docs/brand/homeschool-terminology.md`.
- Publish a reviewed copy deck of every user-visible string that changes, in **all four shipped
  locales** (`en`, `es`, `fr`, `ar`) plus the `en-XA` pseudo-locale where it still exists.
- Publish the **do-not-rename** list: persisted enum raw values, DB column values, wire/JSON fields,
  feature-flag keys, and the `self.lextures.com` origin.
- Ship `scripts/check-homeschool-terminology.sh` — a grep guard with an explicit allowlist — wired
  into `.github/workflows/ci.yml`, warn-only until HS.6 flips it to fail.
- Give every downstream plan a single link to cite instead of re-deciding wording.

## 3. Non-Goals

- **No domain change.** `self.lextures.com` stays exactly as-is: DNS, TLS, the AWS deploy workflow
  (`.github/workflows/deploy-self-aws.yml`), `SchoolCodeLogic.selfLearnerAPIBase`'s *value*, the
  reserved school code `self`, and the ISMS scope entry's host column all keep the current host.
- No rename of the shipped archive `docs/completed/15-self-learner-specific/` (51 inbound refs).
- No new account type, role, or permission. There is no `self_learner` role in the schema today.
- No re-translation of unrelated strings; only the strings listed in §10.
- No changes to the `learning paths` feature (a separate product concept that merely shares a word).

## 4. Personas & User Stories

- **As a homeschooling parent**, I want the site and apps to name my situation directly so I can tell
  in five seconds whether Lextures is built for me.
- **As a homeschooled student**, I want the sign-up path I pick to say "Homeschool", not a term I
  would never use to describe myself.
- **As an engineer**, I want one document to cite for the exact string and symbol name so my rename
  PR does not get bikeshed in review.
- **As a localisation reviewer**, I want the four translations proposed side by side with context so
  I can approve or correct them in a single pass.
- **As an instructor or admin** on a school tenant, I want nothing about my surfaces to change.

## 5. Functional Requirements

- **FR-1.** The canonical English terms MUST be:

  | Use | Term |
  |---|---|
  | Segment / adjective | **Homeschool** (one word, capital H at sentence start or as a label) |
  | Persona, singular | **homeschooler** |
  | Persona, plural / nav label | **Homeschoolers** |
  | Account phrasing | **Homeschool account** |
  | Route slug | `/homeschool` |
  | TS/Swift/Kotlin symbol (lowerCamel) | `homeschool` |
  | TS/Go constant (SCREAMING_SNAKE) | `HOMESCHOOL` |
  | Kotlin/Swift enum case | `Homeschool` / `homeschool` |
  | Analytics `program` value | `homeschool` |

- **FR-2.** The banned terms list MUST be: `self-learner`, `self learner`, `Self-Learner`,
  `selfLearner`, `SelfLearner`, `SELF_LEARNER`, `self_learner`, `self-learners`, `self-learning`.
  The guard MUST be case-insensitive on the hyphen/space forms.
- **FR-3.** "Self-paced", "self-host", "self-hosting", "self-service", and the host
  `self.lextures.com` MUST NOT be flagged — they are unrelated uses of the word "self".
- **FR-4.** The do-not-rename list MUST include, with rationale:
  - `EnvironmentStore.Kind` persisted raw value `"selfLearner"` (iOS `UserDefaults`, Android
    `SharedPreferences`) — changing it resets the stored API base on every installed device.
  - `onboarding_events.program` historical value `'self-learner'` — changing it rewrites analytics
    history.
  - Platform feature keys `ffOnboardingFlow`, `ffStripeBilling`, `ffAiStudyBuddy`.
  - The origin string `https://self.lextures.com`.
  - Archive path `docs/completed/15-self-learner-specific/**`.
- **FR-5.** `scripts/check-homeschool-terminology.sh` MUST exit non-zero when a banned term appears
  outside `docs/brand/homeschool-terminology.md`, the allowlist file, or the archive path, and MUST
  print `file:line` for every hit.
- **FR-6.** The allowlist MUST live in a committed file (`scripts/homeschool-terminology-allow.txt`),
  one path-glob or `path:line-substring` per line, with a comment explaining each entry.
- **FR-7.** The script MUST accept `--warn` (exit 0, print findings) so it can be merged before the
  downstream plans land, and default to failing.
- **FR-8.** The copy deck MUST give, for every changed string: the key (or file:line for hardcoded
  copy), the current English text, the new English text, and the `es` / `fr` / `ar` translations
  marked `draft` or `reviewed`.

## 6. Non-Functional Requirements

- **Performance** — the guard runs `rg` over the tree; MUST complete in < 5 s on CI (exclude
  `node_modules`, `dist`, `*.lock`, `package-lock.json`, `e2e/coverage/**`).
- **Security** — none; the script reads the working tree only and takes no input from the network.
- **Privacy & Compliance** — the ISMS scope statement names the hosted app; its *label* changes in
  HS.6 while the host column does not. No processing-activity or RoPA change.
- **Accessibility** — new copy MUST keep reading level and sentence length comparable to today's
  (target: no increase in Flesch–Kincaid grade over the strings being replaced); nav labels stay
  ≤ 16 characters so the mobile drawer does not wrap.
- **Scalability** — n/a.
- **Reliability** — the guard MUST NOT be the only gate; each downstream plan carries its own tests.
- **Observability** — guard failures surface as a named CI step, `Terminology guard`.
- **Maintainability** — every allowlist entry MUST carry a `#` comment; entries without one fail
  review.
- **Internationalization** — Arabic entries MUST be RTL-clean (no embedded Latin punctuation at
  string edges); the deck records the locale review owner per language.
- **Backward compatibility** — see FR-4; the deck is the record of what deliberately did **not**
  change.

## 7. Acceptance Criteria

- **AC-1.** *Given* the repo at HEAD, *When* `scripts/check-homeschool-terminology.sh --warn` runs,
  *Then* it prints the full current inventory (≈60 hits) and exits 0.
- **AC-2.** *Given* all of HS.2–HS.6 have landed, *When* the script runs without `--warn`, *Then* it
  exits 0 with no findings.
- **AC-3.** *Given* a PR that introduces the string `selfLearner` in `clients/web/src/`, *When* CI
  runs, *Then* the `Terminology guard` step fails and names the offending `file:line`.
- **AC-4.** *Given* the string `self.lextures.com`, `self-paced`, or `self-hosting` anywhere in the
  tree, *When* the script runs, *Then* it produces no finding for them.
- **AC-5.** *Given* `docs/brand/homeschool-terminology.md`, *Then* it contains a row for every string
  changed by HS.2–HS.5 and each row has non-empty `en`, `es`, `fr`, `ar` cells.
- **AC-6.** *Given* the do-not-rename list, *When* a reviewer greps for each listed value, *Then*
  each is still present in the codebase after HS.6 ships.

## 8. Data Model

None. This plan adds documentation and a shell script only.

## 9. API Surface

None.

## 10. UI / UX

No UI. The deliverable is the copy deck that the UI plans consume. Draft content
(`draft` = needs native-speaker review before HS.4 merges):

### English (canonical)

| Where | Current | New |
|---|---|---|
| www nav + footer audience link | `Self-learners` | `Homeschool` |
| www route | `/self-learner` | `/homeschool` |
| www audience page eyebrow | `Self-learner` | `Homeschool` |
| www get-started card title | `Self-learner` | `Homeschool` |
| www get-started card body | `I'm studying independently, for a certification, or on my own schedule.` | `I'm homeschooling, studying for a certification, or learning on my own schedule.` |
| www pricing card label | `Self-learner` | `Homeschool` |
| www pricing card body | `For certification prep, language study, and independent learners who want adaptive practice without running their own server.` | `For homeschool families, certification prep, and language study — adaptive practice without running your own server.` |
| www pricing FAQ question | `How do self-learner accounts work?` | `How do homeschool accounts work?` |
| www pricing hero tail | `…or sign up as an independent learner at self.lextures.com.` | `…or sign up for a homeschool account at self.lextures.com.` |
| mobile get-started card title | `Self-learner` | `Homeschool` |
| mobile get-started card body | `I'm studying independently, for a certification, or on my own schedule.` | `I'm homeschooling, studying for a certification, or learning on my own schedule.` |
| mobile login footer link | `Change school or learning path` | `Change school or homeschool account` |
| admin feature pack label | `Marketplace & self-learner` | `Marketplace & homeschool` |
| admin flag label | `Self-learner onboarding` | `Homeschool onboarding` |
| admin flag label | `Stripe billing (self-learner)` | `Stripe billing (homeschool)` |
| admin flag description | `Persistent self-learner AI companion…` | `Persistent homeschool AI companion…` |
| AI disclosure description | `Standalone study companion for self-learners.` | `Standalone study companion for homeschoolers.` |

### Translations (draft — locale review required)

| Key | es | fr | ar |
|---|---|---|---|
| `auth.getStarted.homeschoolTitle` | `Educación en casa` | `École à la maison` | `التعليم المنزلي` |
| `auth.getStarted.homeschoolDescription` | `Estudio en casa, para una certificación o a mi propio ritmo.` | `J'étudie à la maison, pour une certification ou à mon rythme.` | `أدرس في المنزل، للحصول على شهادة، أو وفق جدولي الخاص.` |
| `auth.getStarted.changeEnvironment` | `Cambiar de escuela o de cuenta de educación en casa` | `Changer d'école ou de compte école à la maison` | `تغيير المدرسة أو حساب التعليم المنزلي` |

Existing `es`/`fr`/`ar` values for the two get-started keys today are literal translations of
*self-taught* (`Autoaprendizaje`, `Apprenant autonome`, `متعلم ذاتي`) and MUST all be replaced —
they do not carry the homeschool meaning in any of the three languages.

## 11. AI / ML Considerations

Not AI-touching, with one downstream note carried into HS.5: the study-buddy system prompt
(`server/internal/service/studybuddy/prompt.go:10`) contains the literal word "self-learners" and is
sent to the model on every turn. Changing it changes model-visible context, so it belongs to HS.5's
eval checklist, not to a blanket find-and-replace here.

## 12. Integration Points

- Internal: `scripts/` (new script + allowlist), `.github/workflows/ci.yml` (new step),
  `docs/brand/` (new terminology doc).
- No external services.
- No webhook or event emissions.

## 13. Dependencies & Sequencing

- Must ship **before**: HS.2, HS.3, HS.4, HS.5, HS.6.
- Must ship after: nothing.
- Shared infra: none. The guard is a shell script; no runner changes needed beyond `ripgrep`, which
  is already available on `ubuntu-latest`.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| "Homeschool" reads as K–12-only and alienates adult certification learners | M | M | Keep the use-case cards (certification, language learning, independent study) on the audience page; the label changes, the audience story stays three-pronged |
| Draft translations ship unreviewed and read as machine output | M | M | Deck marks each cell `draft`/`reviewed`; HS.4 cannot merge with any `draft` cell in `es`/`fr`/`ar` |
| Guard over-matches `self-hosting` / `self-paced` and blocks unrelated PRs | M | L | FR-3 negative cases are AC-4; the script is warn-only until HS.6 |
| Guard under-matches (e.g. `Self‑learner` with a non-ASCII hyphen) | L | L | Pattern uses `self.?learn` (any single separator) rather than a literal hyphen |
| Someone renames the persisted `"selfLearner"` storage value to satisfy the guard | M | **H** | Those exact values are allowlisted with a `# DO NOT RENAME` comment; HS.4 adds a unit test asserting the raw value |
| The user-facing decision is "Home school" (two words), not "Homeschool" | M | L | Single-source: the deck is one file and the slug is one constant; see OQ-1 |

## 15. Rollout Plan

- Feature flag: none (docs + CI only).
- Sequencing: merge the terminology doc + allowlist + script in warn mode → downstream plans land →
  HS.6 removes `--warn` from the CI step.
- Dogfood: run the script locally against the current tree and paste the inventory into the HS.1 PR
  description as the baseline count.
- GA criteria: AC-2 passes on `main`.
- Rollback: delete the CI step; the doc is inert.

## 16. Test Plan

- **Unit** — a fixture directory under `scripts/__fixtures__/terminology/` with one positive file
  (`selfLearner`), one negative file (`self-hosting`, `self-paced`, `self.lextures.com`), and one
  allowlisted file; a `bats`-free bash test asserts exit codes and printed line numbers.
- **Integration** — run the script against the real tree in `--warn` mode; assert non-zero finding
  count before HS.2–HS.6 and zero after.
- **End-to-end** — n/a.
- **Security** — n/a.
- **Accessibility** — reading-level spot check on the six replacement sentences in §10.
- **Performance / load** — assert the script completes in < 5 s in CI (`time` in the step).
- **Manual exploratory** — locale reviewers sign off `es`, `fr`, `ar` rows in the deck.

## 17. Documentation & Training

- New: `docs/brand/homeschool-terminology.md` — canonical terms, banned terms, do-not-rename list,
  copy deck, translation status.
- Updated: `AGENTS.md` commands table gains a `Terminology guard` row.
- Internal runbook: none.

## 18. Open Questions

1. **OQ-1 — "Homeschool" (one word) vs "Home school" (two words).** The rebrand request used
   "Home school". This plan assumes **"Homeschool"**: it is the standard en-US form as noun,
   adjective, and verb; it gives a clean slug (`/homeschool`) and clean symbols; and "Home school"
   reads as a school named Home. The decision is one row in the deck and one route constant — cheap
   to flip before HS.2 merges. **Needs a product yes/no before HS.2.**
2. **OQ-2 — Does "Homeschooler" belong in the nav, or is "Homeschool" better?** Nav today says
   "Self-learners" (plural persona). §10 proposes the segment noun for width reasons (≤ 16 chars).
3. **OQ-3 — Should the audience page keep the certification/language use-case cards** now that the
   label is narrower, or lead with homeschool-specific ones (co-op scheduling, portfolio records,
   state reporting)? Affects HS.2 §10 scope.
4. **OQ-4 — Locale review owners** for `es`, `fr`, `ar` are unassigned.
5. **OQ-5 — Do we announce the rename** to existing hosted accounts, or let it land silently? No
   account identifier changes, so silent is defensible; product's call.

## 19. References

- Files this work touches: `scripts/check-homeschool-terminology.sh` (new),
  `scripts/homeschool-terminology-allow.txt` (new), `docs/brand/homeschool-terminology.md` (new),
  `.github/workflows/ci.yml`, `AGENTS.md`.
- Current inventory sources: `www/src/**`, `clients/web/src/components/settings/**`,
  `clients/mobile/locales/*.json`, `clients/ios/Lextures/**`, `clients/android/app/src/main/**`,
  `server/internal/**`, `server/migrations/142_onboarding_events.sql`, `e2e/lib/**`,
  `docs/isms/scope-statement.md`.
- Related plans: [HS.2](HS.2-www-marketing-site-rebrand.md), [HS.3](HS.3-web-client-rebrand.md),
  [HS.4](HS.4-mobile-clients-rebrand.md),
  [HS.5](HS.5-server-copy-and-onboarding-program.md),
  [HS.6](HS.6-docs-compliance-and-e2e-metadata.md).
