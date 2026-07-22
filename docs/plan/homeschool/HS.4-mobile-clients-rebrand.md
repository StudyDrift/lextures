# HS.4 — iOS & Android: auth screens and mobile locale sources

> Implementation plan. Source: product rebrand of the **self-learner** segment to **Homeschool**.
> Terminology and copy are fixed by [HS.1](HS.1-terminology-copy-deck-and-guardrails.md).
> Code references: `clients/mobile/locales/*.json`, `scripts/sync-mobile-locales.py`,
> `clients/ios/Lextures/Features/Auth/{GetStartedView,LoginView}.swift`,
> `clients/ios/Lextures/Core/Config/{EnvironmentStore,SchoolCodeLogic}.swift`,
> `clients/android/app/src/main/kotlin/com/lextures/android/features/auth/{GetStartedScreen,LoginScreen}.kt`,
> `clients/android/app/src/main/kotlin/com/lextures/android/core/config/{EnvironmentStore,SchoolCodeLogic}.kt`.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | HS.4 |
| **Section** | Mobile clients (iOS + Android) |
| **Severity** | MINOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | PARTIAL — the first screen of both apps offers "Self-learner" vs "School"; the label, the description, the icons, the enum cases, the persisted environment kind, and the tests all carry the old name |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | Mobile team |
| **Depends on** | HS.1 (copy deck, incl. reviewed `es`/`fr`/`ar` translations) |
| **Unblocks** | HS.6 (guard flip) |

---

## 1. Problem Statement

`GetStartedView` (iOS) and `GetStartedScreen` (Android) are the **first screen a new mobile user
sees**: two cards, "Self-learner" and "School", choosing which API base the app talks to. The
left-hand card is the homeschool entry point and it is labelled with the term being retired,
described as "studying independently, for a certification, or on my own schedule", and iconed with a
brain (`brain.head.profile` / `Icons.Default.Psychology`). Underneath, the choice persists as
`EnvironmentStore.Kind.selfLearner`, resolved through `SchoolCodeLogic.selfLearnerAPIBase`, and is
covered by unit tests on both platforms. The visible strings are generated — they must be changed in
`clients/mobile/locales/*.json` and regenerated, never hand-edited — and the existing `es`/`fr`/`ar`
translations say *self-taught* (`Autoaprendizaje`, `Apprenant autonome`, `متعلم ذاتي`), which is not
what "homeschool" means in any of those languages. The persisted storage value is the one thing that
must **not** move: changing it logs every installed device back to the environment chooser.

## 2. Goals

- Relabel the first card to "Homeschool" on both platforms, in all shipped locales, with copy that
  names homeschooling explicitly.
- Rename every `selfLearner` / `SelfLearner` symbol in both codebases to `homeschool` / `Homeschool`.
- Keep the persisted `EnvironmentStore` raw value `"selfLearner"` byte-identical, and prove it with a
  test, so no installed app loses its API base.
- Regenerate `Localizable.xcstrings` and `res/values*/strings.xml` through
  `scripts/sync-mobile-locales.py` so `scripts/check-mobile-i18n.sh` stays green.
- Replace the brain icon with a home/house icon so the card reads correctly at a glance.

## 3. Non-Goals

- No change to `self.lextures.com` — `SchoolCodeLogic`'s API-base **value** and the reserved school
  code `self` both stay.
- No change to the school-code step, its validation rules, its reserved list, or its error keys.
- No change to the persisted `UserDefaults` / `SharedPreferences` **keys** or **values**.
- No new environment kinds and no third card.
- No login/signup form changes beyond the one "Change school or …" link label.
- No re-translation of unrelated mobile strings (~3,000 keys per locale stay untouched).

## 4. Personas & User Stories

- **As a homeschooling parent installing the iOS app**, I want the first card to say "Homeschool" so I
  pick the right path without guessing.
- **As a Spanish-, French-, or Arabic-speaking homeschool user**, I want the card in my language to
  say "homeschool", not "self-taught".
- **As an existing user who already chose the self-learner environment**, I want the app to keep
  talking to `self.lextures.com` after updating — no re-onboarding, no logout.
