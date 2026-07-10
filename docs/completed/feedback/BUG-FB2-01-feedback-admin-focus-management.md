# BUG-FB2-01 — Feedback admin detail view never manages keyboard focus

> Bug report. Follows [../../plan/_TEMPLATE.md](../../plan/_TEMPLATE.md), bug-tuned. Source feature: [FB2 — Web Feedback Admin Page](./FB2-web-feedback-admin.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | BUG-FB2-01 |
| **Section** | Feedback — In-App Feedback & Admin Review |
| **Severity** | MINOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | FIXED — open/close focus management wired; covered by `feedback-admin-panel.test.tsx` |
| **Estimated effort** | XS (≤1d) |
| **Owner (proposed)** | Platform (owns FB2) |
| **Depends on** | — |
| **Unblocks** | FB2 accessibility conformance (WCAG 2.1 AA focus-order/AC) |
| **Component** | `clients/web/src/components/settings/feedback-admin-panel.tsx` |
| **Intentional?** | **No.** FB2 §6, §10 and §16 explicitly require focus to move into the detail on open and back to the originating row on close. The code *attempts* both and both attempts silently no-op. |

---

## 1. Problem Statement

The Feedback admin detail view (`/settings/feedback/:id`) is supposed to move keyboard focus **into** the detail panel when a submission is opened, and **back to the originating table row** when it is closed. Neither happens. A keyboard-only or screen-reader admin who activates a row lands with focus stranded on `document.body` (or the previous element), with no announcement that a new view opened; on close, focus is dropped rather than returned to where they were. This is a WCAG 2.1 AA focus-management defect on an admin surface the plan explicitly promised would be accessible.

Both focus operations are *written* in the code but are wired so they can never fire — this is a latent bug, not a missing feature.

## 2. Goals

- Focus moves to the detail view's first interactive control (the "Back to list" button) once the submission has loaded.
- Focus returns to the exact table row that was activated when the detail view is closed, after the list has re-rendered.
- Behaviour verified by an automated test so the regression cannot silently return.

## 3. Non-Goals

- No redesign of the detail view (drawer vs. route is already decided — route, FB2 §18.1).
- No change to the list/filter/pagination logic.
- No server/API change (this is a pure client focus-management fix).

## 4. Personas & User Stories

- **As a keyboard-only platform admin**, when I press Enter on a feedback row, I want focus to land inside the opened submission so I can read and triage it without hunting with Tab.
- **As a screen-reader admin (NVDA/VoiceOver)**, when I open a submission, I want focus moved into the new view so the reading context follows the navigation.
- **As the same admin**, when I go back, I want focus returned to the row I came from so I keep my place in the queue.

## 5. Functional Requirements

- **FR-1.** On opening a submission, once `detail` has loaded the panel MUST programmatically move focus to the "Back to list" control (or another sensible first focus target inside the detail view).
- **FR-2.** The open-focus effect MUST fire on the transition *to loaded content*, not only on component mount, because at mount the component renders `null`/loading and the focus target does not yet exist in the DOM.
- **FR-3.** On closing the detail view, the panel MUST return focus to the DOM node of the row that was activated, resolved **after** the list has re-rendered its rows (the previously captured `<tr>` reference is detached by then and MUST NOT be reused directly).
- **FR-4.** Focus restoration MUST tolerate the list's post-navigation refetch (`listLoading` briefly replaces the table with a loading paragraph), i.e. it MUST target the row once the rows are actually mounted.
- **FR-5.** The fix MUST NOT introduce focus stealing while the user is typing elsewhere or when the panel is not the active surface.

## 6. Non-Functional Requirements

- **Accessibility** — WCAG 2.1 AA §2.4.3 (Focus Order) and the FB2 a11y NFR; parity with other settings panels' focus behaviour.
- **Performance** — negligible; no extra network calls.
- **Reliability** — deterministic focus regardless of fetch latency (fast cache hit vs. slow load).
- **Maintainability** — prefer resolving the row by a stable attribute (e.g. `data-feedback-row-id`) queried from the DOM on return, rather than holding a raw element ref across an unmount.
- **Internationalization / RTL** — unaffected (focus, not layout).
- **Backward compatibility** — none required; behaviour-only.

## 7. Acceptance Criteria

- **AC-1.** *Given* an admin activates a feedback row with the keyboard, *When* the detail finishes loading, *Then* focus is on the "Back to list" button (assert `document.activeElement`).
- **AC-2.** *Given* the detail view is open, *When* the admin activates "Back to list", *Then* after the list re-renders focus is on the same row element that was originally activated.
- **AC-3.** *Given* a slow detail fetch, *When* the response arrives after the initial `null`/loading renders, *Then* focus still moves into the detail (the effect is not a one-shot mount effect).
- **AC-4.** *Given* automated a11y/focus tests, *Then* AC-1..AC-3 are covered so the regression is caught in CI.

## 8. Data Model

- None. Client-only behaviour.

## 9. API Surface

- None. No request/response changes.

## 10. UI / UX

- Unchanged visuals. The only observable change is where the keyboard caret / screen-reader cursor lands on open and close.
- **Open:** focus → "Back to list" button (first control in reading order of the loaded detail).
- **Close:** focus → the activated row (`<tr tabIndex={0}>`).

## 11. AI / ML Considerations

- None.

## 12. Integration Points

- `clients/web/src/components/settings/feedback-admin-panel.tsx` only.
  - `FeedbackDetailView` open-focus effect — `feedback-admin-panel.tsx:323`.
  - Early `return null` before the focus target is rendered — `feedback-admin-panel.tsx:384`.
  - Row-ref capture and close handler — `openDetail`/`backToList` and `lastOpenedRowRef` at `feedback-admin-panel.tsx:538`, `:608`, `:613`.
  - `FeedbackListRow` (add a stable `data-feedback-row-id={item.id}` to enable DOM re-query) — `feedback-admin-panel.tsx:235`.

## 13. Dependencies & Sequencing

- Independent; ships anytime. No migration, no flag.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Focus fights React Router's own scroll/focus restoration | L | L | Move focus inside an effect keyed on loaded `detail`, and on return use `requestAnimationFrame`/query after rows mount |
| Row no longer present after refetch (deleted/filtered out) | L | L | Fall back to focusing the table/heading if the target row id is absent |
| Over-eager focus steals input focus | L | M | Only move focus on the open/close transitions, not on every render |

## 15. Rollout Plan

- No feature flag; behaviour-only fix under the existing `ff_feedback`-gated surface.
- Ship with the automated focus tests; verify with axe + manual keyboard/VoiceOver pass.
- Rollback: revert the component change (no data implications).

## 16. Test Plan

- **Unit / Integration (Vitest + Testing Library)** — render `FeedbackAdminPanel` at `/settings/feedback/:id` with a mocked detail; assert `document.activeElement` is the Back button after load (AC-1, AC-3); simulate Back and assert focus returns to the row (AC-2).
- **Accessibility** — axe on list + detail; manual NVDA/VoiceOver check that opening a submission moves the reading cursor.
- **Manual exploratory** — keyboard-only: Tab to a row → Enter → confirm focus in detail → Back → confirm focus on the row; repeat with a throttled network to exercise the async path.

## 17. Documentation & Training

- None user-facing. Add a note to the FB2 accessibility checklist that detail open/close focus is covered by an automated test.

## 18. Root Cause & Evidence

**Root cause — open direction (focus never enters the detail):**

`FeedbackDetailView` focuses the back button in a **mount-only** effect:

```tsx
// feedback-admin-panel.tsx:318,323
const backRef = useRef<HTMLButtonElement>(null)
useEffect(() => {
  backRef.current?.focus()
}, [])                       // runs once, on mount
```

But on mount the component has no data yet. The parent renders the detail branch the moment the route param appears, *before* `loadDetail` runs:

```tsx
// feedback-admin-panel.tsx:646
if (detailId) {
  return <FeedbackDetailView detail={detail} loading={detailLoading} ... />
}
```

At that first render `detail === null` and `detailLoading === false`, so the component hits:

```tsx
// feedback-admin-panel.tsx:384
if (!detail) return null
```

The back button (which holds `backRef`) is therefore **not in the DOM** when the `[]` effect fires, so `backRef.current` is `null` and `.focus()` is a no-op. When the fetch later resolves and the button finally renders, the mount effect does **not** re-run (empty dependency array). Net result: focus never moves into the detail. The effect should depend on the loaded `detail` (e.g. `}, [detail])` guarded by `detail && !loading`).

**Root cause — close direction (focus not returned to the row):**

```tsx
// feedback-admin-panel.tsx:538,608,613
const lastOpenedRowRef = useRef<HTMLTableRowElement | null>(null)
function openDetail(id, rowEl) { if (rowEl) lastOpenedRowRef.current = rowEl; navigate(`/settings/feedback/${id}`) }
function backToList() {
  navigate('/settings/feedback')
  window.requestAnimationFrame(() => { lastOpenedRowRef.current?.focus() })
}
```

Opening a submission unmounts the list entirely (the parent returns `<FeedbackDetailView>` instead of the table), which **destroys** the captured `<tr>`. `lastOpenedRowRef.current` now points at a **detached** DOM node; calling `.focus()` on it does nothing. Worse, on return the list re-mounts and immediately refetches, so during the `requestAnimationFrame` tick `listLoading` is `true` and the table isn't even rendered yet — so there is no row to focus regardless. The fix is to remember the row **id**, and after the list renders, query the live DOM (`[data-feedback-row-id="…"]`) and focus that, with a fallback if the row is gone.

**Contradicts the plan (proves it is a bug, not a decision):**

- FB2 §6 (Accessibility NFR): *"focus management into/out of the detail panel."*
- FB2 §10 (Accessibility): *"focus moves into detail on open and back to the row on close."*
- FB2 §16 (Test Plan → Accessibility): *"keyboard row activation; focus management."*

### Related lower-severity findings (feedback epic, triage separately)

1. **`org_id` sourced from the optional JWT claim, not the authoritative lookup.** `handlePostFeedback` derives `orgID` from `auth.UserFromRequest(...).OrgID` (`server/internal/httpserver/feedback_http.go:112`), a claim documented as *"empty when absent from token (legacy JWTs)"* (`server/internal/auth/jwt.go:43`). The same request path already resolves the authoritative org via `organization.OrgIDForUser` inside `validateMeUser` (`server/internal/httpserver/me.go:71`). Normal login/refresh does populate the claim (`orgJWTFieldsForUser`, `refresh.go:164`), so impact is low today, but a legacy/orgless-claim token yields a `NULL org_id` for a user who *does* belong to an org — weakening the `(org_id, status, created_at)` index and the planned org-scoped admin view (FB0 §18.2 / FB2 §18.2). Diverges from FB0 FR-3 ("persist … org_id from the authenticated session"). Suggest resolving org from `OrgIDForUser`. Severity: MINOR.

2. **Per-user rate limiter is consumed before body validation.** `feedbackRateLimited` runs before the body is read/validated (`feedback_http.go:80` precedes the `ReadAll`/`ValidateMessage` at `:84`), so malformed or oversized requests (which return `400`) still consume a token against the 10/10 min quota. Abuse-defensible, but a burst of client-side validation failures or retries can lock a legitimate user out of submitting real feedback. Severity: MINOR.

## 19. References

- Buggy file: `clients/web/src/components/settings/feedback-admin-panel.tsx` (`:318`, `:323`, `:384`, `:538`, `:608`, `:613`, `:646`).
- Source feature plan: [FB2 — Web Feedback Admin Page](./FB2-web-feedback-admin.md) (§6, §10, §16, §18.1).
- Foundation contract: [FB0 — Feedback Data Model, Submission & Admin API](./FB0-feedback-foundation-api.md).
- Related lower-severity finding files: `server/internal/httpserver/feedback_http.go`, `server/internal/httpserver/me.go`, `server/internal/auth/jwt.go`.
- Standards: WCAG 2.1 AA §2.4.3 Focus Order; §2.4.7 Focus Visible.
