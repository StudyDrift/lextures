# W03 — In-App Dialogs & Notifications (Replace Native `alert`/`confirm`/`prompt`)

> Implementation plan. Source: web market-readiness scan (2026-07-06).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | W03 |
| **Section** | Web / UX Consistency & Accessibility |
| **Severity** | MINOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | PARTIAL |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | Frontend platform team |
| **Depends on** | none |
| **Unblocks** | Consistent error UX; cleaner a11y story for buyers |

---

## 1. Problem Statement

The app already ships a global toast system (`sonner` via `components/lms-toaster.tsx`) and a styled,
focus-trapped `ConfirmDialog` (`components/confirm-dialog.tsx`) — but ~**67 native
`window.alert`/`confirm`/`prompt` calls** across ~30 files bypass them. The worst offender is the core
grading surface: `components/annotation/assignment-annotation-workbench.tsx` has **17** native calls
(errors on save, submit, upload, reveal-identities, retry-scan). Native dialogs are unstyled and
theme-blind (a jarring OS chrome popup in an otherwise dark-mode grading flow), block the main thread,
are not translatable (they will never localize under W01), and interrupt screen-reader flow in ways the
app's own live-region-backed toasts do not. For instructors doing high-volume grading, this reads as
unfinished.

## 2. Goals

- Route all user-facing success/error feedback through the existing toast system.
- Route all destructive/irreversible confirmations through `ConfirmDialog` (focus-trapped, keyboard/ESC,
  labelled danger action).
- Replace the two `prompt()` inputs (passkey name, email-template link URL) with real inline inputs/dialogs.
- Leave zero `window.alert/confirm/prompt` in `pages/**` and `components/**` (enforced by lint).

## 3. Non-Goals

- Redesigning the toast or confirm components themselves.
- Changing *what* actions confirm (the set of destructive actions is unchanged — only the mechanism).
- Server-side changes.

## 4. Personas & User Stories

- **As an instructor grading submissions**, I want save/submit errors to appear as in-app toasts I can
  read and retry, not an OS alert that blocks the page.
- **As a screen-reader user**, I want confirmations and errors announced through the app's live regions
  in a consistent way.
- **As a Spanish-language user (W01)**, I want error and confirm copy translated — impossible with
  native dialogs.
- **As an admin deleting a term or revoking a key**, I want a clearly-labelled danger confirm dialog, not
  a bare browser `confirm()`.

## 5. Functional Requirements

- **FR-1.** All `window.alert(...)` / `window.alert` error and info paths MUST be replaced with
  `toast.error(...)` / `toast.success(...)` / `toast(...)` using the existing `sonner` instance.
- **FR-2.** All `window.confirm(...)` / `globalThis.confirm(...)` gating destructive actions MUST be
  replaced with `ConfirmDialog` (title, body, confirm/cancel labels, danger styling for destructive ops).
- **FR-3.** The two `prompt(...)` usages (passkey name in `mfa-factors-panel.tsx`; link URL in
  `EmailTemplateEditor.tsx`) MUST become inline form fields or a small input dialog.
- **FR-4.** Confirm dialogs for destructive actions MUST trap focus, close on ESC, return focus to the
  trigger, and mark the confirm button as the danger action.
- **FR-5.** A lint rule MUST forbid `window.alert|confirm|prompt` (and the `globalThis.`/bare variants)
  in `clients/web/src/pages/**` and `clients/web/src/components/**`.
- **FR-6.** Toast copy and confirm copy MUST be i18n-ready (`t()`), so W01 can translate them.

## 6. Non-Functional Requirements

- **Performance** — No main-thread block (native dialogs block; toasts/dialogs do not). No bundle
  growth (components already shipped).
- **Security** — Confirmations for irreversible actions (revoke token, delete term, sign out session,
  reveal blind-grading identities) remain mandatory; the dialog must not be dismissible-as-confirm.
- **Privacy & Compliance** — n/a (mechanism change).
- **Accessibility** — WCAG 2.1: dialog role/labelling, focus management, ESC to cancel; toasts use the
  existing polite live region. Net a11y improvement over native dialogs.
- **Scalability** — n/a.
- **Reliability** — Error toasts must surface the actionable message (preserve current `e.message`
  strings); no silent failures.
- **Observability** — Optional: count destructive-confirm accept/cancel for high-risk actions.
- **Maintainability** — One `useConfirm()`/`toast` pattern documented; the lint rule prevents regressions.
- **Internationalization** — Enables W01 for these strings (native dialogs cannot be localized).
- **Backward compatibility** — Same actions, same guardrails; only the presentation changes.

## 7. Acceptance Criteria

- **AC-1.** *Given* a save fails in the grading workbench, *When* the error surfaces, *Then* an in-app
  error toast appears (no OS alert) with the failure message and the page remains interactive.
- **AC-2.** *Given* an admin deletes a term, *When* they trigger it, *Then* a focus-trapped danger
  `ConfirmDialog` appears; ESC cancels; confirm proceeds.
