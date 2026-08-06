# UX.6 — Form and Validation System

> Implementation plan. Source: [audit.md](audit.md) §3 G-8.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | UX.6 |
| **Section** | UI/UX — Accessibility & Interaction |
| **Severity** | MAJOR (legal) |
| **Markets** | K12 / HE / HS |
| **Status (today)** | MISSING — no form abstraction; 929 inputs, 10 `aria-invalid` |
| **Estimated effort** | L (1–2mo) |
| **Owner (proposed)** | Design Systems + Accessibility |
| **Depends on** | UX.2 |
| **Unblocks** | UX.8, UX.13; VPAT SC 3.3.x attestation |

---

## 1. Problem Statement

The product contains **929 `<input>` elements** and exactly **10 `aria-invalid`
attributes**. Validation errors are rendered as free-floating text near a field
with no programmatic relationship to it, so a screen-reader user tabbing into an
errored field is told nothing. There are 396 `placeholder=` attributes against
1,159 `<label>`s for 929 inputs, indicating placeholder-as-label in an unknown
number of places. Four files use `zod`; every other form validates bespoke. Errors
surface through 364 hand-rolled `useState<string | null>` banners with per-file
styling and per-file copy. This fails **WCAG 3.3.1 Error Identification**,
**3.3.2 Labels or Instructions** and **3.3.3 Error Suggestion**, and it makes
every form in the product a fresh opportunity to get validation wrong.

## 2. Goals

- Ship a **`Field` abstraction** that makes correct label/description/error
  association the only way to build a form.
- Take `aria-invalid` coverage from 10 to **100% of invalid fields**, and
  guarantee every input has a programmatic label.
- Standardise validation *timing* and *copy* so errors are consistent, specific
  and actionable.
- Give every form a **submission error summary** that moves focus and lists
  errors as links to fields.
- Prevent accidental data loss on navigation away from a dirty form.

## 3. Non-Goals

- Redesigning the settings or course-settings information architecture — that is
  [UX.8](UX.8-settings-and-admin-ia-unification.md).
- Server-side validation logic — unchanged; this plan standardises how server
  errors are *presented* and mapped to fields.
- The rich content editors (TipTap, CodeMirror) as *fields* — they are wrapped,
  not rebuilt.
- Quiz answering UI, which has its own interaction model.

## 4. Personas & User Stories

- **As a screen-reader user**, I want to be told which field is wrong and why when
  I focus it, so that I can fix it without hunting.
- **As a keyboard user**, I want a failed submission to move my focus to a summary
  of what went wrong, so that I am not left at the bottom of a long form.
- **As an instructor filling out course settings**, I want to know a value is
  invalid before I press Save, not after I lose my place.
- **As any user**, I want to be warned before navigating away from unsaved changes.
- **As a user on a flaky connection**, I want a failed save to preserve everything
  I typed and offer a retry.
- **As an engineer**, I want validation to be declared once and shared with the
  server contract, so that client and server cannot disagree.

## 5. Functional Requirements

- **FR-1.** The system MUST provide a `Field` component that owns the
  relationship between label, control, description and error, generating and
  wiring `id`, `aria-describedby`, `aria-invalid` and `aria-required`
  automatically.
- **FR-2.** Every input, textarea, select, combobox, checkbox group, radio group,
  switch and file input in the product MUST be rendered through `Field` or a
  composition of it. A lint rule MUST forbid bare `<input>`/`<select>`/
  `<textarea>` outside `components/ui/`.
- **FR-3.** Every field MUST have a **visible label**. Placeholders MUST NOT be
  used as labels. A lint rule MUST flag a `placeholder` on a control with no
  associated `<label>`.
- **FR-4.** Required fields MUST be marked visually **and** with `aria-required`,
  and the marking convention MUST be explained once per form.
- **FR-5.** Validation timing MUST follow this contract:
  - **On blur** — validate the field the user just left, but only if it has been
    touched.
  - **On change** — re-validate a field **only** once it is already in an error
    state (so errors clear as the user fixes them, but do not appear mid-typing).
  - **On submit** — validate everything.
