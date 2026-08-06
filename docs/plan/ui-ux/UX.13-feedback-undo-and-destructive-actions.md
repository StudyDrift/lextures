# UX.13 — Feedback, Undo and Destructive Actions

> Implementation plan. Source: [audit.md](audit.md) §6 G-13.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | UX.13 |
| **Section** | UI/UX — Interaction |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | THIN — 364 hand-rolled error banners; `toastWithUndo` exists, 1 importer |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Web + Product Design |
| **Depends on** | UX.2, UX.12 |
| **Unblocks** | UX.11 (gradebook undo is a hard requirement there) |

---

## 1. Problem Statement

Feedback about the result of an action is delivered three different ways. **364
files hand-roll a `useState<string | null>` error banner** with per-file styling
and copy. `lib/lms-toast.ts` — which exports `toast`, `toastSaveOk`,
`toastMutationError` and, critically, **`toastWithUndo`** — has **one importer**,
despite `<LmsToaster />` being mounted in `main.tsx`. Meanwhile `useConfirm` is
adopted consistently at 41 sites, which is genuinely good, but per **R-20** the
friction of a confirmation should be proportional to damage and reversibility —
and several of those 41 guard actions that are perfectly reversible. The net
effect: users get inconsistent, sometimes unnecessary, sometimes absent feedback,
and the one pattern that would let us remove friction safely (**R-21**: act now,
offer undo) is implemented and unused.

## 2. Goals

- Establish **one feedback system** — toast for transient results, inline for
  contextual/persistent, dialog for blocking decisions — with a rule for choosing.
- Adopt **undo as the default** for reversible destructive actions, reserving
  confirmation for the genuinely irreversible (**R-20/R-21/R-22**).
- Standardise destructive-action copy so it names the object and the consequence.
- Make optimistic UI the default for small mutations (**R-23**).

## 3. Non-Goals

- Form field validation — that is [UX.6](UX.6-form-and-validation-system.md).
- Loading/empty/error *states* — that is [UX.12](UX.12-loading-empty-error-offline-states.md).
- Notifications (the bell drawer) — a distinct system, out of scope.
- Replacing `sonner` unless UX.2 §18 Q4 decides otherwise.

## 4. Personas & User Stories

- **As an instructor**, I want to archive a module and be told it happened, with a
  chance to undo, instead of confirming first.
- **As an instructor**, I want a real confirmation before deleting an assignment
  with 30 submissions, telling me exactly what will be lost.
- **As any user**, I want to know a save succeeded without hunting for a green box.
- **As any user**, I want a failed action to tell me what failed and let me retry
  without losing my work.
- **As a screen-reader user**, I want results announced without my focus being
  stolen.
- **As an engineer**, I want one function to call so that I stop inventing a banner
  per file.

## 5. Functional Requirements

### The feedback decision rule

- **FR-1.** The system MUST document and enforce this selection rule:

  | Situation | Mechanism |
  |---|---|
  | Action succeeded, user has moved on | **Toast** (transient, 4–6 s) |
  | Action succeeded and is reversible | **Toast + Undo** |
  | Result must persist and is contextual to a region | **Inline** (`InlineAlert` in place) |
  | Action failed and blocks progress | **Inline at the point of failure** + retry |
  | Decision required before proceeding | **Dialog** |
  | Irreversible, high blast radius | **AlertDialog** with typed/explicit confirmation |

- **FR-2.** A lint rule MUST forbid hand-rolled success/error banner state
  (`useState<string | null>` used for feedback) in new code, with a ratcheting
  allowlist for the existing 364.

### Toast

- **FR-3.** Toasts MUST be provided by one host (`<LmsToaster />`, already mounted)
  and one API (`lib/lms-toast.ts`).
- **FR-4.** Toasts MUST support: success, error, info, loading→resolved
  transitions, action buttons, and undo.
- **FR-5.** Toasts MUST NOT be the only carrier of information the user needs to
  act on later. Errors requiring action MUST also render inline.
- **FR-6.** Toasts MUST be announced politely, MUST NOT steal focus, MUST be
  keyboard-dismissible, and MUST be reachable by keyboard to press their action
  (an undo the keyboard cannot reach is not an undo).
- **FR-7.** Toasts MUST pause their timer on hover and on focus, and MUST respect
  `prefers-reduced-motion`.
- **FR-8.** Toasts MUST NOT obscure a focused element
  ([UX.5](UX.5-wcag-2.2-aa-conformance-uplift.md) FR-10).
