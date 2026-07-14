# Email Templates — Markdown-authored system email editor

Epic that lets platform admins author the **system / transactional emails** the
platform sends, in **Markdown** (compiled to email-safe HTML), from a new page in
the **system settings pages**. Once saved, live sends use the edited template;
every slot ships with a sensible default so nothing is blank out of the box.

## Relationship to plan 18.5 (already shipped)

[`docs/completed/18-admin-experience/18.5-email-template-editor.md`](../../completed/18-admin-experience/18.5-email-template-editor.md)
already delivered a **per-org, raw-HTML** template editor in the **admin console**:

- Tables `settings.email_template_slots` (defaults + merge-field catalog) and
  `settings.org_email_templates` (versioned per-org overrides, one active row per
  `org_id + slot_id`).
- `{{token}}` merge engine, HTML sanitization (`bluemonday`), version history,
  restore, reset-to-default, self-test send.
  (`server/internal/service/emailtemplates`, `server/internal/repos/emailtemplates`,
  `server/internal/httpserver/admin_email_templates.go`,
  `clients/web/src/components/admin/EmailTemplateEditor.tsx`.)
- Feature flag `email_template_editor_enabled`.
- Delivery override `RenderForDelivery` (org template → built-in fallback), wired
  into `server/internal/background/email_worker.go`.

This epic **extends** that infrastructure; it does **not** rebuild it. Three gaps
remain that this epic closes:

1. **Authoring is raw HTML.** The request is a Markdown editor that compiles to
   HTML. → **[ET-2](../../completed/emails/ET-2-markdown-compilation-and-delivery-wiring.md)** (done)
2. **System-scope is missing.** The store is per-org only; the platform's *system*
   emails (magic link, password reset, COPPA) have no editable platform-wide
   default, and there is no "system pages" surface for them. →
   **[ET-1](../../completed/emails/ET-1-system-scope-slots-and-defaults.md)** (done) + **[ET-3](../../completed/emails/ET-3-system-settings-email-templates-page.md)** (done)
3. **System emails bypass templates entirely.** `SendMagicLinkEmail`,
   `SendPasswordResetEmail`, and `SendCoppaConsent*` build bodies inline and never
   call `RenderForDelivery`; `jobqueue_email.go` calls `mail.RenderTemplate`
   directly (skipping overrides). → **[ET-2](../../completed/emails/ET-2-markdown-compilation-and-delivery-wiring.md)** (done)

## Resolution order (target)

For a given slot, at send time:

```
per-org active override  →  system (platform) active override  →  built-in code default
```

## Stories

| ID | Plan | Layer | Est. | Status |
|---|---|---|---|---|
| ET-1 | [System-scope slots, markdown column & default content](../../completed/emails/ET-1-system-scope-slots-and-defaults.md) | Data model / migrations | S | **Done** |
| ET-2 | [Markdown→HTML compilation & routing system emails through templates](../../completed/emails/ET-2-markdown-compilation-and-delivery-wiring.md) | Backend service / delivery | M | **Done** |
| ET-3 | [System Email Templates page in system settings (markdown UI)](../../completed/emails/ET-3-system-settings-email-templates-page.md) | Frontend | M | **Done** |

Ship order is ET-1 → ET-2 → ET-3. ET-1 and ET-2 are independently testable via the
API and a test send; ET-3 is the admin-facing surface on top.

## Slots in scope

Existing (18.5, HTML defaults today; gain markdown defaults in ET-1): `welcome`,
`password_reset`, `grade_posted`, `assignment_created`, `assignment_due_reminder`,
`discussion_reply`, `enrollment_confirmed`.

New system-email slots (ET-1): `magic_link`, `coppa_consent`,
`coppa_consent_confirmation`. (Password reset's slot exists but its *direct* send
path is not wired to it yet — fixed in ET-2.)