- **FR-6.** On failed submit, the form MUST render an **error summary** at the top
  with `role="alert"`, move focus to it, and list each error as a link that moves
  focus to the offending field.
- **FR-7.** Error messages MUST state what is wrong **and** how to fix it
  (SC 3.3.3), MUST be specific to the field, and MUST come from i18n keys.
- **FR-8.** Schemas MUST be declared once with **zod** and MUST be the single
  source of validation for the client. Where the server exposes a schema
  (via OpenAPI), the client schema MUST be checked against it in CI.
- **FR-9.** Server validation errors MUST be returned in a **field-addressable**
  shape and mapped onto the corresponding fields, not shown only as a page banner.
- **FR-10.** Forms MUST track dirty state and MUST warn before navigation away
  from unsaved changes. The existing `components/ui/unsaved-changes-banner.tsx`
  MUST be adopted rather than duplicated.
- **FR-11.** A failed save MUST preserve all entered values and offer retry
  without re-entry (this also serves WCAG 3.3.7 Redundant Entry, see
  [UX.5](UX.5-wcag-2.2-aa-conformance-uplift.md)).
- **FR-12.** Fields MUST set correct `autocomplete` tokens (WCAG 1.3.5 Identify
  Input Purpose) for name, email, address, phone, and credential fields.
- **FR-13.** Async validation (e.g. "is this course code taken?") MUST debounce,
  announce its pending state via `aria-busy`, and MUST NOT block submit.
- **FR-14.** Multi-step forms MUST expose progress, allow backward navigation
  without data loss, and MUST NOT ask for anything already supplied.
- **FR-15.** Destructive form actions MUST route through the UX.13 confirm/undo
  policy, not bespoke handling.

## 6. Non-Functional Requirements

- **Performance** — `Field` MUST not re-render the whole form on each keystroke;
  per-field subscription is required. Form state library + zod MUST add ≤10 KB
  gzip to the entry bundle (zod is already a dependency). INP on keystroke ≤200 ms
  p75 on the largest form (course settings, 1,884 lines today).
- **Security** — Client validation is UX only; the server remains authoritative.
  Server error messages MUST NOT be rendered as HTML. Field-addressable errors
  MUST NOT leak the existence of records the user cannot see.
- **Privacy & Compliance** — Delivers WCAG 2.1 SC 1.3.5, 3.3.1, 3.3.2, 3.3.3,
  3.3.4 (Error Prevention for legal/financial/data submissions) and 2.2 SC 3.3.7.
- **Accessibility** — Correctness is the acceptance bar; see AC-1…AC-6.
- **Scalability** — Adding a form means declaring a schema and composing fields;
  no new accessibility work.
- **Reliability** — Form state MUST survive a transient network failure and a
  component remount during save.
- **Observability** — CI emits `fields_without_label`,
  `inputs_outside_field_component`, `forms_without_error_summary`. Runtime emits a
  `form_validation_failed` metric with form id and field (no values) to find
  chronically confusing fields.
- **Maintainability** — One schema per form, colocated; no validation logic in
  JSX.
- **Internationalization** — All error copy from i18n keys with ICU pluralisation;
  error summary reading order correct in RTL.
- **Backward compatibility** — Migration is per-form and behaviour-preserving,
  except that errors become *more* visible and *more* announced.

## 7. Acceptance Criteria

- **AC-1.** *Given* any form in the product, *When* a field is invalid, *Then* it
  carries `aria-invalid="true"` and its error is referenced by
  `aria-describedby`.
- **AC-2.** *Given* the codebase, *When* scanned, *Then* every input control has a
  programmatically associated visible label, and `placeholder`-as-label
  occurrences are **0**.
- **AC-3.** *Given* a form with three errors, *When* the user submits, *Then*
  focus moves to an error summary listing all three, and activating an entry moves
  focus to that field.
