# ET-2 — Markdown→HTML compilation & routing system emails through templates

> Implementation plan. Extends [18.5 Email Template Editor](../18-admin-experience/18.5-email-template-editor.md). Epic index: [emails/README.md](../../plan/emails/README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | ET-2 (extends 18.5) |
| **Section** | Admin Experience — Email Templates |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | **DONE** — Markdown compile + org→system→code delivery wired |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Platform / Backend |
| **Depends on** | ET-1 (markdown column + system scope + slots) |
| **Unblocks** | ET-3 (UI edits map through this pipeline); edited system emails take effect on live sends |

---

## 1. Problem Statement

Two gaps block "edit a template, and the next email uses it." First, templates are
authored as raw HTML; the request is a **Markdown editor that turns into HTML**, so
we need a server-side Markdown→email-safe-HTML compiler. Second, the platform's
**system emails do not flow through the template system at all**:
`SendMagicLinkEmail`, `SendPasswordResetEmail`, and `SendCoppaConsent*` build bodies
inline, and `background/jobqueue_email.go` calls `mail.RenderTemplate` directly —
both skip `RenderForDelivery`, so any saved override is ignored. This story adds the
compilation pipeline and rewires delivery so overrides (system, then per-org) win,
with the built-in Go default as the final fallback.

## 2. Goals

- Compile slot/override **Markdown → sanitized, email-safe HTML** server-side using
  `goldmark` + `bluemonday` (both already vendored), preserving `{{token}}` merge
  fields.
- Make Markdown the **canonical source**: store `source_markdown`, compile to
  `html_body` at save, and auto-derive a plain-text fallback.
- Extend `RenderForDelivery` to resolve **per-org → system → code default** and to
  serve compiled HTML with merge fields applied.
- Route the three hard-coded system senders and the job-queue delivery path through
  the template layer so edited templates are actually sent.
- Fail safe: any compile/render error falls back to the built-in Go template, never
  a blank or broken email.

## 3. Non-Goals

- The editor UI and preview panel (→ ET-3), though this story exposes the
  server-side preview/compile the UI calls.
- New slots or schema (→ ET-1).
- Changing which events trigger emails, recipients, or notification preferences.
- MJML or a full responsive-email framework (compiled HTML uses the existing
  inline-style wrapper `renderWrappedHTML`).

## 4. Personas & User Stories

- **As a platform admin**, I want my Markdown edits to the magic-link email to be
  what users actually receive, so copy changes don't need a deploy.
- **As a security reviewer**, I want authored HTML sanitized server-side so a
  template can't inject script into an email.
- **As an on-call engineer**, I want a broken template to fall back to the default
  automatically and log the fallback, so a bad edit can't stop password resets.
- **As a backend engineer**, I want one delivery entry point that resolves scope
  order consistently for both queued and direct sends.

## 5. Functional Requirements

- **FR-1.** The system MUST provide `Compile(markdown string) (html string, err
  error)` in `service/emailtemplates` that renders Markdown via `goldmark`
  (GFM + hard line breaks, autolinks) then sanitizes via the existing
  `emailSanitizePolicy` (`SanitizeHTML`).
- **FR-2.** Compilation MUST preserve `{{token}}` sequences verbatim (merge happens
  after compile at send/preview time, as 18.5 already does on `html_body`).
- **FR-3.** `Save` (org) and `SaveSystem` MUST accept `source_markdown`, compile it
  to `html_body`, sanitize, derive text (provided `text_body`, else
  `StripHTMLTags(compiled)`), and validate unknown merge tokens against the slot
  catalog (warning, not error) — reusing `ValidateUnknown`.
- **FR-4.** `Preview` MUST accept Markdown, compile+sanitize+merge with sample data,
  and return `{ html, text }` (used by ET-3's live preview and test send).
- **FR-5.** `RenderForDelivery` MUST resolve in order: active **org** override →
  active **system** override → `mail.RenderTemplate` (built-in). The first non-nil
  override supplies compiled `html_body` (+ merge) and `text_body`.
- **FR-6.** A new `RenderSystemForDelivery(ctx, pool, slotName, vars, branding)`
  MUST exist for sends with no org context (auth/magic-link), resolving **system →
  code default** only.
- **FR-7.** `SendMagicLinkEmail`, `SendPasswordResetEmail`, and
  `SendCoppaConsentNotice` / `SendCoppaConsentConfirmation` MUST render via the
  template layer (system scope, or org scope when an org is known) instead of inline
  string building, passing the same `vars` the slot catalog documents.
- **FR-8.** `background/jobqueue_email.go` MUST call `RenderForDelivery` (org-aware)
  instead of `mail.RenderTemplate`, matching `email_worker.go`.
- **FR-9.** On any compile/merge/render error, the system MUST fall back to the
  built-in Go default for that slot and emit a warning log + `fallback` metric; the
  send MUST still proceed.
- **FR-10.** `subject` MUST come from the slot (`slot.Description` today) or an
  explicit per-slot subject field; system senders MUST preserve their current
  subjects as defaults (e.g. "Your StudyDrift sign-in link").

## 6. Non-Functional Requirements

- **Performance** — Compile + sanitize + merge for a transactional email MUST be
  ≤ 50 ms p95; preview endpoint MUST be ≤ 200 ms. Optionally cache compiled HTML by
  `(scope, slot, version)` hash (compiled HTML already persisted at save, so
  delivery does not recompile).
- **Security** — All authored HTML passes `bluemonday` before storage AND the
  preview wrapper is sandboxed by ET-3; no client-side merge/eval; goldmark
  configured with `html.WithUnsafe()` **disabled** (raw HTML in Markdown is dropped,
  then bluemonday re-sanitizes). Merge resolves server-side only.
- **Privacy & Compliance** — COPPA consent send keeps its defensive URL check
  ("looks like a consent URL") and required disclosures; grade/FERPA emails keep
  footer semantics. Test data never leaks across scope.
- **Accessibility** — Compiled HTML uses semantic tags from Markdown (headings,
  lists, descriptive link text); the wrapper sets `lang` and readable defaults.
- **Scalability** — Delivery reads one compiled row; no compile in the hot path.
- **Reliability** — Deterministic fallback chain; idempotent renders; a template
  error is contained to one slot.
- **Observability** — Counters `email_template_compile_total`,
  `email_template_render_fallback_total{scope,slot}`; reuse `RecordSave` /
  `RecordTestSend`; log slot + scope + error on fallback.
- **Maintainability** — Single `Compile` + single `RenderForDelivery` used by all
  senders; `mail/*` inline HTML builders deleted or reduced to the fallback path.
- **Internationalization** — Merge values already localized upstream; compile is
  locale-agnostic.
- **Backward compatibility** — Slots with only `default_html` (no override) still
  render via `mail.RenderTemplate`; org rows saved before ET-1 (null
  `source_markdown`) continue to serve their stored `html_body` until re-saved.

## 7. Acceptance Criteria

- **AC-1.** *Given* `Compile("**Hi** [x]({{link}})")`, *When* called, *Then* it
  returns sanitized HTML containing `<strong>Hi</strong>` and an `<a href>` with the
  literal `{{link}}` preserved, and no `<script>` survives an injected
  `<script>alert(1)</script>`.
- **AC-2.** *Given* a saved **system** override for `magic_link`, *When*
  `SendMagicLinkEmail` runs, *Then* the delivered HTML matches the compiled override
  with `{{link}}`/`{{expires_at}}` resolved (verified via a capturing test
  provider).
- **AC-3.** *Given* an org override AND a system override for `password_reset`,
  *When* a reset email is sent for a user in that org, *Then* the **org** override is
  used (per-org wins).
- **AC-4.** *Given* a system override whose Markdown compiles to invalid/empty
  output, *When* a send occurs, *Then* the built-in Go default is used, the send
  succeeds, and `email_template_render_fallback_total` increments.
- **AC-5.** *Given* `jobqueue_email.go` delivery for a slot with an org override,
  *When* the job runs, *Then* the override is used (parity with `email_worker.go`).
- **AC-6.** *Given* a `Preview` call with Markdown and no `text_body`, *When*
  returned, *Then* `text` is a tag-free rendering derived from the compiled HTML.

## 8. Data Model

No schema changes (ET-1 owns them). Behavioral change: `html_body` in
`org_email_templates` / `system_email_templates` is now the **compiled artifact** of
`source_markdown`; `source_markdown` is authoritative for re-editing. `Save` writes
both in one transaction.

## 9. API Surface

No new HTTP routes here (ET-3 wires them); service signatures change:

```go
// service/emailtemplates
func Compile(markdown string) (string, error)               // goldmark → bluemonday

func (s *Service) SaveSystem(ctx, slotID, markdown string,
    text, replyTo, senderName *string, actor uuid.UUID) (*SaveResult, error)
func (s *Service) Save(ctx, orgID, slotID, markdown string, …) (*SaveResult, error) // now markdown-first

func (s *Service) Preview(markdown string, text *string,
    data map[string]string) PreviewResult                   // markdown-first

func RenderForDelivery(ctx, pool, orgID, slot, vars, branding) (mail.RenderedEmail, error)       // org→system→code
func RenderSystemForDelivery(ctx, pool, slot, vars, branding) (mail.RenderedEmail, error)          // system→code
```

Signature note: 18.5's `Save`/`Preview` currently take `htmlBody`. This story
migrates them to Markdown; the ET-3 admin-console callers are updated in lockstep
(the existing HTML editor is replaced by the markdown editor in ET-3, so no
dual-format period is required). If a staged rollout is preferred, keep `SaveHTML`
as a thin deprecated shim — see §18.

## 10. UI / UX

None here. ET-3 calls `Preview`/`Save`/`SaveSystem` and renders the returned HTML in
a sandboxed iframe.

## 11. AI / ML Considerations

Not applicable.

## 12. Integration Points

- `server/internal/service/emailtemplates/` — new `compile.go` (`Compile`);
  `service.go` (`Save`, `SaveSystem`, `Preview`, `RenderForDelivery`,
  `RenderSystemForDelivery`); `metrics.go` (fallback/compile counters).
- `server/internal/repos/emailtemplates/repo.go` — `Save`/`SaveSystem` persist
  `source_markdown` + compiled `html_body` in a tx.
- `server/internal/mail/magic_link.go`, `password_reset.go`, `coppa_consent.go` —
  render via template layer; keep inline builders as the fallback used by
  `mail.RenderTemplate`.
- `server/internal/background/jobqueue_email.go` — swap `mail.RenderTemplate` →
  `RenderForDelivery`.
- `server/internal/background/email_worker.go` — already uses `RenderForDelivery`;
  no change beyond scope-order update.
- `server/internal/service/authservice/{credentials,magic_link}.go` — call sites
  unchanged (they call the `mail.Send*` wrappers).
- Vendored: `github.com/yuin/goldmark` v1.7.17, `github.com/microcosm-cc/bluemonday`
  v1.0.27.

## 13. Dependencies & Sequencing

- Must ship after: ET-1.
- Must ship before: ET-3 (UI depends on markdown Save/Preview).
- Shared infra: existing mail providers (SMTP/SES); no new deps.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Markdown mangles `{{token}}` (e.g. inside code spans) | M | M | Document tokens live in prose/links, not code fences; golden test compiles each default and asserts tokens survive; merge runs post-compile |
| Broken override blocks a critical send (password reset) | L | H | FR-9 fallback chain + fallback metric + alert; unit test forces compile error path |
| Sanitizer strips a needed tag (e.g. `<table>` in COPPA notice) | M | M | Extend `emailSanitizePolicy` to allow email-safe `table/tr/td/th` with `style`; snapshot COPPA compile |
| Signature migration breaks 18.5 admin-console callers | M | M | Update callers in same PR; optional `SaveHTML` shim (§18); CI type-checks |
| Double-sanitize alters output vs. 18.5 stored HTML | L | L | goldmark unsafe-disabled → bluemonday once; golden snapshot of the 7 existing slots |

## 15. Rollout Plan

- Feature flag: `email_template_editor_enabled` still gates editing; delivery
  rewiring ships behind it too — when off, senders resolve **code default only**
  (no override lookup), preserving today's behavior exactly.
- Sequencing: land `Compile` + tests → migrate `Save`/`Preview` → rewire
  `RenderForDelivery` scope order → rewire system senders + jobqueue → enable flag on
  staging.
- Dogfood: save a system override for `magic_link` on staging, trigger a real
  sign-in, confirm the edited copy arrives; then a bad edit → confirm fallback.
- GA criteria: golden snapshots stable; fallback metric ~0 in steady state; test
  sends verified across SMTP and SES.
- Rollback: disable flag → senders revert to code defaults; stored overrides remain
  but are not consulted.

## 16. Test Plan

- **Unit** — `Compile` (formatting, GFM, autolink, token preservation, script/attr
  stripping); text-fallback derivation; scope-order resolution in
  `RenderForDelivery`/`RenderSystemForDelivery`; forced compile-error → fallback.
- **Integration** — Save system override → `SendMagicLinkEmail` via a capturing
  provider asserts delivered HTML == compiled override with merges applied;
  org-vs-system precedence; jobqueue path parity with worker path.
- **End-to-end** — deferred to ET-3 (UI), but a Go integration test drives
  save→send for each system slot.
- **Security** — inject `<script>`, `onerror=`, `javascript:` href, raw `<iframe>`
  in Markdown; assert all stripped from stored + delivered HTML.
- **Accessibility** — assert compiled output uses `<h*>`, `<ul>/<li>`, and link
  text (no bare URLs) for defaults.
- **Performance / load** — micro-benchmark `Compile`; assert delivery path does not
  recompile (reads stored `html_body`).
- **Manual exploratory** — send each system email to a real inbox; check Gmail +
  Apple Mail rendering.

## 17. Documentation & Training

- Update the 18.5 help-center article: templates are authored in **Markdown**;
  supported syntax; that `{{tokens}}` must stay outside code spans.
- Runbook: the fallback chain and how to read
  `email_template_render_fallback_total`; how to force-reset a bad override.
- API reference: `Preview`/`Save`/`SaveSystem` now markdown-first.

## 18. Open Questions

1. Migrate 18.5 `Save`/`Preview` to markdown-only, or keep a deprecated `SaveHTML`
   shim for one release? (Recommend clean migration since ET-3 replaces the only
   caller UI in the same epic.)
2. Cache compiled HTML in memory keyed by version hash, or rely on the persisted
   `html_body`? (Recommend persisted only; recompute on save.)
3. Should `subject` become an editable per-slot field now, or stay slot-derived?
   (Recommend a follow-up; keep slot-derived subjects this story.)
4. Allow email-safe `<table>` in the sanitizer policy for layout-heavy notices?
   (Recommend yes, scoped to `table/tr/td/th` + `style`.)

## 19. References

- `server/internal/service/emailtemplates/{service,merge,sanitize,metrics}.go`.
- `server/internal/background/{email_worker,jobqueue_email}.go`.
- `server/internal/mail/{magic_link,password_reset,coppa_consent,templates,send}.go`.
- goldmark: https://github.com/yuin/goldmark · bluemonday:
  https://github.com/microcosm-cc/bluemonday
- Related: [ET-1](ET-1-system-scope-slots-and-defaults.md),
  [ET-3](ET-3-system-settings-email-templates-page.md).
