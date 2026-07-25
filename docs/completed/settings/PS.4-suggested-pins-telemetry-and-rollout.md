# PS.4 — Suggested Pins, Discoverability Telemetry & GA Rollout

> Implementation plan (**completed**). Source: authoring-UX gap — closing the loop on "which settings are actually hidden?" and solving the cold-start empty pinned group. Folder overview: [README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | PS.4 |
| **Section** | Pinned Editor Settings |
| **Severity** | MINOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | DONE |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | Web client team + Product analytics |
| **Depends on** | PS.1, PS.2, PS.3 |
| **Unblocks** | — |

---

## 1. Problem Statement

PS.3 ships a pinned group that starts **empty**: the instructors most hurt by buried settings are the
ones least likely to go hunting for a pin affordance, so the feature risks helping only power users who
already know where everything is. Separately, the premise behind this whole folder — "some important
settings are kind of hidden" — is currently an assertion with no measurement behind it. PS.4 gives the
feature a cold-start answer (suggested pins) and gives the product team the evidence loop (which
settings get pinned, searched for, and never touched) that says whether pinning solved the problem or
whether the information architecture itself needs to change.

## 2. Goals

- Give a first-time user a **useful, dismissible set of suggested pins** so the pinned group is not an
  empty box on day one.
- Instrument pin, search, and setting-interaction events well enough to rank which settings are hard
  to find, without collecting anything beyond product-usage data.
- Publish an internal dashboard/queryable metric set that answers "which settings are hidden?" with
  data.
- Take the feature from pilot to **GA**: flip `ff_pinned_settings` on by default, with documented
  criteria and a rollback.
- Feed findings back as a concrete recommendation on default section order and on whether suggested
  pins should become defaults.

## 3. Non-Goals

- No machine-learned personalisation or recommendation model — suggestions are a static, curated list
  per surface, revised from aggregate data by humans.
- No automatic pinning: suggestions are always opt-in, never applied without an explicit action.
- No admin- or org-configurable suggestion sets.
- No cross-user behavioural profiling, and no per-learner data — this is staff-facing authoring
  telemetry only.
- No changes to the pin API, schema, or the pinning UX built in PS.3 (beyond the suggestion strip).

## 4. Personas & User Stories

- **As a first-time instructor**, I want a starting set of pins I can accept or ignore, so that the
  pinned group is useful before I have learned the panel.
- **As an instructor**, I want to dismiss suggestions permanently, so that the product stops nudging me
  once I have my own setup.
- **As a product manager**, I want to know which settings instructors search for but rarely reach, so
  that I can fix the information architecture instead of relying on pinning as a workaround.
- **As a designer**, I want evidence about how many settings people actually pin, so that the 8-pin cap
  and the group's default ordering are informed rather than guessed.
- **As a privacy officer**, I want authoring telemetry to be aggregate, staff-scoped, and free of
  content, so that instrumentation does not create a new compliance surface.
- **As an on-call engineer**, I want the GA flip to be one flag with a documented rollback, so that a
  regression is a one-minute fix.

## 5. Functional Requirements

### Suggested pins

- **FR-1.** The system MUST ship a curated `SUGGESTED_PINS: Record<SettingSurface, string[]>` in the
  settings registry module, referencing only IDs that resolve via `resolveSettingId`.
- **FR-2.** Suggestions MUST render only when the user has **zero** pins for that surface and has not
  dismissed suggestions, replacing PS.3's first-run hint (FR-20 of PS.3) rather than stacking with it.
- **FR-3.** The suggestion strip MUST offer per-suggestion "Pin" actions and a single "Not now"
  dismissal; it MUST NOT offer "pin all" (a bulk action would fill the cap with settings the user never
  chose).
- **FR-4.** Accepting a suggestion MUST use exactly the PS.3 pin path (optimistic + debounced save), so
  suggested and manual pins are indistinguishable once stored.
- **FR-5.** Dismissal MUST persist per surface in `localStorage`
  (`lextures.pinned-settings.suggestions-dismissed.{surface}`) and MUST also be implied by the user
  pinning anything on that surface.
- **FR-6.** Suggestion content MUST be revisable in one file with no migration and no server deploy
  coupling.
- **FR-7.** A suggested ID that no longer resolves MUST be skipped silently, and a registry test MUST
  fail the build if any suggested ID is unresolvable.

### Telemetry

- **FR-8.** The client MUST emit these events (through the existing web telemetry path used by the app,
  not a new vendor): `settings_pin_added`, `settings_pin_removed`, `settings_pin_reordered`,
  `settings_pin_save_failed`, `settings_suggestion_accepted`, `settings_suggestion_dismissed`,
  `settings_search_performed`, `settings_search_zero_results`, `settings_control_changed`.
- **FR-9.** Every event MUST carry `surface` and, where applicable, `setting_id` (registry key) plus a
  coarse `role` dimension (`instructor`, `admin`, `other`); events MUST NOT carry course IDs, item IDs,
  item titles, setting *values*, or free-text search queries.
- **FR-10.** `settings_search_performed` MUST record only a **normalised query hash** and the result
  count — never the raw string — so that a query cannot leak course or student names.
- **FR-11.** Events MUST be debounced/sampled so that a typing burst produces at most one
  `settings_search_performed` per 1 s idle window, and `settings_control_changed` at most one per
  control per 2 s.
- **FR-12.** The server MUST expose Prometheus counters for the API side —
  `pinned_settings_writes_total{surface}`, `pinned_settings_rejects_total{reason}` (defined in PS.2) —
  and PS.4 MUST add `pinned_settings_pins_gauge` (histogram of list length observed on write, bucketed
  0–12).
- **FR-13.** Telemetry MUST be suppressed entirely when `ffPinnedSettings` is off, and MUST respect any
  existing product-analytics opt-out already honoured by the web client.

### Reporting

- **FR-14.** The team MUST produce a queryable breakdown, refreshed at least weekly, of: pins per user
  (distribution), top pinned settings per surface, top searched-then-unmatched queries (by hash bucket
  with a manual decode list of known queries), and settings never pinned/never changed.
- **FR-15.** A written **"hidden settings" review** MUST be produced 4 weeks after GA, recommending
  concrete IA changes (section reordering, promoting a setting out of an accordion, merging sections)
  or explicitly recording that none are needed.

### GA

- **FR-16.** `ff_pinned_settings` MUST default to **true** only after every GA criterion in §15 is met
  and recorded in the rollout ticket.
- **FR-17.** The kill-switch runbook MUST be updated with the GA default and the observed effect of
  flipping the flag off after users have accumulated pins.

## 6. Non-Functional Requirements

- **Performance** — Telemetry MUST be fire-and-forget and MUST NOT block any interaction; batched with
  the app's existing transport. Added payload per editor session < 2 KB. Suggestion strip adds
  < 5 ms to panel render.
- **Security** — No new endpoints. Query hashing uses a non-reversible digest with a per-deployment
  salt so hashes cannot be dictionary-attacked back to student names across tenants.
- **Privacy & Compliance** — Staff-facing product analytics only; no student data, no education
  records, so no FERPA record obligation and no COPPA surface. Event fields are enumerated in FR-9 and
  the enumeration MUST be enforced by a typed event schema (a rogue field cannot be added ad hoc). The
  analytics inventory (RoPA, see `docs/plan/standards/S05`) MUST list these events.
- **Accessibility** — The suggestion strip MUST meet WCAG 2.1 AA: each suggestion is a labelled button
  ("Pin Due date to top"), the strip is a landmark-free `<section aria-labelledby>`, dismissal is
  keyboard reachable, and acceptance announces through PS.3's existing live region.
- **Scalability** — Event volume is bounded by authoring sessions (orders of magnitude below learner
  traffic); no new storage tier required.
- **Reliability** — Telemetry failure MUST never surface to the user or affect pinning; the suggestion
  strip MUST render from static data with no network dependency.
- **Observability** — This plan *is* the observability work; additionally, an alert MUST fire if
  `pinned_settings_rejects_total{reason="shape"}` is non-zero for 15 minutes (indicates a client/server
  contract break).
- **Maintainability** — Suggestions live beside the registry; the event schema lives in one typed
  module so names cannot drift between call sites.
- **Internationalization** — Suggestion strip copy externalised like PS.3's strings. Query
  normalisation MUST be Unicode-aware (NFKC + lowercase) before hashing so non-English queries bucket
  correctly.
- **Backward compatibility** — Additive. Turning telemetry off (or the flag off) returns the product to
  PS.3 behaviour with no data migration.

## 7. Acceptance Criteria

- **AC-1.** *Given* a user with zero quiz pins who has never dismissed suggestions, *When* they open a
  quiz editor with the flag on, *Then* the suggestion strip renders with the curated quiz suggestions
  and PS.3's generic first-run hint does not also render.
- **AC-2.** *Given* the suggestion strip, *When* the user clicks "Pin" on one suggestion, *Then* it is
  pinned via the normal pin path, the strip disappears, and the pin persists across reload.
- **AC-3.** *Given* the suggestion strip, *When* the user clicks "Not now", *Then* the strip never
  returns for that surface on that device, and no pins are created.
- **AC-4.** *Given* a user who already has ≥ 1 pin on a surface, *When* they open that editor, *Then*
  no suggestion strip renders.
- **AC-5.** *Given* a suggested ID that no longer resolves, *When* the strip renders, *Then* that entry
  is skipped and the remaining suggestions render; *And* the registry test fails in CI.
- **AC-6.** *Given* the flag is on, *When* the user pins, unpins, reorders, searches, and changes a
  control, *Then* exactly the events in FR-8 are emitted with only the fields in FR-9.
- **AC-7.** *Given* a search for "Smith essay", *When* `settings_search_performed` is emitted, *Then*
  the payload contains a hash and a result count and does not contain the raw string anywhere.
- **AC-8.** *Given* the user types 12 characters in 3 seconds, *When* telemetry flushes, *Then* at most
  one `settings_search_performed` event is emitted for that burst.
- **AC-9.** *Given* `ffPinnedSettings` is off, *When* the editor is used, *Then* zero pin/suggestion
  telemetry events are emitted.
- **AC-10.** *Given* a `PUT` of a 5-key list, *When* the server records metrics, *Then*
  `pinned_settings_pins_gauge` observes 5 and `pinned_settings_writes_total{surface="quiz"}`
  increments by 1.
- **AC-11.** *Given* a week of pilot traffic, *When* the reporting query runs, *Then* it returns pins
  per user, top pinned settings per surface, and zero-result search buckets without manual data
  wrangling.
- **AC-12.** *Given* every GA criterion in §15 is met, *When* the flag default flips to true, *Then*
  existing users' pins are unaffected and users with no pins see the suggestion strip.
- **AC-13.** *Given* the suggestion strip, *When* `axe` runs and a keyboard-only pass is performed,
  *Then* there are zero violations and every action is reachable and announced.

## 8. Data Model

No schema changes. PS.4 adds:

- **Static client data** — `SUGGESTED_PINS` in `clients/web/src/lib/settings-registry.ts`.
  Initial curation (to be revised from data per FR-15):

  | Surface | Suggested pins (initial) |
  |---|---|
  | quiz | `quiz.scheduling.due-date`, `quiz.attempts-grading.points-worth`, `quiz.presentation.lockdown-mode`, `quiz.attempts-grading.late-policy` |
  | assignment | `assignment.scheduling.due-date`, `assignment.grading.points-worth`, `assignment.grade-posting.policy`, `assignment.academic-integrity.originality-mode` |

- **Local storage** — `lextures.pinned-settings.suggestions-dismissed.{surface}`.
- **Metrics** — `pinned_settings_pins_gauge` (histogram, buckets 0,1,2,3,4,5,6,8,10,12) in
  `server/internal/telemetry/metrics.go`; PS.2's counters reused.
- **Event schema** — typed module `clients/web/src/lib/settings-telemetry.ts`; fields limited to
  `event`, `surface`, `setting_id?`, `role`, `query_hash?`, `result_count?`, `position?`, `pin_count?`.

## 9. API Surface

No new HTTP routes and no changes to PS.2's contract. Telemetry uses the web client's existing
analytics transport; server metrics are exposed on the existing `/metrics` endpoint
(`server/internal/telemetry/metrics.go`). No WebSocket events. No OpenAPI change.

## 10. UI / UX

### New component

- `clients/web/src/components/settings-panel/suggested-pins-strip.tsx` — renders above the accordions
  when eligible (FR-2): a short line of copy plus up to four suggestion chips, each a button, and a
  "Not now" text button.

### Modified

- `pinned-settings-group.tsx` / the panels — mount the strip in place of PS.3's first-run hint when
  suggestions are eligible.
- `use-pinned-settings.ts` — expose `suggestionsEligible` and `dismissSuggestions`.

### Key user flows

1. First open with zero pins → strip appears → user pins one suggestion → strip disappears, Pinned
   group appears with that setting.
2. First open with zero pins → user clicks "Not now" → strip gone permanently for that surface.
3. User with pins → no strip, ever.

### States

- **Loading** — the strip renders only after pins load and resolve to zero (avoids a flash for users
  who do have pins).
- **Empty / error** — if the registry yields no resolvable suggestions, nothing renders.
- **Offline** — the strip is static and still renders; accepting a suggestion follows PS.3's
  save-failure path.

### Mobile / responsive

Chips wrap; the strip is capped at two lines with no horizontal scroll.

### Accessibility annotations

`<section aria-labelledby="suggested-pins-heading">` with a visually-hidden heading; each chip is a
`<button>` named "Pin {label} to top"; "Not now" is a `<button>` named "Dismiss pin suggestions";
acceptance announces through PS.3's live region.

### Copy & i18n keys

`settingsPanel.suggestions.heading` ("Suggested pins"), `.intro` ("Settings other instructors keep at
the top"), `.pinAction` ("Pin {label} to top"), `.dismiss` ("Not now").

## 11. AI / ML Considerations

Not applicable by design (§3). If a future iteration ranks suggestions per user, it would require a new
plan covering model choice, cold-start behaviour, fairness across teaching contexts, and an opt-out —
none of which is in scope here.

## 12. Integration Points

- Internal modules: `clients/web/src/lib/settings-registry.ts` (suggestions),
  new `clients/web/src/lib/settings-telemetry.ts`,
  `clients/web/src/components/settings-panel/*`,
  `server/internal/telemetry/metrics.go`,
  `server/internal/httpserver/pinned_settings_http.go` (metric emission),
  `server/internal/repos/platformconfig/features.go` (GA default flip).
- Analytics: the web client's existing event transport; the team's warehouse/dashboard for FR-14.
- Compliance: analytics inventory entry (`docs/plan/standards/S05-ropa-data-inventory-mapping.md`).
- Runbook: `docs/runbooks/` pinned-settings kill switch (created in PS.2, updated here).

## 13. Dependencies & Sequencing

- Must ship after: **PS.1**, **PS.2**, **PS.3**.
- Must ship before: nothing — PS.4 closes the folder.
- Shared infra needed: existing metrics pipeline and product-analytics transport; no new services.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Suggestions become de facto defaults and homogenise everyone's panel | M | M | Opt-in only, never auto-applied (FR-4); measure the accept rate and drop suggestions if they dominate organic pins |
| Search-query telemetry leaks student or course names | L | H | Hash-only with per-deployment salt (FR-10, §6 Security); typed event schema rejects a raw-query field; review gate before merge |
| Curated suggestions are wrong for a market (K-12 vs HE differ sharply) | M | M | Ship one list, review at 4 weeks (FR-15) with per-market breakdown; per-market lists are a cheap follow-up if the data supports it |
| Telemetry volume or cost surprises | L | L | Sampling/debounce (FR-11); authoring traffic is small relative to learner traffic |
| GA flip regresses editors for orgs that never piloted | L | M | Staged flip by environment with the flag as an instant rollback; alert on editor error rate during the flip window |
| The "hidden settings" review never happens once the feature ships | M | M | FR-15 makes it a deliverable of this plan with a dated owner in the rollout ticket |
| Analytics opt-out users are silently excluded, biasing the data | M | L | Report opt-out share alongside findings so conclusions are read with the right denominator |

## 15. Rollout Plan

- **Feature flag** — `ff_pinned_settings` (shared). PS.4 is the plan that flips its default to
  **true**.
- **Sequencing** — (1) telemetry module + server metric, (2) suggestion strip, (3) enable on internal
  + pilot orgs and collect 2 weeks, (4) review data and revise the curated list, (5) staged GA flip
  (internal → pilot → all), (6) 4-week hidden-settings review (FR-15).
- **Dogfood / pilot** — Same cohorts as PS.3.
- **GA criteria** —
  - PS.3's pilot metric met (≥ 25 % of active authors pinned ≥ 1 setting in two weeks);
  - `pin_save_failed` < 0.5 % of pin attempts;
  - zero open accessibility defects on the pin and suggestion UI;
  - `pinned_settings_rejects_total{reason="shape"}` at zero from the shipped client for 7 days;
  - no regression in editor page error rate or CLS during the pilot;
  - runbook updated and on-call briefed.
- **Comms** — Release-note entry, help-center article (from PS.3), and an admin note that
  `ff_pinned_settings` now defaults on and how to turn it off.
- **Rollback path** — Flip the flag off (instant, pins retained). Suggestions and telemetry are inert
  when the flag is off, so no separate rollback is needed.

## 16. Test Plan

- **Unit** — Suggestion eligibility logic (zero pins × not dismissed × flag on); dismissal persistence;
  unresolvable-suggestion skipping; event-schema validation rejects unknown fields; query
  normalisation + hashing is stable, salted, and non-reversible; debounce windows per FR-11.
- **Integration (component)** — Strip renders/does not render per AC-1/AC-4; accepting a suggestion
  routes through the pin path and hides the strip (AC-2); "Not now" persists (AC-3); no events when the
  flag is off (AC-9).
- **Integration (server)** — `pinned_settings_pins_gauge` observes the submitted list length and
  `pinned_settings_writes_total` increments (AC-10); metrics absent when the flag is off.
- **End-to-end** — Extend `e2e/tests/pinned-settings.spec.ts`: fresh user sees the strip, accepts a
  suggestion, reloads, sees the pin and no strip; a second fresh user dismisses and never sees it again.
- **Security** — Assert no event payload contains a raw query, course id, item id, or setting value
  (network-interception test); confirm the salt is not shipped to the client in a form that enables
  cross-tenant correlation.
- **Accessibility** — `axe` on the strip in light and dark; keyboard-only accept and dismiss;
  screen-reader announcement on accept (AC-13).
- **Performance / load** — Confirm telemetry is non-blocking (interaction latency unchanged with the
  transport stubbed to a 2 s delay); measure added bundle size (< 4 KB gzipped budget).
- **Manual exploratory** — Fresh profile per surface; dismiss on one device and confirm the (expected)
  per-device behaviour; run through a full authoring session and eyeball the emitted event stream
  against FR-8/FR-9.

## 17. Documentation & Training

- End-user docs: extend PS.3's help-center article with a "Suggested pins" section explaining that
  suggestions are optional and dismissible.
- Admin docs: note the GA default of `ff_pinned_settings` and how to disable it org-wide.
- Internal: analytics event dictionary entry for the nine events, their fields, and their retention;
  the FR-15 hidden-settings review published to the product docs space.
- Runbook: update the pinned-settings kill-switch runbook with GA state, the alert on
  `pinned_settings_rejects_total`, and the expected user-visible effect of a flip.
- API reference: no change.

## 18. Open Questions

1. Should suggestions differ by market (K-12 / HE / homeschool) or by the course's grade levels?
   **Proposed:** one list at launch; decide from the 4-week review.
2. Should the dismissal flag be per-user server-side instead of per-device `localStorage`?
   **Proposed:** per-device for now — it is a nudge, not a preference; revisit if users complain about
   re-prompting on a second machine.
3. What is the retention period for authoring telemetry events, and does it inherit the platform's
   general product-analytics retention? Needs the privacy owner's sign-off before GA.
4. Is a decode list for query hashes (mapping known safe queries like "lockdown" to their hash)
   acceptable to privacy, or should zero-result analysis rely only on counts? **Proposed:** a
   pre-registered allowlist of setting-related terms, hashed at build time.
5. If the review (FR-15) concludes the IA itself should change, does that become a PS.5 plan or a
   change request against the existing panels? **Proposed:** PS.5 if it touches more than section
   order.
6. Should the pinned-settings pattern extend to other surfaces (content page, module, course settings)
   once GA data exists? Deliberately deferred to a future plan.

## 19. References

- Existing files: `server/internal/telemetry/metrics.go`,
  `server/internal/repos/platformconfig/features.go`,
  `clients/web/src/components/settings-panel/` (PS.1/PS.3),
  `clients/web/src/lib/settings-registry.ts` (PS.1), `e2e/tests/`,
  `docs/runbooks/adaptive-content-kill-switch.md` (runbook pattern).
- Related plans: [PS.1](PS.1-settings-registry-and-addressable-controls.md),
  [PS.2](PS.2-pinned-settings-data-model-and-api.md),
  [PS.3](PS.3-pin-and-reorder-ux-in-editor-panels.md),
  [S05 — RoPA / data inventory](../../plan/standards/S05-ropa-data-inventory-mapping.md).
- External standards: WCAG 2.1 AA; RFC 2119; GDPR Art. 5(1)(c) data minimisation (applied to event
  fields).