- **FR-9.** Concurrent toasts MUST be stacked and capped (suggested 3), with
  overflow collapsed into a count.

### Undo

- **FR-10.** Every **reversible** destructive action MUST use toast + undo instead
  of a confirmation dialog.
- **FR-11.** The undo window MUST be visible (a countdown or progress indicator)
  and MUST be at least 6 seconds.
- **FR-12.** Undo MUST be **server-backed** — an `undoToken` returned by the
  mutation, single-use and expiring — not a client-side replay of an inverse
  operation.
- **FR-13.** When the undo window expires, the affordance MUST disappear; the
  action MUST NOT appear reversible when it is not.
- **FR-14.** A failed undo MUST be surfaced explicitly, never silently swallowed.
- **FR-15.** For bulk operations, undo MUST restore the **whole batch**, and the
  toast MUST state the count ("12 submissions excused — Undo").

### Confirmation

- **FR-16.** Confirmation dialogs MUST be reserved for the irreversible or
  high-blast-radius. Each of the current 41 `useConfirm` sites MUST be
  re-classified against FR-1; reversible ones MUST convert to undo.
- **FR-17.** Confirmation copy MUST name the **object** and the **consequence**
  ("Delete *Quiz 3*? 28 student submissions and their grades will be permanently
  removed."), and the confirm button MUST be **verb-specific** ("Delete quiz"),
  never "OK" or "Yes" (**R-22**).
- **FR-18.** For the highest blast radius (delete a course, delete an organisation,
  submit final grades to SIS), the dialog MUST require typing the object's name.
- **FR-19.** The destructive action MUST be the non-default; focus MUST land on
  Cancel.
- **FR-20.** Confirmation dialogs MUST state what **cannot** be undone, explicitly.

### Optimistic UI

- **FR-21.** Small, high-frequency mutations (toggle, reorder, star, mark complete,
  cell edit) MUST be optimistic (**R-23**): apply immediately, reconcile on
  response, revert with an explanation on failure.
- **FR-22.** An optimistic revert MUST be visible and announced — never a silent
  snap-back.
- **FR-23.** Optimistic updates MUST NOT be used where the server may legitimately
  transform the value (grade rounding, name normalisation) unless the transformed
  value is reconciled visibly.

## 6. Non-Functional Requirements

- **Performance** — Toast host MUST add ≤4 KB gzip (sonner is already a
  dependency). Optimistic updates MUST make perceived latency for the covered
  actions ≤50 ms.
- **Security** — `undoToken` MUST be opaque, single-use, expiring, and bound to the
  actor and resource. Undo MUST re-check authorisation — a user whose permission was
  revoked mid-window MUST NOT be able to undo. Undo of a destructive action MUST be
  audit-logged as its own event.
- **Privacy & Compliance** — Undo of an action affecting education records MUST
  appear in the audit trail so the record history is complete
  (`../standards/S09-ferpa-hardening.md`).
- **Accessibility** — FR-6/FR-7/FR-8 are the acceptance bar. Toasts are a
  `role="status"` region; the undo button is genuinely focusable; an error toast
  requiring action is duplicated inline (FR-5) so it cannot be missed.
- **Scalability** — Undo support is a server capability per action type; the set
  grows by declaration, not by bespoke client code.
- **Reliability** — Undo MUST be idempotent. A double-press MUST not double-undo.
- **Observability** — Emit `toast_shown` (kind), `undo_offered`, `undo_used`,
  `undo_expired`, `undo_failed`, `confirm_shown`, `confirm_cancelled`,
  `optimistic_reverted`. **`confirm_cancelled` rate is the direct signal that a
  confirmation is unnecessary friction; `undo_used` validates the FR-10 policy.**
- **Maintainability** — One toast module, one confirm hook, one inline alert
  component.
- **Internationalization** — All feedback copy from i18n keys with ICU
  pluralisation (batch counts); toasts positioned correctly in RTL.
- **Backward compatibility** — Converting a confirmation to undo changes muscle
  memory. Conversions MUST be listed in release notes and rolled out behind a flag.

## 7. Acceptance Criteria

- **AC-1.** *Given* the codebase, *When* the feedback lint runs, *Then* new
  hand-rolled banner state is **0** and the existing allowlist only decreases.
- **AC-2.** *Given* a reversible destructive action, *When* performed, *Then* it
  executes immediately and a toast with a working Undo and a visible countdown
  appears.
- **AC-3.** *Given* an undo is pressed, *When* it succeeds, *Then* the prior state
  is restored and the restoration is announced.
- **AC-4.** *Given* the undo window expires, *When* it does, *Then* the affordance
  disappears and pressing it afterwards is impossible.
- **AC-5.** *Given* an undo fails, *When* it does, *Then* the failure is shown
  explicitly with the reason.
- **AC-6.** *Given* an irreversible action, *When* triggered, *Then* a dialog names
  the object and consequence, the confirm button is verb-specific, and focus is on
  Cancel.
- **AC-7.** *Given* the highest-blast-radius actions, *When* triggered, *Then* the
  user must type the object's name to proceed.
- **AC-8.** *Given* the 41 existing `useConfirm` sites, *When* re-classified,
  *Then* every one is documented as either irreversible (dialog retained) or
  reversible (converted to undo).
- **AC-9.** *Given* a bulk action on 12 rows, *When* undone, *Then* all 12 are
  restored and the toast stated the count.
- **AC-10.** *Given* keyboard-only operation, *When* a toast with Undo appears,
  *Then* the user can reach and activate Undo without a mouse and without losing
  their place.
- **AC-11.** *Given* a screen reader, *When* a toast appears, *Then* it is
  announced politely and focus is not moved.
- **AC-12.** *Given* an optimistic update fails, *When* it reverts, *Then* the
  revert is visible, explained and announced.
- **AC-13.** *Given* an undo of an education-record change, *When* performed,
  *Then* both the original action and the undo appear in the audit log.
- **AC-14.** *Given* a user whose permission is revoked during the undo window,
  *When* they press Undo, *Then* it is rejected with an explanation.

## 8. Data Model

```sql
-- server/migrations/NNN_undo_tokens.sql
CREATE TABLE undo_tokens (
  token        text        PRIMARY KEY,             -- opaque, high-entropy
  actor_id     uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  action_type  text        NOT NULL,                -- e.g. 'gradebook.bulk_set'
  payload      jsonb       NOT NULL,                -- inverse operation description
  expires_at   timestamptz NOT NULL,
  consumed_at  timestamptz,
  created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX undo_tokens_expiry_idx ON undo_tokens (expires_at) WHERE consumed_at IS NULL;
```

- **Retention** — a scheduled job (the repo already has a job scheduler) purges
  rows past `expires_at + 1 hour`. Tokens are short-lived by design and are not
  education records.
- **Backfill** — none.
- `payload` MUST contain only what is needed to invert the action, never a copy of
  the affected records beyond identifiers and prior values.

## 9. API Surface

```ts
// Mutations that support undo return a token alongside their normal response.
type UndoableResponse<T> = T & {
  undo?: { token: string; expiresAt: string; description: string }
}

// POST /api/v1/undo                                   (auth: token's actor)
type UndoRequest  = { token: string }
type UndoResponse = { undone: true; description: string }
// 410 Gone     — expired or already consumed
// 403 Forbidden — actor no longer authorised for the underlying resource
```

- A single generic endpoint keeps clients simple; the server dispatches on
  `action_type`.
- Undo MUST be idempotent: a second call with a consumed token returns `410`, not
  a second undo.
- No WebSocket events. Undo shares the rate limit of the originating action.
- **OpenAPI** — the endpoint and the `UndoableResponse` envelope MUST be
  documented; `make openapi-check` passes.

## 10. UI / UX

- **New pages** — none. New UI: toast stack with countdown, typed-confirmation
  dialog variant.
- **Modified pages** — the 364 files with hand-rolled banners, migrated by
  directory; the 41 `useConfirm` sites, re-classified.
- **Key user flows**
  1. Instructor archives a module → it disappears immediately → "Module archived —
     Undo (6)" → they press Undo → it returns.
  2. Instructor deletes a quiz with submissions → dialog: "Delete *Quiz 3*? 28
     submissions and their grades will be permanently removed. This cannot be
     undone." → button "Delete quiz" → Cancel focused by default.
  3. Admin deletes an organisation → must type the org name.
  4. A save fails → inline error at the point of failure with retry, plus a toast.
- **States** — toast: entering, showing, paused (hover/focus), action-pressed,
  resolving, exiting. Confirm: idle, typing-required-unsatisfied, submitting,
  failed.
- **Mobile/responsive** — toasts anchor bottom on small viewports, above any
  sticky bottom bar; undo remains tappable at ≥44px.
- **Accessibility annotations** — toast region `role="status"` `aria-live="polite"`
  and `aria-atomic`; undo is a real `<button>` in the tab order; countdown has a
  text equivalent; timers pause on focus; confirm dialog uses the UX.2 `AlertDialog`
  with focus on Cancel.
- **Copy & i18n** — shared catalogue under `common.feedback.*` and
  `common.destructive.*` at parity across four locales. **Copy rules:** state what
  happened in past tense ("Module archived"), name the object, use ICU plurals for
  counts, and in confirmations state explicitly what cannot be undone.

## 11. AI / ML Considerations

Not AI-touching. One constraint on AI surfaces as consumers: AI-initiated changes
(grading-agent suggestions, adaptive content rewrites) MUST use the same undo
affordance and MUST be visibly attributed to the agent in both the toast and the
audit log, so a human can always reverse an automated change.

## 12. Integration Points

- **External** — `sonner` (already a dependency).
- **Internal**
  - `clients/web/src/lib/lms-toast.ts` — becomes load-bearing
  - `clients/web/src/components/lms-toaster.tsx`, `main.tsx`
  - `clients/web/src/components/use-confirm.tsx` — re-classified, rebuilt on
    UX.2 `AlertDialog`
  - New: `components/ui/inline-alert.tsx`
  - `clients/web/src/lib/api.ts` — surfaces `undo` envelopes
  - `clients/web/src/pages/lms/gradebook/**` — the primary consumer
    ([UX.11](UX.11-data-table-and-gradebook-system.md) FR-16)
  - `server/internal/httpserver` — undo endpoint and per-action inverse handlers
  - Existing audit-log subsystem — undo events
  - Existing job scheduler — token purge
- **Events** — feedback telemetry into `server/internal/telemetry`.

## 13. Dependencies & Sequencing

- **Must ship after** — [UX.2](../../completed/ui-ux/UX.2-core-component-library-and-adoption-ratchet.md)
  (`AlertDialog`, `Toast`), [UX.12](UX.12-loading-empty-error-offline-states.md)
  (error classification and inline error surfaces).
- **Must ship before** — [UX.11](UX.11-data-table-and-gradebook-system.md), which
  depends on gradebook undo.
- **Shared infra** — audit log; job scheduler for token purge.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Removing a confirmation lets users destroy something they meant to keep | M | **H** | FR-10 applies **only** to server-verified-reversible actions; each conversion requires an implemented, tested inverse operation before the dialog is removed. AC-8 re-classification is documented and reviewed |
| Undo token expires while the user is still reading the toast | M | M | ≥6 s minimum with visible countdown; timer pauses on hover and focus; expiry removes the affordance rather than failing on press |
| Server-side inverse operations are substantial work per action type | **H** | M | Start with a small, high-value set (archive, bulk grade, excuse, hide/post, reorder). Actions without an inverse simply keep their confirmation |
| Toasts are missed by users who look elsewhere | **H** | M | FR-5: anything requiring action is also inline. Toasts carry results, never obligations |
| Optimistic updates make the UI lie under poor connectivity | M | M | FR-22 visible, announced reverts; optimistic scope limited to small, idempotent mutations |
| 364-file banner migration stalls | H | L | Ratcheting lint; migrate opportunistically alongside other directory work rather than as a dedicated push |
| Undo bypasses an authorisation change | L | **H** | FR/AC-14: undo re-checks authorisation server-side, always |

## 15. Rollout Plan

- **Feature flag** — `ffUndoActions`, default off, gating each confirmation→undo
  conversion. The toast/inline consolidation ships unflagged.
- **Sequencing**
  1. `InlineAlert` + toast API consolidation + copy catalogue + gallery entries.
  2. Lint as warning; allowlist generated for the 364 banners.
  3. Undo token infrastructure server-side (endpoint, storage, purge job, audit).
  4. Inverse operations for the initial action set.
  5. Re-classification of the 41 `useConfirm` sites — **documented and reviewed**
     before any conversion.
  6. Conversions behind `ffUndoActions`, starting with the lowest-stakes
     (archive/reorder), watching `undo_used` and support signal.
  7. Typed confirmation for the highest-blast-radius actions.
  8. Banner migration by directory; lint flipped to error.
- **Dogfood** — internal org, 2 weeks per conversion tier.
- **GA criteria** — AC-1…AC-14 green; `undo_used` rate demonstrates the affordance
  is discoverable; **zero reports of unintended data loss** attributable to a
  removed confirmation.
- **Rollback** — `ffUndoActions` off restores confirmations. Undo infrastructure is
  additive and remains.

## 16. Test Plan

- **Unit** — toast stacking, cap and overflow; timer pause on hover/focus;
  countdown; undo token lifecycle (issue, consume, expire, double-consume);
  optimistic apply/revert; confirm dialog typed-name gating; copy pluralisation.
- **Integration** — undo endpoint: valid, expired (410), consumed (410),
  wrong actor (403), permission revoked mid-window (403); audit entries for both
  action and undo; bulk undo restores the full batch; purge job removes expired
  tokens.
- **End-to-end** — Playwright: archive → undo → verify restoration; bulk excuse →
  undo → verify all 12; delete-with-submissions → dialog copy assertions → cancel →
  nothing changed; typed confirmation for org delete; optimistic toggle failing and
  reverting visibly.
- **Security** — token entropy and unguessability; replay; cross-user undo attempt;
  undo after permission revocation; audit completeness; ensure `payload` carries no
  record content beyond identifiers and prior values.
- **Accessibility** — axe on toast and dialog variants × 4 themes; screen-reader
  scripts: hear a toast without focus moving; reach and press Undo by keyboard;
  hear an optimistic revert; navigate a typed-confirmation dialog. Verify toasts do
  not obscure focus (UX.5 FR-10).
- **Performance / load** — optimistic perceived latency ≤50 ms; toast host bundle
  delta; undo endpoint p95.
- **Manual exploratory** — QA checklist per converted action: perform, undo,
  perform-and-let-expire, perform-and-undo-twice, perform offline.

## 17. Documentation & Training

- **End-user** — help-centre: "Undoing a change" — which actions can be undone and
  for how long.
- **Instructor / admin** — a list of irreversible actions and what they affect,
  cross-linked from the destructive dialogs themselves.
- **Engineer** — `docs/guides/feedback-and-destructive-actions.md`: the FR-1
  selection rule, how to add undo support to an action (inverse operation + token +
  audit), the destructive-copy rules, when optimistic UI is and is not appropriate.
- **API reference** — OpenAPI for `/api/v1/undo` and the `UndoableResponse`
  envelope.
- **Runbook** — "A user says undo didn't work": reading undo token state and audit
  entries.
- **Copy guide** — destructive-action microcopy patterns added to the terminology
  guidance.

## 18. Open Questions

1. Which of the 41 `useConfirm` sites are genuinely irreversible? Requires a
   one-pass review with product; this is AC-8 and should happen early since it
   sizes the whole plan.
2. What is the initial set of undoable actions? *Proposed: module archive, module
   reorder, gradebook bulk set, excuse, hide/post grades, discussion post delete,
   file move.*
3. Undo window length — 6 s, or longer for bulk operations? *Recommendation: 6 s
   for single, 10 s for bulk, with the countdown always visible.*
4. Should there be a persistent "Recently undone" or activity log surface for
   instructors, beyond the audit log?
5. Do we keep `sonner` or move toasts into the UX.2 library
   (UX.2 §18 Q4)? *Recommendation: keep sonner, wrap it — it already handles
   stacking, pausing and reduced motion.*
6. Does undo of an AI-initiated change need distinct treatment in the audit log for
   the EU AI Act record-keeping obligations in `../standards/S13-eu-ai-act-high-risk.md`?

## 19. References

- Existing files: `clients/web/src/lib/lms-toast.ts` (`toastWithUndo` —
  implemented, 1 importer), `clients/web/src/components/lms-toaster.tsx`,
  `clients/web/src/main.tsx`, `clients/web/src/components/use-confirm.tsx`
  (41 call sites), `clients/web/src/lib/errors.ts`
- Research: [research.md](research.md) R-20, R-21, R-22, R-23
- Audit: [audit.md](audit.md) G-13, G-9
- External: [NN/g — Confirmation Dialogs Can Prevent User Errors (If Not Overused)](https://www.nngroup.com/articles/confirmation-dialog/),
  [Confirm or undo? — Josh Wayne](https://joshwayne.com/posts/confirm-or-undo/),
  [SaaS Destructive Actions & Confirmation UX Patterns](https://www.saasui.design/blog/saas-destructive-actions-confirmation-ux-patterns)
- Related plans: [UX.2](../../completed/ui-ux/UX.2-core-component-library-and-adoption-ratchet.md),
  [UX.11](UX.11-data-table-and-gradebook-system.md),
  [UX.12](UX.12-loading-empty-error-offline-states.md),
  `../standards/S09-ferpa-hardening.md`
