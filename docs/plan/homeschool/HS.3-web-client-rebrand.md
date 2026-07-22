# HS.3 — Web client: auth screens, onboarding & admin labels

> Implementation plan. Source: product rebrand of the **self-learner** segment to **Homeschool**.
> Terminology and copy are fixed by [HS.1](HS.1-terminology-copy-deck-and-guardrails.md).
> Code references: `clients/web/src/pages/{login,signup}.tsx`,
> `clients/web/public/locales/{en,es,fr,ar}/auth.json`,
> `clients/web/src/components/settings/{platform-feature-definitions.ts,platform-settings-panel.tsx}`,
> `clients/web/src/components/onboarding/use-onboarding-redirect.ts`, `e2e/lib/platform-feature-matrix.ts`.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | HS.3 |
| **Section** | Web client (`clients/web`) |
| **Severity** | MINOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | THIN — the auth screens never name the segment at all: every string assumes the visitor belongs to a school (`"the email your course or school uses"`, placeholder `you@school.edu`), even on `self.lextures.com` where no org is resolved. Admin surfaces still say "self-learner" in four places. |
| **Estimated effort** | XS (≤1d) |
| **Owner (proposed)** | Web team |
| **Depends on** | HS.1 (copy deck) |
| **Unblocks** | HS.6 (guard flip) |

---

## 1. Problem Statement

Two separate problems land in the same PR. First, the **admin** surfaces still use the old segment
name: the platform-settings pack is labelled "Marketplace & self-learner", and three feature flags
carry "self-learner" in their label or description — with an e2e test asserting those exact strings.
Second, the **auth screens** — the first thing a homeschool signup sees on `self.lextures.com` — are
written entirely for school users. `auth.login.subtitle` says "Use the email your course or school
uses", the email placeholder is `you@school.edu`, and the parent checkbox is phrased around "when my
school links my account". A homeschooling parent creating an account is told, three times on one
screen, that this product is for someone else. The rebrand is the moment to fix both.

## 2. Goals

- Rename the four admin-surface strings to the HS.1 terms, in lockstep with the e2e label matrix.
- Make the login and signup screens read correctly for a visitor with **no** school, without
  breaking the school-tenant experience.
- Keep every feature-flag **key** unchanged (`ffOnboardingFlow`, `ffStripeBilling`,
  `ffAiStudyBuddy`) — labels only.
- Ship all four locales (`en`, `es`, `fr`, `ar`) in the same PR so no locale regresses to English.
- Leave no `self-learner` string in `clients/web/`.

## 3. Non-Goals

- No change to the auth *flow*: no new screens, no school-vs-homeschool chooser in the web client
  (that chooser exists on www `/get-started` and in the mobile apps, not here).
- No change to `account_type`. Signup sends `account_type: 'parent' | undefined`; there is no
  homeschool account type and this plan does not add one.
- No feature-flag key renames, no flag-pack restructuring, no new flags.
- No redesign of `PublicAuthShell`, `OidcSignInButtons`, or the MFA/magic-link screens.
- No change to org-scoped login copy (`auth.login.orgSubtitle`) — that path already knows the org.

## 4. Personas & User Stories

- **As a homeschooling parent creating an account on `self.lextures.com`**, I want the sign-up screen
  to make sense for someone with no school, so I do not assume I am in the wrong place.
- **As a homeschooled student**, I want the email field's example not to be `you@school.edu`.
- **As a parent/guardian registering for read-only access**, I want the checkbox to describe both
  routes — a school linking my account, or my own homeschool account being linked.
- **As a student at a school tenant**, I want my org-branded login screen to be unchanged.
- **As a global admin**, I want the Settings → Global platform pack and flag labels to use the same
  segment name the marketing site uses.

## 5. Functional Requirements

- **FR-1.** `platform-settings-panel.tsx:34` MUST change
  `marketplace: 'Marketplace & self-learner'` → `'Marketplace & homeschool'`.
- **FR-2.** `platform-feature-definitions.ts` MUST change:
  - line 242 `label: 'Stripe billing (self-learner)'` → `'Stripe billing (homeschool)'`
  - line 488 `label: 'Self-learner onboarding'` → `'Homeschool onboarding'`
  - line 497 description `'Persistent self-learner AI companion…'` → `'Persistent homeschool AI companion…'`
- **FR-3.** `e2e/lib/platform-feature-matrix.ts:738,930` MUST be updated **in the same commit** —
  that file's header states "UI labels must match platform-feature-definitions.ts exactly" and
  `e2e/tests/platform-features-matrix-meta.spec.ts` enforces it.
- **FR-4.** The comment at `use-onboarding-redirect.ts:7` MUST say "homeschool learners".
- **FR-5.** `auth.login.subtitle` MUST be rewritten so it does not presuppose a school. Proposed en:
  `Use the email on your Lextures account. If your school connects single sign-on, those options appear here.`
- **FR-6.** `auth.login.emailPlaceholder` MUST become a school-neutral example (`you@example.com`).
  The `ar` locale currently carries the untranslated English value and MUST be updated too.
