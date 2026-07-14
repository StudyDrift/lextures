# ET-3 — System Email Templates page in system settings (markdown UI)

> Implementation plan. Extends [18.5 Email Template Editor](../18-admin-experience/18.5-email-template-editor.md). Epic index: [emails/README.md](../../plan/emails/README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | ET-3 (extends 18.5) |
| **Section** | Admin Experience — Email Templates |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | **DONE** — system settings Email Templates page + platform API |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Platform / Frontend |
| **Depends on** | ET-1 (slots + system scope), ET-2 (markdown compile + Preview/Save APIs) |
| **Unblocks** | Self-service editing of every system/transactional email from the system settings pages |

---

## 1. Problem Statement

Admins have no place to view or edit the platform's system emails. The 18.5 editor
lives in the per-org **admin console** and authors raw HTML; the request is a page in
the **system settings pages** where an admin selects a template, sees its current
content, edits it in the **same Markdown editor used elsewhere in the app**, and
saves — after which live sends use it. This story delivers that page against the
ET-1/ET-2 backend: a template picker, a Markdown editor with merge-field insertion,
a live HTML preview, test-send, version history, and reset-to-default.

## 2. Goals

- Add an **Email Templates** page within the system settings surface
  (`clients/web/src/pages/lms/settings.tsx`, alongside `PlatformSettingsPanel`),
  gated by `email_template_editor_enabled` and super-admin.
- Reuse the app's existing Markdown editing stack (block-editor / markdown toolbar,
  `marked`/`markdown-it` for instant client preview) rather than the 18.5 TipTap
  HTML editor.
- Let admins **select a slot**, view the effective source (system override →
  slot default), edit Markdown, and see a **live compiled HTML preview** in a
  sandboxed frame.
- Support insert-merge-field, unknown-token warnings, **save**, **send test to
  self**, **version history + restore**, and **reset to default** — mapping to the
  ET-2 system-scope service.
- Keep it accessible (WCAG 2.1 AA) and responsive.

## 3. Non-Goals

- Backend compile/render/scope logic (→ ET-2) and schema (→ ET-1).
- Rebuilding the per-org admin-console editor (18.5) — though ET-2's markdown
  migration means that page's editor is swapped to markdown too (shared component).
- WYSIWYG HTML editing, image upload into templates (URL references only), or
  per-locale editing.

## 4. Personas & User Stories

- **As a platform/super-admin**, I want to open "Email Templates" in system
  settings, pick "Magic sign-in link", edit the Markdown, and save so the next
  sign-in email uses my copy.
- **As an admin**, I want a live preview with sample data so I can see the rendered
  email before saving.
- **As an admin**, I want to send myself a test so I can verify it in a real inbox.
- **As an admin who broke a template**, I want "Reset to default" and version
  history so I can recover safely.
- **As a screen-reader user**, I want the editor, merge-field picker, and preview
  updates to be announced and keyboard-operable.

## 5. Functional Requirements

- **FR-1.** The system MUST show an **Email Templates** entry in the system settings
  navigation, visible only when `emailTemplateEditorEnabled` is true and the user is
  super-admin; selecting it renders the templates page.
- **FR-2.** The page MUST list all slots (from `GET …/email-templates`) with
  description and a badge for "Customized" vs "Default".
- **FR-3.** Selecting a slot MUST load its effective Markdown source (system
  override if present, else `default_markdown`) plus `reply_to`/`sender_name` and
  the merge-field catalog.
- **FR-4.** The editor MUST be the app's shared **Markdown** editor with a toolbar
  (bold, italic, link, lists, headings) and a **merge-field picker** that inserts
  `{{token}}` at the cursor; each field shows its human description from the catalog.
- **FR-5.** The page MUST render a **live preview** of the compiled HTML with sample
  data in a **sandboxed iframe** (`sandbox` attribute, no scripts), updating as the
  admin types (debounced), using client `marked`/`markdown-it` for instant feedback
  and the server `POST …/preview` for the canonical render on demand/save.
