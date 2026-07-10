# FB1 — Web "Share Feedback" Top-Nav Button & Form

> Implementation plan. Source: Product request — in-app "Share Feedback" mechanism (2026-07-10). Follows [../_TEMPLATE.md](../_TEMPLATE.md). Consumes [FB0](./FB0-feedback-foundation-api.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | FB1 |
| **Section** | Feedback — In-App Feedback & Admin Review |
| **Severity** | MINOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | COMPLETED |
| **Estimated effort** | XS–S (≤1w) |
| **Platforms** | Web (React + Tailwind) |
| **Owner (proposed)** | Web |
| **Depends on** | [FB0](./FB0-feedback-foundation-api.md) |
| **Unblocks** | End-to-end feedback loop on web |
| **Permission** | Any authenticated user |

---

## 1. Problem Statement

Web users have nowhere to tell us what's wrong or what they want. There is no visible, low-friction way to send feedback from inside the app, so signal is lost. This story adds a **prominent, always-available "Share Feedback" button in the top nav** that opens a short form and posts to the FB0 submit endpoint, closing the loop for the largest client surface.

## 2. Goals

- A **Share Feedback** control that stands out in the top nav on every authenticated page.
- A lightweight modal form: required message, optional category, submit — nothing more.
- Auto-attach source (`web`), app version, and current route as context (no user effort).
- Clear success/error feedback with graceful, non-blocking failure.
- Fully accessible (keyboard, focus trap, screen-reader) and localized.

## 3. Non-Goals

- No backend/schema work (FB0).
- No admin/list/detail views (FB2).
- No file/screenshot attachments (future — §18).
- No feedback affordance inside focus modes (quiz/reading top bars) — MVP keeps it on the standard shell only.

## 4. Personas & User Stories

- **As any signed-in web user**, I click **Share Feedback** in the top nav, type a note, and send it in seconds.
- **As a user who hit a bug on a specific page**, I want the app to remember which page I was on so I don't have to describe it.
- **As a keyboard/screen-reader user**, I can open, complete, and submit the form without a mouse.

## 5. Functional Requirements

- **FR-1.** A **Share Feedback** button MUST render in the top-bar widget cluster (`top-bar.tsx`), visible on all standard authenticated routes, gated by the `ff_feedback` flag.
- **FR-2.** The button MUST visually **stand out** — accent fill (indigo brand) or a distinct outlined pill with icon + label, not blending into the icon-only controls.
- **FR-3.** Activating it MUST open a modal dialog with: a required multiline **message** field, an optional **category** select (`bug`/`idea`/`question`/`praise`/`other`), a **Send** and **Cancel** action.
- **FR-4.** The client MUST disable Send while the message is empty/whitespace and MUST show a character counter approaching the 5,000 cap.
- **FR-5.** On Send, the client MUST `POST /api/v1/feedback` with `source:"web"`, `app_version`, and `context.route` = current pathname (+ `locale`, `viewport`), via `authorizedFetch`.
- **FR-6.** On success (`201`) the modal MUST close and a success toast ("Thanks for your feedback") MUST show.
- **FR-7.** On error the form MUST stay open, preserve input, and show an inline error; on `429` it MUST show a friendly "slow down" message.
- **FR-8.** The dialog MUST trap focus, close on Esc / backdrop, and restore focus to the trigger.
- **FR-9.** On narrow viewports the control MAY collapse to an icon-only button (with `aria-label`) but MUST remain visible.

## 6. Non-Functional Requirements

- **Performance** — form code lazy-loaded (like `AiTutorMenu`/`ReadingPreferencesPanel`); no measurable top-bar render regression; submit non-blocking.
- **Security** — send only via `authorizedFetch`; never inject server response as HTML; no identity in the body (server derives it).
- **Privacy** — copy notes that feedback is read by staff; don't auto-scrape page content beyond the route path.
- **Accessibility** — WCAG 2.1 AA: labeled trigger, `role="dialog"` + `aria-modal`, focus order, visible focus ring, ≥44px target, contrast on the accent button; error announced via `aria-live`.
- **Reliability** — a failed submit surfaces an error and allows retry; never crashes the shell.
- **Internationalization** — all strings via web i18n (`src/i18n`); RTL-safe (uses logical properties like existing top-bar).
- **Maintainability** — new `feedback-widget.tsx` + `feedback-dialog.tsx` under `components/layout` or `components/feedback`, mirroring `help-widget.tsx`.

## 7. Acceptance Criteria

- **AC-1.** *Given* any authenticated page with `ff_feedback` on, *Then* a visually prominent **Share Feedback** control appears in the top nav.
- **AC-2.** *Given* the control is activated, *Then* a focus-trapped modal opens with message + category + Send/Cancel.
- **AC-3.** *Given* an empty message, *Then* Send is disabled.
- **AC-4.** *Given* a valid message + category, *When* Send is clicked, *Then* a `POST /api/v1/feedback` fires with `source:"web"` and `context.route` set, the modal closes, and a success toast shows.
- **AC-5.** *Given* the endpoint returns `429`, *Then* the form stays open with a friendly rate-limit message and input preserved.
- **AC-6.** *Given* a keyboard-only user, *Then* they can open, fill, submit, and the dialog restores focus to the trigger on close.
- **AC-7.** *Given* `ff_feedback` is off, *Then* the control does not render.

## 8. Data Model

- None (client only). Reads `ff_feedback` from the platform-features context; posts to FB0.

## 9. API Surface

- Consumes `POST /api/v1/feedback` (FB0 §9). No new endpoints.

## 10. UI / UX

- **Placement:** top-bar right cluster in `top-bar.tsx`, near `HelpWidgetMenu` (feedback and help are cousins). Order suggestion: Reading prefs · AI tutor · Help · **Share Feedback** · Notifications · View-as · User menu.
- **Style:** accent pill — e.g. indigo-600 fill, white text, `MessageSquarePlus`/`Megaphone` lucide icon + "Share Feedback" label on ≥md; icon-only with `aria-label` on small screens. Reuse the button idioms already in `top-bar.tsx` (rounded-xl, focus-visible ring).
- **Flow:** (1) click trigger → (2) modal opens, message focused → (3) type, optionally pick category → (4) Send → (5) toast + close.
- **States:** default, submitting (spinner on Send, disabled), success (toast + close), error (inline, retryable), rate-limited (friendly copy), empty (Send disabled). Offline: show "You're offline — try again" and keep input.
- **Copy/i18n keys:** `feedback.button`, `feedback.dialog.title`, `feedback.message.label`, `feedback.message.placeholder`, `feedback.category.label` + option labels, `feedback.send`, `feedback.cancel`, `feedback.success`, `feedback.error`, `feedback.rateLimited`.
- **Accessibility annotations:** trigger `aria-haspopup="dialog"` / `aria-expanded`; dialog `role="dialog" aria-modal="true" aria-labelledby`; Esc + backdrop close; focus returns to trigger.

## 11. AI / ML Considerations

- None.

## 12. Integration Points

- `clients/web/src/components/layout/top-bar.tsx` (mount the trigger).
- New `clients/web/src/components/feedback/feedback-widget.tsx` + `feedback-dialog.tsx` (or under `components/layout`, matching `help-widget.tsx`).
- `clients/web/src/lib/api.ts` (`authorizedFetch`), platform-features context (`ff_feedback`), existing toast primitive, web i18n.

## 13. Dependencies & Sequencing

- After FB0 (submit endpoint + `ff_feedback` flag). Recommended to land first among FB1/FB2/FB3 to validate the end-to-end path.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Button crowds an already busy top bar | M | M | Collapse to icon-only on small screens; single accent control; UX review |
| Users expect a reply | M | L | Copy sets expectations ("We read every note; we may not reply individually") |
| Accidental double-submit | L | L | Disable Send while in-flight; optional idempotency_key |
| Distracting/too prominent | L | M | Accent pill, not animated/persistent banner; dogfood for tuning |

## 15. Rollout Plan

- Gated by `ff_feedback` (FB0). Internal org → all.
- Rollback: flag off hides the control.
- Watch `feedback_submitted_total{source="web"}` and client error logs post-launch.

## 16. Test Plan

- **Unit** — trigger renders under flag; Send disabled on empty; payload shape (source/route/version); toast on success; error/429 handling.
- **Integration** — mocked `POST /api/v1/feedback` success + failure paths.
- **E2E (Playwright)** — open from top nav → type → send → toast + closed; keyboard-only path; flag-off hides control.
- **Accessibility** — axe on the dialog; focus trap + restore; screen-reader label check; contrast on accent button.
- **Manual** — RTL locale, narrow viewport collapse, offline behavior.

## 17. Documentation & Training

- Help-center: "How to share feedback" (where the button is, what happens next).
- Release note announcing the button.

## 18. Open Questions

1. Attachments/screenshots — worth it for bug reports? (Future; would need object storage + scan.)
2. Include feedback in focus modes (quiz/reading top bars)? MVP excludes them.
3. Exact icon + label wording ("Share Feedback" vs "Feedback" vs "Send Feedback") — settle in UX review.

## 19. References

- Existing files this work touches: `clients/web/src/components/layout/top-bar.tsx`, `clients/web/src/components/layout/help-widget.tsx` (pattern), `clients/web/src/lib/api.ts`, platform-features context.
- Related plans: [FB0](./FB0-feedback-foundation-api.md), [FB2](../../plan/feedback/FB2-web-feedback-admin.md), [FB3](../../plan/feedback/FB3-mobile-share-feedback.md).