- **FR-7.** `auth.signup.subtitle` MUST be rewritten to cover both audiences. Proposed en:
  `One account for your courses, assignments, and messages — whether you are learning at home or through a school.`
- **FR-8.** `auth.signup.registerAsParent` MUST be rewritten so it does not require a school.
  Proposed en: `I am registering as a parent or guardian for read-only access to a learner's account.`
- **FR-9.** All four locale files (`en`, `es`, `fr`, `ar`) MUST be updated for FR-5…FR-8 in the same
  PR, using the HS.1 deck; no key may be added to one locale and not the others.
- **FR-10.** No key names change — only values — so no `i18n` fallback or missing-key path is
  exercised.
- **FR-11.** After this plan, `rg -i 'self.?learn' clients/web/` MUST return zero matches.

## 6. Non-Functional Requirements

- **Performance** — string-only; locale JSON size delta < 1 KB per locale. No new bundle imports.
- **Security** — no authn/authz change. The parent checkbox still only sets
  `account_type: 'parent'`; server-side authorization is untouched.
- **Privacy & Compliance** — the parent-registration copy is a consent-adjacent string; the reworded
  version MUST still make clear the access granted is **read-only** and requires a link performed by
  someone else. Legal review of the FR-8 wording before merge.
- **Accessibility** — WCAG 2.1 AA: label/`for` associations, `aria-describedby` on the password
  field, and the `role="status"`/`role="alert"` regions are unchanged. New strings MUST not exceed
  the current line count at 320 px (the signup card is `max-w-md`).
- **Scalability** — n/a.
- **Reliability** — a missing key falls back to the key string, which would be visibly broken;
  FR-9 + the locale parity test prevent it.
- **Observability** — none required.
- **Maintainability** — admin labels stay defined once in `platform-feature-definitions.ts`; e2e
  mirrors it.
- **Internationalization** — `ar` is RTL; the reworded strings MUST be checked in RTL layout
  (`clients/web/src/i18n/rtl-locales.ts` drives `dir`).
- **Backward compatibility** — no keys removed, so a stale cached locale bundle degrades to old copy
  rather than to a missing string.

## 7. Acceptance Criteria

- **AC-1.** *Given* a global admin on Settings → Global platform, *When* the marketplace pack renders,
  *Then* its heading reads "Marketplace & homeschool" and the flag rows read "Homeschool onboarding"
  and "Stripe billing (homeschool)".
- **AC-2.** *Given* `e2e/tests/platform-features-matrix-meta.spec.ts`, *When* the suite runs, *Then*
  label parity passes with the new strings.
- **AC-3.** *Given* an unauthenticated visitor at `self.lextures.com/signup`, *Then* the subtitle
  contains no word implying a required school, and the email placeholder is not `you@school.edu`.
- **AC-4.** *Given* a school tenant login at `{slug}.lextures.com`, *When* the org resolves, *Then*
  `auth.login.orgSubtitle` still renders (unchanged behaviour) and SSO buttons are unaffected.
- **AC-5.** *Given* each of `en`, `es`, `fr`, `ar`, *When* `/login` and `/signup` render, *Then* every
  visible string is translated (no raw key, no English fallback in a non-English locale).
- **AC-6.** *Given* `rg -i 'self.?learn' clients/web/src clients/web/public`, *Then* there are zero
  matches.
- **AC-7.** *Given* `npm run lint && npm run typecheck && npm run test` in `clients/web/`, *Then* all
  pass, including `src/pages/__tests__/{login,signup}.test.tsx`.

## 8. Data Model

None.

## 9. API Surface

None. `POST /api/v1/auth/signup` and `POST /api/v1/auth/login` request/response shapes are unchanged;
`account_type` keeps its current `'parent' | undefined` domain.

## 10. UI / UX

**Modified pages** — `/login`, `/signup`, Settings → Global platform (marketplace pack).
**New pages/components** — none.

**Key flows**

1. Homeschool signup: `self.lextures.com/signup` → fill name/email/password/timezone → Create account
   → post-auth redirect (`pickPostAuthPath`) — unchanged except copy.
2. Parent signup: same screen, checkbox ticked → `account_type: 'parent'` — unchanged except copy.
3. School signup at a tenant host — unchanged.
4. Admin reads the marketplace pack in Settings → Global platform — label only.

**States** — loading (`Creating account…`), error (`role="status"` message), and the password-policy
fetch failure path are all untouched; only static copy changes.

**Mobile / responsive** — the auth card is `max-w-md`; the two rewritten sentences MUST be checked at
320 px so the header does not push the form below the fold.

**Accessibility annotations** — focus order (display name → email → password → timezone → parent
checkbox → submit) unchanged; the parent checkbox keeps its `htmlFor` association; the password
strength meter keeps `aria-live="polite"`.