- **FR-6.** Saving (`PUT …/email-templates/{slotId}`) MUST send `source_markdown`
  (+ optional `text_body`, `reply_to`, `sender_name`); the response's
  `unknownFields` MUST surface as a non-blocking warning banner.
- **FR-7.** The page MUST offer **Send test to me** (`POST …/{slotId}/test`) with a
  confirmation showing the recipient (the admin's own email) and a success/toast.
- **FR-8.** The page MUST show **version history** (`GET …/{slotId}/history`) in a
  drawer with timestamps and **Restore** (`POST …/{slotId}/restore`).
- **FR-9.** The page MUST offer **Reset to default** (`POST …/{slotId}/reset`) with a
  confirm dialog; after reset, subsequent sends use the slot default and history is
  preserved.
- **FR-10.** All states — loading, empty (no slots), save error, preview error,
  unsaved-changes navigation guard — MUST be handled explicitly.

## 6. Non-Functional Requirements

- **Performance** — Slot list + selected slot load ≤ 1s p95; client preview updates
  ≤ 100ms after debounce; server preview ≤ 200ms (ET-2).
- **Security** — Preview iframe is `sandbox`ed (no `allow-scripts`); rendered HTML is
  the server-sanitized output; no `dangerouslySetInnerHTML` outside the sandboxed
  frame; super-admin gate enforced server-side (ET-3 routes), not just hidden nav.
- **Privacy & Compliance** — Test email targets only the requesting admin's address
  (server-enforced, 18.5 behavior). COPPA slot editing surfaces a note that required
  disclosures must remain.
- **Accessibility** — WCAG 2.1 AA: editor + toolbar + merge picker fully keyboard
  operable; merge picker reachable via toolbar button and `{{` trigger; preview
  region `aria-live="polite"` announces "Preview updated"; focus order editor →
  preview → actions; visible focus rings; dialogs trap focus. axe-core clean.
- **Scalability** — Static slot count; no pagination needed.
- **Reliability** — Optimistic save disabled; save is confirmed by server response;
  navigation guard prevents silent loss of edits.
- **Observability** — Reuse backend counters; optional web analytics event on save /
  test-send / reset.
- **Maintainability** — Share one `MarkdownEmailEditor` component between the system
  page and the 18.5 admin-console page (both markdown-first post-ET-2).
- **Internationalization** — All UI strings via `react-i18next` (`common` + a new
  `emailTemplates` namespace); no hard-coded copy. Template *content* stays
  admin-authored.
- **Backward compatibility** — When the flag is off, the nav entry and routes are
  hidden/blocked; the existing 18.5 admin-console page continues to function.

## 7. Acceptance Criteria

- **AC-1.** *Given* a super-admin with the flag on, *When* they open system settings,
  *Then* an "Email Templates" entry is present; *Given* the flag off, *Then* it is
  absent and the route returns 404/forbidden.
- **AC-2.** *Given* the templates page, *When* the admin selects "magic_link" and
  edits the Markdown, *Then* the live preview updates within ~100ms and shows the
  compiled HTML with sample `{{link}}`/`{{expires_at}}` resolved.
- **AC-3.** *Given* an edited template, *When* the admin clicks Save, *Then* a new
  system version is stored and a subsequent real magic-link email uses the edited
  copy (end-to-end with ET-2).
- **AC-4.** *Given* an unknown token `{{foo.bar}}`, *When* the admin saves, *Then* a
  yellow warning lists the unknown field and the save still succeeds.
- **AC-5.** *Given* Send-test, *When* confirmed, *Then* the email arrives at the
  admin's own address within 60s with merges resolved.
- **AC-6.** *Given* two saved versions, *When* the admin opens history and clicks
  Restore on v1, *Then* v1 becomes active; *Given* Reset-to-default, *Then* sends use
  the slot default and history is retained.
- **AC-7.** *Given* keyboard-only navigation, *When* the admin tabs through the page,
  *Then* editor, merge picker, preview, and all actions are reachable and operable;
  axe-core reports no violations.

## 8. Data Model

None. Consumes ET-1 columns/tables via ET-2/ET-3 APIs.

## 9. API Surface

New **system-scope** HTTP routes mirroring the 18.5 admin-console routes, mounted
under the platform settings namespace and gated by super-admin +
`email_template_editor_enabled` (server-side):

```
GET    /api/v1/settings/platform/email-templates                 -> Slot[] (+ customized flag)
GET    /api/v1/settings/platform/email-templates/{slotId}        -> { slot, activeVersion? }
PUT    /api/v1/settings/platform/email-templates/{slotId}        { sourceMarkdown, textBody?, replyTo?, senderName? } -> { version, unknownFields? }
GET    /api/v1/settings/platform/email-templates/{slotId}/history -> version[]
POST   /api/v1/settings/platform/email-templates/{slotId}/restore { versionId } -> version
POST   /api/v1/settings/platform/email-templates/{slotId}/reset   -> 204
POST   /api/v1/settings/platform/email-templates/{slotId}/test    -> 202 (to requesting admin)
POST   /api/v1/settings/platform/email-templates/{slotId}/preview { sourceMarkdown, textBody?, sampleData? } -> { html, text }
```

Handlers live in a new `server/internal/httpserver/settings_email_templates.go`,
reusing `emailtemplatesvc.Service` system-scope methods (ET-2). Request/response
JSON mirrors `admin_email_templates.go` with `sourceMarkdown` replacing `htmlBody`.
OpenAPI updated. Rate-limit: test-send throttled per admin (reuse existing
limiter).

## 10. UI / UX

- **New page/component:** `clients/web/src/components/settings/system-email-templates-panel.tsx`
  registered in `clients/web/src/pages/lms/settings.tsx` next to
  `PlatformSettingsPanel`; nav gated in the system-settings nav (mirror
  `side-nav-admin-links.tsx` gating on `emailTemplateEditorEnabled`).
- **Shared editor:** `clients/web/src/components/settings/markdown-email-editor.tsx`
  wrapping the app's markdown editing stack (block-editor toolbar /
  `markdown-format-toolbar`), reused by the 18.5 admin page after ET-2.