- **AC-4.** *Given* an errored field, *When* the user corrects it, *Then* the
  error clears on change; *When* a valid untouched field is being typed into,
  *Then* no error appears mid-typing.
- **AC-5.** *Given* a screen reader, *When* the user tabs into an errored field,
  *Then* the label, requirement, current value and error are announced.
- **AC-6.** *Given* the top 40 forms, *When* axe runs, *Then* there are **0**
  violations in the `cat.forms` category.
- **AC-7.** *Given* a server returns a field-addressable 422, *When* it is
  received, *Then* the error is attached to the correct field, not only to a page
  banner.
- **AC-8.** *Given* a dirty form, *When* the user navigates away, *Then* they are
  warned and can cancel.
- **AC-9.** *Given* a save fails with a network error, *When* the user retries,
  *Then* all values are intact and no re-entry is needed.
- **AC-10.** *Given* the lint rules, *When* CI runs, *Then* bare form controls
  outside `components/ui/` number **0** and the allowlist is empty.
- **AC-11.** *Given* a client zod schema and the corresponding OpenAPI schema,
  *When* the contract check runs, *Then* any divergence in required fields or
  types fails CI.
- **AC-12.** *Given* the largest form, *When* a keystroke is measured, *Then* INP
  ≤200 ms at p75.

## 8. Data Model

None. UX.6 is client-side. No tables, columns, enums, indexes, migrations or
backfill.

## 9. API Surface

No new routes. One **cross-cutting contract change** to existing write endpoints:

```ts
// Standard validation error envelope, returned with HTTP 422.
type ValidationErrorResponse = {
  error: 'validation_failed'
  message: string                    // human-readable summary, i18n key preferred
  fields: {
    path: string                     // dot/bracket path, e.g. "sections[0].code"
    code: string                     // stable machine code, e.g. "already_taken"
    message: string                  // fallback human text
    params?: Record<string, unknown> // for client-side ICU interpolation
  }[]
}
```

- Adoption is **incremental**: endpoints migrate as their forms migrate. The
  client MUST tolerate the legacy shape and fall back to a page-level banner.
- No WebSocket changes. Existing rate limits apply.
- **OpenAPI** — the envelope MUST be a shared component schema referenced by every
  migrated write endpoint; `make openapi-check` must pass.

## 10. UI / UX

- **New pages** — none. The UX.2 gallery gains a "Forms" section demonstrating the
  full states matrix.
- **Modified pages** — every form surface; highest density in
  `components/settings/` (48 panels), `pages/lms/course-settings.tsx` (1,884
  lines), `pages/lms/course-create.tsx`, `pages/admin/**`, auth pages.
- **Key user flows**
  1. Fill a form → blur an invalid field → see a specific, actionable error →
     correct it → error clears.
  2. Submit with errors → focus moves to summary → jump to first error → fix →
     resubmit.
  3. Edit settings → navigate away → be warned → cancel → values intact.
  4. Save fails offline → banner explains → retry when back online → values intact.
- **States** — per field: default, focused, filled, disabled, read-only,
  required, invalid, warning, pending (async), success. Per form: pristine,
  dirty, submitting, submitted-with-errors, saved.
- **Mobile/responsive** — single-column below `md`; correct `inputmode` and
  keyboard types; the error summary is sticky-adjacent so it is not lost above the
  fold.
- **Accessibility annotations** — error summary is `role="alert"` and receives
  focus; field errors use `aria-describedby` (not `aria-errormessage`, for AT
  support breadth); async pending uses `aria-busy`; success announced politely.
- **Copy & i18n** — a shared error-copy catalogue under `common.validation.*`
  (e.g. `required`, `tooShort`, `invalidEmail`, `alreadyTaken`,
  `mustBeAfterStartDate`) at parity across all four locales. Copy guidance:
  say what is wrong and what to do, never "Invalid input".

## 11. AI / ML Considerations

Not AI-touching. *(Adjacent, out of scope: AI-assisted error explanation for
complex validation such as grading-scheme conflicts. Noted for UX.16 follow-up;
would require the same PII-redaction and cost-budget treatment as other AI
surfaces.)*

