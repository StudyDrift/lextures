# ET-1 — System-scope slots, markdown column & default content

> Implementation plan. Extends [18.5 Email Template Editor](../18-admin-experience/18.5-email-template-editor.md). Epic index: [emails/README.md](../../plan/emails/README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | ET-1 (extends 18.5) |
| **Section** | Admin Experience — Email Templates |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | **DONE** — markdown column, system scope, system-email slots + defaults shipped |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | Platform / Backend |
| **Depends on** | 18.5 (slot + org-template tables shipped) |
| **Unblocks** | ET-2 (compilation + delivery wiring), ET-3 (system settings UI) |

---

## 1. Problem Statement

The platform sends transactional "system" emails — magic-link sign-in, password
reset, and COPPA parent-consent notices — whose copy is hard-coded in Go
(`server/internal/mail/magic_link.go`, `password_reset.go`, `coppa_consent.go`).
The 18.5 editor only knows *per-org* slots and stores **HTML** defaults, so there is
no editable, platform-wide default for these emails and no way to author them in
Markdown. This story adds the data-model foundation: a Markdown source column, a
**system (platform) scope**, and slot definitions + good defaults for the missing
system emails, so ET-2/ET-3 can make them editable end-to-end.

## 2. Goals

- Add a canonical **Markdown source** to slots and to override rows, alongside the
  compiled HTML the delivery path already consumes.
- Introduce a **system (org-less) template scope** so platform-wide defaults can be
  overridden without touching per-org rows.
- Define and seed slots for the currently hard-coded system emails: `magic_link`,
  `coppa_consent`, `coppa_consent_confirmation`.
- Ship a **good Markdown default** for every slot (new and existing) with a
  documented merge-field catalog.
- Preserve the 18.5 resolution semantics while inserting the new system layer:
  per-org → system → built-in code default.

## 3. Non-Goals

- The Markdown→HTML compilation pipeline and delivery wiring (→ ET-2).
- Any UI (→ ET-3).
- Marketing/bulk email, SMS, or per-user template overrides.
- Removing the built-in Go code defaults in `mail/*` — they remain the final
  fallback (ET-2 keeps them as the safety net).

## 4. Personas & User Stories

- **As a platform admin**, I want the magic-link and password-reset emails to exist
  as editable slots with defaults so that I can later customize them without a
  deploy.
- **As a compliance owner (K-12)**, I want the COPPA consent email to be a
  first-class, defaulted slot so its required disclosures are preserved and
  auditable.
- **As a backend engineer**, I want a single documented resolution order
  (org → system → code) so delivery behavior is predictable.
- **As a self-learner tenant admin (single-org)**, I want sensible defaults out of
  the box so no system email is ever blank.

## 5. Functional Requirements

- **FR-1.** The system MUST add a `default_markdown TEXT NOT NULL` column to
  `settings.email_template_slots` holding the canonical Markdown default for each
  slot.
- **FR-2.** The system MUST add a `source_markdown TEXT` column to
  `settings.org_email_templates` so per-org overrides retain their Markdown source
  (the existing `html_body` becomes the compiled artifact, populated by ET-2).
- **FR-3.** The system MUST provide a **system-scope** store for org-less overrides:
  a new table `settings.system_email_templates` mirroring the org table minus
  `org_id`, versioned with a single active row per `slot_id`.
- **FR-4.** The system MUST seed new slots `magic_link`, `coppa_consent`, and
  `coppa_consent_confirmation`, each with a description, merge-field catalog, and
  `default_markdown` / `default_html` / `default_text`.
- **FR-5.** The system MUST backfill `default_markdown` for the seven existing 18.5
  slots so every slot has a Markdown source equivalent to its current HTML default.
- **FR-6.** Each slot's `merge_fields` catalog MUST enumerate exactly the tokens the
  built-in Go sender supplies (e.g. `magic_link` → `link`, `expires_at`,
  `user.first_name`), so ET-3 can validate unknown tokens.
- **FR-7.** The migration MUST be idempotent (`IF NOT EXISTS`, `ON CONFLICT DO
  NOTHING`) and ship a matching `.down.sql` that drops the new table/columns and
  the three new slot rows only.
- **FR-8.** The Go repo layer (`repos/emailtemplates`) MUST expose the new columns
  (`Slot.DefaultMarkdown`) and system-scope accessors (`GetActiveSystem`,
  `SaveSystem`, `ListHistorySystem`, `RestoreSystem`, `ResetSystem`) with the same
  shapes as the existing org accessors.

## 6. Non-Functional Requirements

- **Performance** — Slot reads are low-volume and cacheable; no query touches more
  than one indexed row. System-scope lookup MUST be a single indexed `SELECT`.
- **Security** — No secrets stored. `created_by` references `"user".users(id)`.
  System-scope writes are gated (super-admin) at the API layer in ET-3; the table
  itself carries no tenant data.
- **Privacy & Compliance** — The `coppa_consent` default MUST retain the 16 CFR
  §312.4(c) direct-notice content (what is collected, how used, third-party
  sharing, expiry) currently in `coppa_consent.go`. No PII stored in defaults.
- **Accessibility** — N/A at data layer; default Markdown MUST compile to
  semantic HTML (headings, lists, link text) so ET-2 output is screen-reader
  friendly.
- **Scalability** — `system_email_templates` is bounded by slot count; one active
  row per slot. Indexed by `slot_id`.
- **Reliability** — Built-in Go defaults remain the terminal fallback; a missing
  slot row never blocks a send (ET-2 handles fallback).
- **Observability** — No new metrics here; ET-2 adds render/compile counters.
- **Maintainability** — Adding a future system email = one slot-seed row +
  merge-field catalog; documented in the runbook (§17).
- **Internationalization** — Defaults authored in English; `merge_fields` values
  are human descriptions surfaced in ET-3. Per-locale defaults are out of scope
  (tracked as an open question).
- **Backward compatibility** — Existing `org_email_templates` rows keep working;
  `source_markdown` is nullable and backfilled lazily (ET-2 populates on next save).

## 7. Acceptance Criteria

- **AC-1.** *Given* the migration is applied, *When* `SELECT default_markdown FROM
  settings.email_template_slots` runs, *Then* every slot row returns non-empty
  Markdown.
- **AC-2.** *Given* the migration is applied, *When* the three new slot ids are
  queried, *Then* `magic_link`, `coppa_consent`, and `coppa_consent_confirmation`
  each return a description, non-empty `merge_fields`, and defaults.
- **AC-3.** *Given* `settings.system_email_templates`, *When* two rows are inserted
  for the same `slot_id` with `is_active = true`, *Then* the unique partial index
  rejects the second (one active system row per slot).
- **AC-4.** *Given* the down migration, *When* it is applied, *Then* the new table,
  the two new columns, and the three seeded slot rows are removed and the seven
  original 18.5 slots remain intact.
- **AC-5.** *Given* `repos/emailtemplates.GetActiveSystem(ctx, "magic_link")` with
  no override present, *When* called, *Then* it returns `(nil, nil)` (no error),
  signaling "fall through to slot default".

## 8. Data Model

Migration `server/migrations/373_email_templates_markdown_system_scope.sql`
(+ `.down.sql`). Number is the next free index (372 is the latest).

```sql
-- Markdown source alongside compiled HTML (ET-1).
ALTER TABLE settings.email_template_slots
    ADD COLUMN IF NOT EXISTS default_markdown TEXT NOT NULL DEFAULT '';
ALTER TABLE settings.org_email_templates
    ADD COLUMN IF NOT EXISTS source_markdown TEXT;

-- System (platform) scope: org-less overrides, one active row per slot.
CREATE TABLE IF NOT EXISTS settings.system_email_templates (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slot_id         TEXT NOT NULL REFERENCES settings.email_template_slots (id),
    source_markdown TEXT NOT NULL,
    html_body       TEXT NOT NULL,   -- compiled + sanitized (populated by ET-2)
    text_body       TEXT,
    reply_to        TEXT,
    sender_name     TEXT,
    created_by      UUID REFERENCES "user".users (id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    is_active       BOOLEAN NOT NULL DEFAULT true
);
CREATE UNIQUE INDEX IF NOT EXISTS system_email_templates_active
    ON settings.system_email_templates (slot_id) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS system_email_templates_history
    ON settings.system_email_templates (slot_id, created_at DESC);

-- New system-email slots (defaults abbreviated; see repo seed for full copy).
INSERT INTO settings.email_template_slots
    (id, description, merge_fields, default_html, default_text, default_markdown)
VALUES
  ('magic_link', 'Passwordless sign-in link',
   '{"user.first_name":"Recipient first name","link":"One-time sign-in link","expires_at":"Link expiration"}'::jsonb,
   '', '', 'Sign in to your account without a password.\n\n[Sign in now]({{link}})\n\nThis link works once and expires {{expires_at}}. If you did not request it, ignore this email.'),
  ('coppa_consent', 'COPPA parent consent notice (16 CFR §312.4(c))',
   '{"student.name":"Student first name","org.name":"Organization name","link":"Consent review link","expires_at":"Consent link expiration"}'::jsonb,
   '', '', '## Parent permission required\n\nA school account has been created for **{{student.name}}** on {{org.name}}.\n\nUnder COPPA we need your permission before activating the account.\n\n- **What we collect:** first name, school ID, course progress, quiz responses\n- **How we use it:** deliver coursework and track learning progress\n- **Third-party sharing:** none without your consent\n\n[Review & give permission]({{link}})\n\nThis link expires {{expires_at}}. If unexpected, ignore this — no account activates without approval.'),
  ('coppa_consent_confirmation', 'COPPA consent confirmed',
   '{"student.name":"Student first name","org.name":"Organization name"}'::jsonb,
   '', '', 'You have given permission for **{{student.name}}** to use {{org.name}}.\n\nTheir account is now active. You can manage privacy settings or revoke permission any time by contacting your school.')
ON CONFLICT (id) DO NOTHING;

-- Backfill default_markdown for the seven existing 18.5 slots.
UPDATE settings.email_template_slots SET default_markdown =
  'Welcome to **{{org.name}}**, {{user.first_name}}!\n\nWe are glad you joined. [Sign in to get started]({{link}}).'
  WHERE id = 'welcome' AND default_markdown = '';
UPDATE settings.email_template_slots SET default_markdown =
  'Hi {{user.first_name}},\n\nWe received a request to reset your password. [Reset your password]({{link}}).\n\nThis link expires {{expires_at}}. If you did not request this, ignore this email.'
  WHERE id = 'password_reset' AND default_markdown = '';
-- …grade_posted, assignment_created, assignment_due_reminder, discussion_reply,
--   enrollment_confirmed backfilled identically (Markdown mirrors current HTML).
```

- **Indexes & constraints:** partial unique index on `(slot_id) WHERE is_active`
  guarantees one active system override per slot; history index for the ET-3
  version drawer.
- **Backfill strategy:** `default_markdown` backfilled in-migration for the 7
  existing slots; `org_email_templates.source_markdown` left NULL and populated on
  next save by ET-2 (existing HTML remains authoritative until then).
- **Naming:** follows repo convention `server/migrations/NNN_*.sql` (+ `.down.sql`).

## 9. API Surface

No new HTTP routes in this story (ET-3 adds them). Go repo additions in
`server/internal/repos/emailtemplates/repo.go`:

```go
type Slot struct { /* … */ DefaultMarkdown string }

// System-scope mirror of the existing org accessors.
func GetActiveSystem(ctx, pool, slotID string) (*SystemVersion, error)
func SaveSystem(ctx, pool, SaveSystemInput) (*SystemVersion, error)   // deactivates prior active row in a tx
func ListHistorySystem(ctx, pool, slotID string) ([]SystemVersion, error)
func RestoreSystem(ctx, pool, slotID, versionID) (*SystemVersion, error)
func ResetSystem(ctx, pool, slotID string) error                     // deactivate active row → slot default
```

`SystemVersion` mirrors `OrgVersion` (adds `SourceMarkdown`, drops `OrgID`).
`OrgVersion` gains `SourceMarkdown *string`.

## 10. UI / UX

None in this story. ET-3 consumes the new columns and system-scope accessors.

## 11. AI / ML Considerations

Not applicable.

## 12. Integration Points

- `server/migrations/` — new migration pair.
- `server/internal/repos/emailtemplates/repo.go` — new columns + system accessors
  + `SystemVersion` type; update `ListSlots`/`GetSlot` scans to select
  `default_markdown`.
- `server/internal/mail/coppa_consent.go`, `magic_link.go`, `password_reset.go` —
  read only in this story to lift exact default copy/merge-field names; the code
  defaults stay until ET-2 rewires delivery.
- Consumers in ET-2 (`service/emailtemplates`) and ET-3 (`httpserver`).

## 13. Dependencies & Sequencing

- Must ship after: 18.5 (tables exist).
- Must ship before: ET-2, ET-3.
- Shared infra: Postgres migration runner; no new services.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Backfilled Markdown drifts from current HTML output | M | L | ET-2 golden test compiles each `default_markdown` and diffs against a snapshot of current sender HTML |
| COPPA default loses a required disclosure | L | H | Default seeded verbatim from `coppa_consent.go`; compliance review of the seed row in PR |
| Two active system rows for one slot | L | M | Partial unique index enforces single active; `SaveSystem` flips prior active in one tx |
| Merge-field catalog omits a token the sender supplies | M | M | ET-1 catalog derived directly from each sender's `vars`; ET-2 golden test asserts every supplied token is in the catalog |

## 15. Rollout Plan

- Feature flag: reuse `email_template_editor_enabled` (already present) — data
  layer ships dark; nothing reads system scope until ET-2.
- Sequencing: schema → seed/backfill → repo accessors → (ET-2 code) → (ET-3 flag
  flip for pilot).
- Dogfood: apply to staging; verify slot rows and a manual `SaveSystem` round-trip.
- Rollback: `.down.sql` drops the table/columns/new slots; org rows untouched.

## 16. Test Plan

- **Unit** — repo tests for `GetActiveSystem` (empty → nil,nil), `SaveSystem`
  (active flip), `RestoreSystem`, `ResetSystem`; `ListSlots` returns
  `DefaultMarkdown`.
- **Integration** — apply up then down migration on a scratch DB; assert slot
  count and that the 7 original slots survive the down migration.
- **End-to-end** — deferred to ET-2/ET-3 (needs compile + API).
- **Security** — assert `system_email_templates` has no `org_id` and cannot be
  written without `created_by` FK integrity.
- **Accessibility** — N/A.
- **Performance / load** — N/A (bounded rows).
- **Manual exploratory** — inspect seeded `coppa_consent.default_markdown` renders
  to a complete notice (in ET-2 preview).

## 17. Documentation & Training

- Internal runbook: "Adding a new system-email slot" — seed row shape, merge-field
  catalog rule (must match sender `vars`), resolution order.
- Update [18.5 plan](../18-admin-experience/18.5-email-template-editor.md)
  cross-reference note pointing at this epic.

## 18. Open Questions

1. Per-locale defaults — one `default_markdown` per slot now, or a
   `(slot_id, locale)` matrix? (Recommend: single default now; revisit with i18n.)
2. Should `system_email_templates` reuse `org_email_templates` with a nullable
   `org_id` instead of a dedicated table? (Recommend dedicated table — keeps the
   per-org unique index and FK clean; documented trade-off.)
3. Retention of system-scope version history (unbounded vs. last N)? (Recommend
   unbounded initially; low volume.)

## 19. References

- `server/migrations/352_email_template_editor.sql` (18.5 schema + current seeds).
- `server/internal/repos/emailtemplates/repo.go` (accessors to mirror).
- `server/internal/mail/{magic_link,password_reset,coppa_consent}.go` (source copy
  + merge tokens).
- Related: [ET-2](ET-2-markdown-compilation-and-delivery-wiring.md),
  [ET-3](ET-3-system-settings-email-templates-page.md).