- **AC-3.** *Given* a user adds a passkey, *When* naming it, *Then* an inline input is used (no
  `prompt()`).
- **AC-4.** *Given* CI runs on a PR adding `window.confirm` in `pages/**`, *Then* lint fails.
- **AC-5.** *Given* a screen reader, *When* a toast/confirm fires, *Then* it is announced via the app's
  live region / dialog semantics.

## 8. Data Model

- None.

## 9. API Surface

- None. Uses `sonner`'s `toast` API and the existing `ConfirmDialog` props.

## 10. UI / UX

- **Modified:** grading annotation workbench (17), course files (7), course live/discussions/office-hours,
  library catalog, moderation, several course-settings pages, and settings panels (MFA, access keys,
  service tokens, organizations, terms, account) — full list from the scan below.
- **Flows:** error → toast (auto-dismiss, dismissible, retry where relevant); destructive action →
  ConfirmDialog → toast on completion.
- **States:** toasts stack (max 5, existing config); confirm dialog has loading state on the confirm
  button during async work.
- **Accessibility:** focus trap + return-focus on the dialog; live-region announcement on toasts.
- **Copy & i18n:** wrap all copy in `t()`.

## 11. AI / ML Considerations

- Not applicable.

## 12. Integration Points

- `clients/web/src/components/lms-toaster.tsx` (sonner), `clients/web/src/components/confirm-dialog.tsx`.
- Consumers (highest first): `components/annotation/assignment-annotation-workbench.tsx` (17),
  `pages/lms/course-files-page.tsx` (7), `pages/lms/course-live-page.tsx`,
  `pages/lms/library-catalog-page.tsx`, `pages/lms/course-office-hours-page.tsx`,
  `pages/lms/course-discussions-page.tsx`, `pages/admin/AdminUsers.tsx`,
  `pages/admin/AdminEmailTemplates.tsx`, `components/settings/{mfa-factors,integrations-access-keys,
  admin-service-tokens,organizations,terms-settings,account-settings-view}.tsx`,
  `components/annotation/feedback-media-player.tsx`, and the single-call tail (moderation, webhooks,
  evaluation templates, course blueprint/cross-listing/sections/feed, creator learning paths).
- Lint: `clients/web/eslint-plugin-lextures-i18n.js` sibling rule or a `no-native-dialogs` rule.

## 13. Dependencies & Sequencing

- **Must ship after:** none.
- **Must ship before:** W01 translation of these strings (native dialogs can't be localized).
- **Shared infra:** none.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| A destructive action loses its confirm during migration | M | H | Checklist of every `confirm()` → mapped `ConfirmDialog`; test each. |
| Async confirm double-submits | L | M | Disable confirm button + loading state while pending. |
| Toast missed by user for critical errors | M | M | Use `toast.error` with longer duration / manual dismiss for critical failures. |
| Regression reintroduces native dialogs | M | L | Lint gate (FR-5). |

## 15. Rollout Plan

- **Feature flag:** none — this is a straight refactor, shippable file-by-file.
- **Sequencing:** grading workbench first (highest volume + most visible) → course files → settings →
  tail → enable lint gate once count hits 0.
- **Pilot:** internal dogfood of the grading flow.
- **GA criteria:** zero native dialog calls in `pages/**`/`components/**`; lint gate on.
- **Rollback:** per-file; low risk.

## 16. Test Plan

- **Unit** — ConfirmDialog focus-trap/ESC/return-focus; toast error path renders message.
- **Integration** — grading save/submit error → toast; term delete → confirm → toast.
- **End-to-end** — Playwright: no `window.alert/confirm` dialogs intercepted during the grading and
  admin destructive flows (assert none via `page.on('dialog')`).
- **Security** — each destructive action still requires explicit confirm.
- **Accessibility** — axe + screen-reader on the dialog; live-region announcement on toast.
- **Manual exploratory** — dark mode (native dialogs were the worst there).

## 17. Documentation & Training

- Engineering: "Use `toast` + `ConfirmDialog`, never `window.alert/confirm/prompt`" in the web README /
  contributing guide, with the lint rule reference.

## 18. Open Questions

1. Add a thin `useConfirm()` promise-based wrapper around `ConfirmDialog` to make migration mechanical?
2. Which errors warrant a persistent toast (manual dismiss) vs auto-dismiss?

## 19. References

- `clients/web/src/components/annotation/assignment-annotation-workbench.tsx` (17 native calls).
- `clients/web/src/components/lms-toaster.tsx`, `clients/web/src/components/confirm-dialog.tsx`.
- Full offender list from `grep -rnE "(window\.|globalThis\.)?(alert|confirm|prompt)\("` (2026-07-06 scan).
- Related plans: [W01](W01-i18n-application-coverage.md).
- Standards: WCAG 2.1 SC 2.4.3 (Focus Order), 4.1.2 (Name, Role, Value), 4.1.3 (Status Messages).