## 12. Integration Points

- **External** — `zod` (already a dependency, v4). A form-state library
  (see §18 Q1) — candidates: React Hook Form, TanStack Form, or a thin in-house
  hook.
- **Internal**
  - `clients/web/src/components/ui/` — `Field`, `Input`, `Select`, `Combobox`,
    `Checkbox`, `Radio`, `Switch`, `Textarea`, `FileInput`, `Fieldset`,
    `ErrorSummary`
  - `clients/web/src/components/ui/unsaved-changes-banner.tsx` — adopted
  - `clients/web/src/lib/errors.ts` (`readApiErrorMessage`) — extended to parse
    the 422 envelope
  - `clients/web/src/lib/generated/openapi-types.ts` — schema contract check
  - `clients/web/src/components/settings/**` (48 panels)
  - `clients/web/src/pages/lms/course-settings.tsx`, `course-create.tsx`
  - `server/internal/httpserver` — 422 envelope
  - `clients/web/eslint-rules/` — new lint rules
- **Events** — `form_validation_failed` telemetry into the existing
  `server/internal/telemetry` metrics layer.

## 13. Dependencies & Sequencing

- **Must ship after** — [UX.2](UX.2-core-component-library-and-adoption-ratchet.md)
  (form controls are library components).
- **Should ship alongside** — [UX.4](UX.4-aria-widget-and-focus-management-remediation.md)
  (comboboxes and selects share ARIA contracts).
- **Must ship before** — [UX.8](UX.8-settings-and-admin-ia-unification.md)
  (settings restructure should land on the new form system, not the old one).
- **Shared infra** — none.
- **Internal sequencing**: `Field` + summary + zod integration → pilot on
  `components/settings/account-settings-view.tsx` → the other 47 settings panels →
  course settings → admin → auth → long tail.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| 929 inputs is a very large migration that stalls | **H** | **H** | Ratcheting lint (FR-2) prevents regression; migrate by directory with named owners; settings panels first (48 files, ~40% of the value) |
| Adding a form-state library increases bundle and learning cost | M | M | Evaluate a thin in-house hook first (§18 Q1); measure against the 10 KB budget; whatever is chosen must be the *only* one |
| Client and server validation disagree, producing errors users cannot resolve | M | **H** | FR-8 + AC-11 contract check against OpenAPI; server remains authoritative and its 422 always wins |
| Validation-on-blur feels aggressive on long forms | M | M | FR-5 contract is deliberately conservative (touched-only, clear-on-change); validate in dogfood and tune once, globally |
| Server 422 envelope adoption lags the client migration | **H** | M | Client tolerates the legacy shape and degrades to a page banner; envelope adoption tracked per endpoint |
| Error copy becomes inconsistent again across 40 forms | M | M | Shared `common.validation.*` catalogue; lint flags inline English error strings |

## 15. Rollout Plan

- **Feature flag** — none for the component migration. `ffValidationEnvelope`
  gates the *server* 422 envelope per endpoint group so backend and frontend can
  land independently.
- **Sequencing**
  1. `Field`, `ErrorSummary`, zod integration, gallery entries, lint as warning.
  2. Pilot: account settings. Measure axe, INP, and support-ticket signal.
  3. `components/settings/` (48 panels).
  4. Course settings + course create (the two largest forms).
  5. Admin pages.
  6. Auth pages (coordinate with UX.5 FR-16/FR-17).
  7. Server 422 envelope per endpoint group behind `ffValidationEnvelope`.
  8. Lint flipped to error; allowlists deleted.
- **Dogfood** — internal org; validation-timing feedback gathered explicitly.
- **GA criteria** — AC-1…AC-12 green; no increase in form-abandonment telemetry.
- **Rollback** — per-form revert; envelope rollback by flag.

## 16. Test Plan

- **Unit** — `Field` id/aria wiring; validation timing state machine (touched,
  blurred, errored, corrected); zod schema resolution; server-error→field mapping
  including unknown paths; dirty tracking; autocomplete tokens.