**Copy & i18n keys** — `auth.login.subtitle`, `auth.login.emailPlaceholder`, `auth.signup.subtitle`,
`auth.signup.registerAsParent` in `clients/web/public/locales/{en,es,fr,ar}/auth.json`. Admin labels
are hardcoded English in TSX, matching that file's existing convention.

## 11. AI / ML Considerations

Not AI-touching. The one AI-adjacent string — the `ffAiStudyBuddy` flag *description* — is an admin
label, not a prompt. The model-visible prompt lives in
[HS.5](HS.5-server-copy-and-onboarding-program.md).

## 12. Integration Points

- Internal: `clients/web/src/components/settings/*` (admin), `clients/web/public/locales/*` (i18n),
  `e2e/lib/platform-feature-matrix.ts` (label parity contract),
  `clients/web/src/components/onboarding/use-onboarding-redirect.ts` (comment only).
- External: none.
- Emissions: none.

## 13. Dependencies & Sequencing

- Must ship after: [HS.1](HS.1-terminology-copy-deck-and-guardrails.md).
- Must ship **with**: the `e2e/lib/platform-feature-matrix.ts` edit (FR-3) — same commit, or CI
  breaks on `main`.
- Must ship before: [HS.6](HS.6-docs-compliance-and-e2e-metadata.md).
- Shared infra: none.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Admin label edited without the e2e matrix → red `main` | H | M | FR-3 makes it one commit; PR checklist item; the meta spec fails fast and names both files |
| Reworded parent-consent copy weakens the read-only disclosure | L | H | Legal review gate in §6; AC keeps "read-only" mandatory in the string |
| Non-English locales ship stale while `en` changes | M | M | FR-9 + AC-5; locale files are edited together |
| Arabic rewrite breaks RTL rendering (Latin punctuation at string edges) | L | L | RTL visual check in the test plan |
| Rewriting login copy confuses school users who *do* need their school email | M | M | FR-5 keeps the SSO sentence; org-scoped path (AC-4) is untouched and is what school users actually see |

## 15. Rollout Plan

- Feature flag: none. Copy-only changes behind a flag cost more than they protect.
- Sequencing: single PR → merge → normal web deploy. No migration, no backfill.
- Dogfood: run `clients/web` locally in all four locales and screenshot `/login`, `/signup`, and the
  Settings marketplace pack.
- GA criteria: AC-1…AC-7 green.
- Rollback: revert the PR — no state to unwind.

## 16. Test Plan

- **Unit** — `src/pages/__tests__/signup.test.tsx` and `login.test.tsx`: assert on `data-testid`/role
  queries rather than the rewritten literals wherever they currently assert copy, so the tests do not
  re-break on the next copy edit; add one assertion that the placeholder is not `you@school.edu`.
- **Integration** — `e2e/tests/platform-features-matrix-meta.spec.ts` (label parity),
  `platform-features-ui-sample.spec.ts` (renders the pack).
- **End-to-end** — Playwright: sign up a fresh homeschool account on the default host and assert the
  post-auth redirect; sign in on an org host and assert org copy still renders.
- **Security** — authz matrix unchanged; re-run the existing signup authz tests to confirm the parent
  checkbox still cannot self-grant a link.
- **Accessibility** — axe on `/login` and `/signup` in `en` and `ar`; screen-reader pass on the parent
  checkbox label.
- **Performance / load** — n/a.
- **Manual exploratory** — all four locales × light/dark × 320 px / desktop on both auth screens.

## 17. Documentation & Training

- End-user docs: none.
- Admin docs: if the Settings → Global platform screenshots in the help centre show the marketplace
  pack, reshoot them.
- API reference: none.
- Internal runbook: none.

## 18. Open Questions

1. Should the web client gain an explicit "school vs homeschool" chooser like www `/get-started` and
   the mobile apps have, or does the host (`self.` vs `{slug}.`) remain the only signal? Out of scope
   here; worth a follow-up plan if signup drop-off shows confusion.
2. Does legal want specific wording for FR-8, given it is consent-adjacent copy?
3. Should the email placeholder be `you@example.com` or something warmer? Trivial, but it is the most
   visible string on the screen.
4. Are there help-centre screenshots showing "Marketplace & self-learner" that need reshooting?

## 19. References

- Existing files this work touches: `clients/web/src/pages/{login,signup}.tsx`,
  `clients/web/public/locales/{en,es,fr,ar}/auth.json`,
  `clients/web/src/components/settings/platform-feature-definitions.ts`,
  `clients/web/src/components/settings/platform-settings-panel.tsx`,
  `clients/web/src/components/onboarding/use-onboarding-redirect.ts`,
  `e2e/lib/platform-feature-matrix.ts`.
- External standards: WCAG 2.1 AA (SC 1.3.1, 3.3.2), RFC 2119.
- Related plans: [HS.1](HS.1-terminology-copy-deck-and-guardrails.md),
  [HS.5](HS.5-server-copy-and-onboarding-program.md),
  [W01 — i18n application coverage](../../completed/web/W01-i18n-application-coverage.md).