- **As a school user**, I want the second card and the whole school-code flow unchanged.
- **As a mobile engineer**, I want the symbol names to match the marketing and admin vocabulary so
  cross-surface greps work.

## 5. Functional Requirements

### Strings (source of truth: `clients/mobile/locales/*.json`)

- **FR-1.** Rename keys `auth.getStarted.selfLearnerTitle` → `auth.getStarted.homeschoolTitle` and
  `auth.getStarted.selfLearnerDescription` → `auth.getStarted.homeschoolDescription` in **every**
  locale file present under `clients/mobile/locales/` (today: `en`, `es`, `fr`, `ar`, `en-XA`).
- **FR-2.** Values MUST come from the [HS.1 deck](HS.1-terminology-copy-deck-and-guardrails.md#10-ui--ux)
  and every non-English cell MUST be marked `reviewed` before merge.
- **FR-3.** `auth.getStarted.changeEnvironment` ("Change school or learning path") MUST be reworded —
  "learning path" collides with the shipped *Learning paths* feature. Proposed en:
  `Change school or homeschool account`.
- **FR-4.** `clients/ios/Lextures/Resources/Localizable.xcstrings` and
  `clients/android/app/src/main/res/values*/strings.xml` MUST be regenerated by running
  `python3 scripts/sync-mobile-locales.py`, **not** hand-edited. The Android resource name derives
  mechanically (`auth_getStarted_homeschoolTitle`).
- **FR-5.** `scripts/check-mobile-i18n.sh` MUST pass, which requires the regenerated artefacts to be
  committed and byte-identical to a fresh sync.

### iOS

- **FR-6.** `EnvironmentStore.Kind` MUST rename the case and **pin the raw value**:
  `case homeschool = "selfLearner"`. Swift derives `String` raw values from the case name, so an
  unpinned rename silently changes what is written to `UserDefaults` — this is the single highest-risk
  line in the plan.
- **FR-7.** `EnvironmentStore.selectSelfLearner()` → `selectHomeschool()`; the call site
  `GetStartedView.swift:66` follows.
- **FR-8.** `SchoolCodeLogic.selfLearnerAPIBase` → `homeschoolAPIBase`, value unchanged
  (`"https://self.lextures.com"`).
- **FR-9.** `GetStartedView.swift` MUST use the new localisation keys (lines 63–64) and swap
  `systemImage: "brain.head.profile"` → `"house.fill"` (line 62).
- **FR-10.** The file-header comment at `GetStartedView.swift:3` MUST read "choose homeschool vs
  school".
- **FR-11.** `MobileRoleKind.selfLearner` (`Core/Routing/MobileDestinations.swift:10`) MUST be renamed
  to `.homeschool`. It is a `String`-raw enum with no current reader; the PR MUST confirm by grep that
  nothing decodes it from the server or from disk before renaming, and MUST pin the raw value if
  anything does.

### Android

- **FR-12.** `EnvironmentStore.Kind.SelfLearner("selfLearner")` → `Homeschool("selfLearner")` — the
  `storageValue` argument is already explicit, so only the case name changes.
- **FR-13.** `EnvironmentStore.selectSelfLearner()` → `selectHomeschool()`; call site
  `GetStartedScreen.kt:76` and the `onSelfLearner` parameter (lines 75, 102, 132) follow.
- **FR-14.** `SchoolCodeLogic.SELF_LEARNER_API_BASE` → `HOMESCHOOL_API_BASE`, value unchanged.
- **FR-15.** `GetStartedScreen.kt` MUST use `R.string.auth_getStarted_homeschool*` (lines 130–131) and
  swap `Icons.Default.Psychology` → `Icons.Default.Home` (line 129 / import at line 19).
- **FR-16.** `MobileRoleKind.SelfLearner` (`core/navigation/MobileDestinations.kt:23`) → `Homeschool`,
  with the same no-reader confirmation as FR-11.

### Tests

- **FR-17.** `clients/ios/LexturesTests/SchoolCodeLogicTests.swift` MUST rename
  `testSelfLearnerBase` / `testSelectSelfLearner` and MUST add an assertion that
  `EnvironmentStore.Kind.homeschool.rawValue == "selfLearner"`.
- **FR-18.** `clients/android/app/src/test/.../SchoolCodeLogicTest.kt:40` MUST reference
  `HOMESCHOOL_API_BASE` and MUST add `assertEquals("selfLearner", Kind.Homeschool.storageValue)`.
- **FR-19.** Both platforms MUST add a **forward-compatibility** test: writing the legacy raw value
  `"selfLearner"` into the store and re-reading it MUST yield the homeschool kind and the
  `self.lextures.com` base — i.e. an app upgraded in place keeps its environment.

## 6. Non-Functional Requirements

- **Performance** — no runtime cost; string catalogue size unchanged (keys renamed, not added).
- **Security** — none. No credential, token, or network-config behaviour changes.
- **Privacy & Compliance** — no new data collected or stored.
- **Accessibility** — the card is an `accessibilityElement(children: .combine)` on iOS; the combined
  label must still read title-then-description. Both new strings MUST fit two lines at the largest
  Dynamic Type / `fontScale = 2.0` setting without truncation on a 375 pt-wide device. The icon is
  decorative (`contentDescription = null` / `.accessibilityHidden`) and stays so.
- **Scalability** — n/a.
- **Reliability** — **the upgrade path is the reliability story**: FR-6/FR-12 + FR-19 are what keep
  existing installs pointed at the right API base.
- **Observability** — none required. If mobile onboarding analytics are added later, use the HS.1
  `homeschool` program value.
- **Maintainability** — generated resources stay generated; the sync script is the only writer.
- **Internationalization** — RTL check for `ar` on the get-started card; `en-XA` pseudo-locale
  regenerated if it is still present in `clients/mobile/locales/` at implementation time.
- **Backward compatibility** — persisted values pinned; no key removed from the locale bundles that
  an older binary would request (older binaries ship their own compiled resources, so no runtime
  coupling).

## 7. Acceptance Criteria

- **AC-1.** *Given* a fresh install of either app, *When* the get-started screen renders, *Then* the
  first card reads "Homeschool" with the homeschool description and a house icon.
- **AC-2.** *Given* device language `es`, `fr`, or `ar`, *When* the card renders, *Then* the reviewed
  translation shows — not English, not the old *self-taught* wording.
- **AC-3.** *Given* an app installed **before** this change with the self-learner environment
  selected, *When* the user updates and launches, *Then* the app skips the chooser and still resolves
  `https://self.lextures.com` (verified by FR-19's test and by a manual upgrade run).
- **AC-4.** *Given* `python3 scripts/sync-mobile-locales.py` is run on a clean tree, *Then*
  `git diff --quiet -- clients/ios/.../Localizable.xcstrings clients/android/app/src/main/res` and
  `scripts/check-mobile-i18n.sh` both pass.
- **AC-5.** *Given* the school card, *When* tapped, *Then* the school-code step, validation errors,
  preview host, and reserved-code behaviour are byte-for-byte unchanged.
- **AC-6.** *Given* the login screen on either platform, *Then* the footer link reads "Change school or
  homeschool account" and still returns to the chooser.
- **AC-7.** *Given* `rg -i 'self.?learn' clients/ios clients/android` excluding the pinned raw-value
  lines, *Then* there are zero matches; the pinned lines are allowlisted in HS.1 with a
  `# DO NOT RENAME` comment.
- **AC-8.** *Given* `.github/workflows/ci-ios.yml` and `ci-android.yml`, *Then* both are green.

## 8. Data Model

No server-side schema change. Two **client-local persisted** values, both deliberately unchanged:

| Store | Key | Value | Change |
|---|---|---|---|
| iOS `UserDefaults` | `lextures.environment.kind` | `"selfLearner"` \| `"school"` | **none** (raw value pinned, FR-6) |
| iOS `UserDefaults` | `lextures.environment.apiBaseURL` | `https://self.lextures.com` | none |
| Android `SharedPreferences` (`lextures_environment`) | `kind` | `"selfLearner"` \| `"school"` | **none** (FR-12) |
| Android `SharedPreferences` | `apiBaseURL` | `https://self.lextures.com` | none |

No migration, no backfill — the point is that there is nothing to migrate.

## 9. API Surface

None. Neither app calls a new endpoint; the API base resolution is unchanged.

## 10. UI / UX

**Modified screens** — Get started (chooser step) on iOS and Android; the login screen's
change-environment link on both.

**Key flows**

1. First launch → chooser → **Homeschool** → `EnvironmentStore.selectHomeschool()` → login.
2. First launch → chooser → School → school-code step → `selectSchool(code)` → login *(unchanged)*.
3. Login → "Change school or homeschool account" → back to the chooser.
4. Upgrade from a pre-rename build → **no chooser** → straight to the previously selected
   environment.

**States** — the chooser has no loading/error state (both cards are local); the school-code step's
states are untouched.

**Responsive** — iOS card is `maxWidth: 520` centred; Android uses `AuthScreenContainer`. Both must be
checked at `fontScale = 2.0` and on a small phone (iPhone SE / 360 dp).

**Accessibility annotations** — iOS: `pathCard` keeps `.accessibilityElement(children: .combine)`, so
the combined label becomes "Homeschool, I'm homeschooling, studying for a certification, or learning
on my own schedule."; icon stays decorative. Android: the `Row` is `clickable`, icon
`contentDescription = null`; ensure the click target stays ≥ 48 dp.

**Copy & i18n keys** — `auth.getStarted.homeschoolTitle`, `auth.getStarted.homeschoolDescription`,
`auth.getStarted.changeEnvironment`, all in `clients/mobile/locales/*.json`.

## 11. AI / ML Considerations

Not AI-touching.

## 12. Integration Points

- Internal: `scripts/sync-mobile-locales.py` (generator), `scripts/check-mobile-i18n.sh` (parity
  gate), `clients/ios/Lextures/Core/Config/*`, `clients/android/.../core/config/*`,
  `clients/{ios,android}` auth features, both unit-test targets.
- External: none. (SF Symbols `house.fill` and Material `Icons.Default.Home` are already available;
  no new dependency.)
- CI: `.github/workflows/ci-ios.yml`, `.github/workflows/ci-android.yml`.

## 13. Dependencies & Sequencing

- Must ship after: [HS.1](HS.1-terminology-copy-deck-and-guardrails.md) — specifically the
  **reviewed** `es`/`fr`/`ar` cells; this plan cannot merge with `draft` translations.
- Independent of HS.2/HS.3/HS.5; the apps read no rebranded server string on this screen.
- Must ship before: [HS.6](HS.6-docs-compliance-and-e2e-metadata.md).
- Shared infra: none. Ships on the next normal app-store release train.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Swift enum rename silently changes the `UserDefaults` raw value → every installed app returns to the chooser and loses its API base | **M** | **H** | FR-6 pins `= "selfLearner"`; FR-17/FR-19 assert it; manual upgrade-in-place test in §16; the line is allowlisted in HS.1 so the guard cannot pressure someone into "fixing" it |
| `MobileRoleKind` raw value is decoded somewhere not found by grep | L | M | FR-11/FR-16 require an explicit no-reader confirmation in the PR; pin the raw value if any reader exists |
| Generated resources hand-edited → `check-mobile-i18n.sh` fails on `main` | M | M | FR-4/FR-5 + AC-4; the check regenerates and diffs |
| Machine translations ship and read badly to native speakers | M | M | HS.1 gate: no `draft` cell may merge |
| New description overflows at large Dynamic Type | M | L | §6 constraint + screenshot test at `fontScale = 2.0` |
| App Store screenshots still show "Self-learner" | M | L | Add a store-asset refresh item to the release checklist |

## 15. Rollout Plan

- Feature flag: none. Gating a first-run label behind a remote flag adds a network dependency to the
  pre-auth path.
- Sequencing: locale JSON → run sync script → platform code + tests → CI green → merge → next release
  train (both stores together, so the two platforms do not disagree in public).
- Dogfood: TestFlight / internal track build, exercised on (a) a fresh install and (b) an
  **upgrade over a pre-rename build with the self-learner environment already chosen**.
- GA criteria: AC-1…AC-8 green plus the manual upgrade check.
- Rollback: revert the PR and ship a patch build. Because no persisted value changed, a downgrade or
  revert is a no-op for user state.

## 16. Test Plan

- **Unit** — iOS `LexturesTests/SchoolCodeLogicTests.swift`: renamed tests, raw-value pin assertion,
  legacy-value read-back (FR-17, FR-19). Android `SchoolCodeLogicTest.kt`: same three (FR-18, FR-19).
- **Integration** — run `scripts/check-mobile-i18n.sh` locally and in CI; assert a clean regeneration
  diff.
- **End-to-end** — manual on device: fresh install → Homeschool → login; upgrade-in-place → no
  chooser; school path unchanged.
- **Security** — n/a (no auth behaviour change), but confirm the change-environment link still clears
  only environment state, not tokens, exactly as today.
- **Accessibility** — VoiceOver and TalkBack pass over both cards; verify combined labels, 48 dp
  targets, and `fontScale = 2.0` layout; RTL check under Arabic.
- **Performance / load** — n/a.
- **Manual exploratory** — chooser in all shipped locales × light/dark × small/large phone; confirm
  the school-code step is visually identical to the previous build.

## 17. Documentation & Training

- End-user docs: if the help centre shows a get-started screenshot, reshoot it.
- Admin/instructor docs: none.
- API reference: none.
- Internal: add "regenerate mobile locales, never hand-edit" to the PR checklist (it already exists
  as a script; make it explicit for this PR). Add store screenshots to the release checklist.

## 18. Open Questions

1. `Icons.Default.Home` vs `Icons.Default.Cottage` on Android, and `house.fill` vs
   `house.and.flag.fill` on iOS — which pair reads best next to `graduationcap.fill`/`School`?
2. Is `en-XA` still a shipped pseudo-locale for both platforms? iOS dropped it recently
   (commit `d2a16dc`) while `clients/mobile/locales/en-XA.json` and `values-en-rXA/` still exist —
   confirm the intended set before regenerating, so the sync does not resurrect it.
3. Does anything decode `MobileRoleKind`'s raw value from the server (FR-11/FR-16)? Must be answered
   in-PR, not assumed.
4. Do we want a one-time in-app notice for existing self-learner users explaining the rename, or is
   silent correct given nothing about their account changes? (Recommend silent.)
5. Who owns the App Store / Play Store listing copy if it mentions self-learners?

## 19. References

- Existing files this work touches: `clients/mobile/locales/{en,es,fr,ar,en-XA}.json`,
  `clients/ios/Lextures/Features/Auth/{GetStartedView,LoginView}.swift`,
  `clients/ios/Lextures/Core/Config/{EnvironmentStore,SchoolCodeLogic}.swift`,
  `clients/ios/Lextures/Core/Routing/MobileDestinations.swift`,
  `clients/ios/LexturesTests/SchoolCodeLogicTests.swift`,
  `clients/android/app/src/main/kotlin/com/lextures/android/features/auth/{GetStartedScreen,LoginScreen}.kt`,
  `clients/android/app/src/main/kotlin/com/lextures/android/core/config/{EnvironmentStore,SchoolCodeLogic}.kt`,
  `clients/android/app/src/main/kotlin/com/lextures/android/core/navigation/MobileDestinations.kt`,
  `clients/android/app/src/test/kotlin/com/lextures/android/core/config/SchoolCodeLogicTest.kt`,
  plus the generated `Localizable.xcstrings` and `res/values*/strings.xml`.
- External standards: Apple HIG (SF Symbols, Dynamic Type), Material 3 (touch target ≥ 48 dp),
  WCAG 2.1 AA (SC 1.4.4 Resize Text).
- Related plans: [HS.1](HS.1-terminology-copy-deck-and-guardrails.md),
  [M0.4 mobile i18n](../../completed/mobile/) (locale generation contract),
  [MOB.9 — native sign in with Apple/Google](../../completed/mobile/MOB.9-native-sign-in-with-apple-google.md)
  (shipped; it touches the same two auth screens, so re-read its final layout before moving the
  path cards).