- **Integration** — full form lifecycle against MSW mocks: success, 422 envelope,
  legacy error shape, network failure + retry, concurrent save conflict.
- **End-to-end** — Playwright on the 10 highest-traffic forms: submit-with-errors
  → summary → jump-to-field → fix → success; navigate-away warning; offline save
  then retry.
- **Security** — server error messages rendered as text not HTML; 422 field paths
  cannot enumerate inaccessible records; authz unchanged by the envelope.
- **Accessibility** — axe `cat.forms` on the top 40 forms (AC-6); screen-reader
  scripts (NVDA, VoiceOver) for: focus an errored field; submit and traverse the
  summary; async validation pending and resolved; required-field announcement.
- **Performance / load** — keystroke INP on the largest form (AC-12); bundle
  delta gate; re-render count assertion (a keystroke must not re-render siblings).
- **Manual exploratory** — QA checklist per migrated directory covering the full
  per-field and per-form states matrix in §10.

## 17. Documentation & Training

- **End-user** — none (behaviour should be self-evident).
- **Admin / instructor** — none.
- **Engineer** — `docs/guides/forms.md`: declare a schema, compose fields, wire
  the summary, map server errors, the validation-timing contract, and the error-
  copy rules.
- **API reference** — the `ValidationErrorResponse` component schema in OpenAPI,
  plus a note in the API guide that all write endpoints converge on it.
- **Runbook** — "Form lint failed: bare input" and "Client/server schema
  divergence".
- **Copy guide** — extend the existing terminology guidance with validation
  microcopy rules (say what is wrong + how to fix; never blame the user).

## 18. Open Questions

1. Form-state library: React Hook Form, TanStack Form, or a thin in-house hook
   over zod? *Recommendation: evaluate an in-house hook first — our forms are
   mostly flat and the bundle budget is tight — but do not hand-roll if arrays and
   nested objects (sections, rubrics, grading schemes) prove painful. Decide with
   a 2-day spike on course settings, the hardest case.*
2. Does the Go server already emit a consistent validation shape we can adopt
   rather than invent? Needs a survey of `server/internal/httpserver` error paths.
3. Should `aria-errormessage` be used instead of `aria-describedby`?
   *Recommendation: `aria-describedby` for now — AT support is broader; revisit.*
4. Do the rich editors (TipTap syllabus blocks, CodeMirror activities) participate
   in form validation, or validate on their own terms?
5. What is the retention policy for the `form_validation_failed` metric, and does
   it need a DPIA entry given it identifies user + form? Coordinate with
   `../standards/S06-dpia-pia-algorithmic-impact.md`.

## 19. References

- Existing files: `clients/web/src/components/settings/` (48 panels, especially
  `account-settings-view.tsx`, `people-panel.tsx`, `roles-permissions-panel.tsx`),
  `clients/web/src/pages/lms/course-settings.tsx`,
  `clients/web/src/pages/lms/course-create.tsx`,
  `clients/web/src/components/ui/unsaved-changes-banner.tsx`,
  `clients/web/src/lib/errors.ts`,
  `clients/web/src/lib/generated/openapi-types.ts`
- Research: [research.md](research.md) R-18, R-19, R-35
- Audit: [audit.md](audit.md) G-8, G-13
- External: [WCAG 2.2 SC 3.3.1–3.3.4, 1.3.5](https://www.w3.org/TR/WCAG22/),
  [WAI Forms Tutorial](https://www.w3.org/WAI/tutorials/forms/)
- Related plans: [UX.2](UX.2-core-component-library-and-adoption-ratchet.md),
  [UX.4](UX.4-aria-widget-and-focus-management-remediation.md),
  [UX.5](UX.5-wcag-2.2-aa-conformance-uplift.md),
  [UX.8](UX.8-settings-and-admin-ia-unification.md),
  [UX.13](UX.13-feedback-undo-and-destructive-actions.md)