- **API client:** extend `clients/web/src/lib/email-templates-api.ts` with
  system-scope functions (or a sibling `system-email-templates-api.ts`).
- **Layout / flow (numbered):**
  1. Left: slot list with Default/Customized badges.
  2. Center: Markdown editor + toolbar + merge-field picker (`{{` or button).
  3. Right: sandboxed iframe live preview (sample data), `aria-live` status.
  4. Header actions: Save · Send test · History · Reset to default.
  5. Footer: unknown-token warning banner; unsaved-changes indicator.
- **States:** loading skeletons; empty (feature on but no slots — shouldn't happen);
  save error toast; preview error inline; navigation guard on dirty edits.
- **Mobile / responsive:** editor full-width; preview behind a "Preview" tab toggle;
  actions collapse into a menu.
- **Accessibility annotations:** documented in FR/NFR — labelled controls, focus
  management in dialogs, `aria-live` preview, keyboard merge insertion.
- **Copy & i18n:** new `emailTemplates` i18n namespace; slot descriptions come from
  the API (server-owned).

## 11. AI / ML Considerations

Not applicable. (Future: "improve this copy" assist — out of scope.)

## 12. Integration Points

- `clients/web/src/pages/lms/settings.tsx` — register panel + nav entry.
- `clients/web/src/context/platform-features-context.tsx` /
  `platform-feature-definitions.ts` — `emailTemplateEditorEnabled` already surfaced;
  reuse for nav gating.
- `clients/web/src/components/editor/block-editor/markdown-format-toolbar.tsx` and
  the markdown stack — shared editor base.
- `clients/web/src/lib/email-templates-api.ts` — extend for system scope.
- `clients/web/src/app.tsx` / `lazy-pages.ts` — lazy-load the panel if routed
  standalone.
- Backend: `server/internal/httpserver/settings_email_templates.go` (new) +
  `emailtemplatesvc` system-scope methods (ET-2).

## 13. Dependencies & Sequencing

- Must ship after: ET-1, ET-2.
- Must ship before: GA of admin-editable system emails.
- Shared infra: existing settings page shell, i18n, feature-flag context, mail
  delivery.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Client (`marked`) preview diverges from server (`goldmark`) output | M | M | Server `POST /preview` is canonical; run it on blur/save; label client preview "approximate" |
| XSS via preview render | L | H | Sandboxed iframe (no scripts) + server-sanitized HTML only |
| Admin edits COPPA email and drops required disclosures | M | H | Inline compliance note + non-blocking reminder; keep code-default fallback |
| Nav shown but routes unguarded | L | H | Enforce super-admin + flag in handlers, not just nav visibility |
| Editor component fork drifts from 18.5 | M | L | Single shared `MarkdownEmailEditor`; 18.5 migrated to it in ET-2/ET-3 |

## 15. Rollout Plan

- Feature flag: `email_template_editor_enabled` gates nav + routes; default off.
- Sequencing: land API handlers → panel + shared editor → nav gating → enable for
  super-admin pilot on staging.
- Dogfood: edit `magic_link` on staging, preview, test-send, then trigger a real
  sign-in; verify; then reset-to-default.
- GA criteria: axe-core clean, e2e green, client/server preview parity acceptable,
  test-send verified.
- Rollback: disable flag → page + routes hidden/blocked; stored overrides retained
  but (per ET-2) not consulted while flag off.

## 16. Test Plan

- **Unit** — panel rendering per state; merge-field insertion at cursor;
  unknown-token warning; dirty-nav guard; api-client request shaping.
- **Integration** — mock API: select → edit → preview → save → history → restore →
  reset happy paths and error paths.
- **End-to-end (Playwright)** — super-admin opens Email Templates → edits magic_link
  → live preview updates → save → send test → verify version in history → reset;
  flag-off asserts nav/route absence.
- **Security** — attempt non-super-admin access to routes (expect 403); inject
  script in Markdown and confirm sandboxed preview does not execute.
- **Accessibility** — axe-core on the page + dialogs; keyboard-only walkthrough;
  screen-reader announcement of preview updates.
- **Performance / load** — debounced preview does not thrash; server preview p95
  within budget.
- **Manual exploratory** — cross-client inbox check of a test send; mobile layout.

## 17. Documentation & Training

- Help center: "Editing system email templates" — where the page lives (system
  settings), Markdown basics, merge fields, preview, test, history, reset.
- Admin guide: which emails are system-scope vs per-org (link to 18.5), and the
  resolution order.
- Update the 18.5 article to note the editor is now Markdown-based and shared.

## 18. Open Questions

1. Mount as a settings **panel/tab** within the existing platform settings page, or a
   standalone route `/settings/platform/email-templates`? (Recommend a panel/tab for
   discoverability alongside SMTP/Provider settings.)
2. Should the per-org 18.5 admin-console page switch to this shared markdown editor
   in the same release, or lag by one? (Recommend same release to avoid two editors.)
3. Expose `reply_to`/`sender_name` at system scope in this page, or defer to a
   later branding pass? (Recommend expose; fields already exist.)
4. Separate flag for the *system* page (e.g. `system_email_templates_enabled`) vs.
   reusing `email_template_editor_enabled`? (Recommend reuse to avoid flag sprawl.)

## 19. References

- `clients/web/src/components/admin/EmailTemplateEditor.tsx` (18.5 editor to
  supersede), `clients/web/src/lib/email-templates-api.ts`.
- `clients/web/src/pages/lms/settings.tsx`,
  `clients/web/src/components/settings/platform-settings-panel.tsx`,
  `clients/web/src/components/layout/side-nav-admin-links.tsx` (nav gating pattern).
- `clients/web/src/components/editor/block-editor/markdown-format-toolbar.tsx`
  (markdown stack); `marked` / `markdown-it` (already in `clients/web/package.json`).
- `server/internal/httpserver/admin_email_templates.go` (route/handler pattern to
  mirror for system scope).
- Related: [ET-1](ET-1-system-scope-slots-and-defaults.md),
  [ET-2](ET-2-markdown-compilation-and-delivery-wiring.md).
