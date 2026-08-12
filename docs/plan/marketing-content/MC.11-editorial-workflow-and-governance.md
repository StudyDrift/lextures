# MC.11 — Editorial Workflow, Review & Governance

> Implementation plan. Source: [docs/plan/marketing-content/README.md](README.md) §Plans; supersedes
> the file-based editorial tooling in `www/scripts/editorial*.mjs` and `check-help-freshness.mjs` for
> DB-sourced content.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | MC.11 |
| **Section** | MC — Marketing Content Platform |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | PARTIAL — an editorial calendar, pillar model, gap analysis and a 180-day freshness check exist as Node scripts over files; none of it is visible to a content expert or enforceable on published content |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Docs/Content + Web platform |
| **Depends on** | MC.2, MC.10 |
| **Unblocks** | MC.15 |

---

## 1. Problem Statement

Pull requests currently provide the governance: someone reviews, CI lints, git records who changed
what. Moving content into a database removes all three unless we rebuild them deliberately. At the
same time, the editorial system we already designed — six content pillars with article floors, a
calendar, gap analysis, a 180-day freshness rule for help articles, brief references and review
owners — lives in scripts that read files and print to a terminal. If content moves and this does
not, we lose review, lose freshness enforcement, and hand the content team a publishing tool with no
editorial discipline attached.

## 2. Goals

- Make review a real workflow in the product: assignment, queue, approve/request-changes with notes,
  and a visible audit trail.
- Enforce freshness the way the file-based check did, but continuously and per-article: review due
  dates, overdue queues, and a health report that names owners.
- Bring the editorial calendar and pillar/gap model into the workspace so planning and publishing
  share one system.
- Govern the exceptions: every publish override, every retired author, every redirect and every
  deletion is recorded and reportable.
- Notify people about what they own — review requests, approaching due dates, failed scheduled
  publishes — without becoming noise.

## 3. Non-Goals

- No inline block-level comments/annotations on drafts (change notes + review notes only for v1).
- No content approval routing rules engine (multi-step, conditional approvers). One review step.
- No replacement of `www/scripts/editorial*.mjs` for file-based pages, which remain until MC.15 and
  keep working.
- No SEO analytics/rank tracking — that is [SEO.15](../seo/SEO.15-measurement-search-console-and-ai-share-of-voice.md).
- No AI content scoring or suggestion of what to write next.

## 4. Personas & User Stories

- **As a content expert**, I want to send an article to a named reviewer and know when they respond.
- **As an editor**, I want one queue of everything waiting on me, oldest first.
- **As a help-center owner**, I want to see which articles are past their review date and who owns
  them, so accuracy does not decay silently.
- **As a marketing lead**, I want to see the calendar of what is scheduled and which pillars are
  under-served, so planning is grounded in what exists.
- **As a compliance reviewer**, I want a report of every publish that bypassed the quality gate, with
  the justification and who approved it.
- **As an author**, I do not want an inbox full of content notifications; I want the ones about my
  work.

## 5. Functional Requirements

- **FR-1.** Articles MUST support an explicit reviewer assignment (`reviewer_slug` plus an assigned
  user), set when submitting for review or by an editor, and MUST appear in that reviewer's queue.
- **FR-2.** A **Review queue** view MUST list articles in `in_review`, sorted by submission time,
  showing author, kind, category, submitted-at, quality score and blocking findings count. It MUST be
  filterable to "assigned to me" and MUST be the default landing tab for `…:review` holders.
- **FR-3.** Approve and Request changes MUST require nothing more than one click and (for request
  changes) a note of ≥ 10 characters, and MUST record both on a `content_reviews` row and a revision.
- **FR-4.** Every published article MUST carry a `review_due_on`. When absent, it MUST default on
  publish to `published_at + review_interval_days`, configurable per kind (default 180 for `doc`
  — matching today's staleness rule — and 365 for `blog`).
- **FR-5.** An **Overdue** filter and a **Content health** view MUST show articles past
  `review_due_on`, grouped by owner and category, with the same 10% staleness threshold surfaced as a
  status ("12 of 70 help articles are stale — threshold 10%").
- **FR-6.** A "Mark reviewed" action MUST set `reviewed_at = today`, extend `review_due_on` by the
  interval, and record a revision with `change_note='reviewed, no content change'` — without
  requiring an edit.
- **FR-7.** An **Editorial calendar** view MUST show scheduled publishes and planned items on a month
  grid, with drag-free scheduling (date picker) and links to each article.
- **FR-8.** Planned items MUST be representable without a full draft: a `content_briefs` record with
  title, target pillar/cluster, primary question, owner, target date and brief reference — the
  database equivalent of today's `docs/plan/seo/briefs/` and calendar rows.
- **FR-9.** A **Pillar coverage** panel MUST report, per pillar, published article count against the
  configured floor (the six pillars and floors currently in `editorial-core.mjs`), and MUST list gaps.
- **FR-10.** Publish overrides (MC.10 FR-14) MUST require a typed justification, MUST be recorded, and
  MUST appear in a **Governance** report listing date, article, actor, rules bypassed and
  justification.
- **FR-11.** Notifications MUST be emitted (in-app, using the existing notification system, with an
  email option) for: review requested (to the reviewer), changes requested (to the author), approved
  (to the author), scheduled publish failed (to author + publisher), review due in 14 days (to the
  owner), review overdue weekly digest (to the content owner group).
- **FR-12.** Notification preferences MUST be respected via the existing user notification settings;
  digests MUST be one message, not one per article.
- **FR-13.** A scheduled job `marketing_content_review_sweep` (daily) MUST compute due/overdue states,
  emit notifications and refresh the health report snapshot.
- **FR-14.** A scheduled job `marketing_content_link_health` (weekly) MUST check external links in
  published articles (HEAD with fallback GET, 10 s timeout, respecting robots) and record failures for
  the health view; it MUST NOT block publishing.
- **FR-15.** A revision-retention policy MUST prune non-published revisions older than 18 months
  (configurable), keeping every revision that was ever published and the last 20 revisions per
  article regardless of age.
- **FR-16.** All governance views MUST be permission-gated: health and calendar require `…:view`;
  overrides report requires `…:review`; retention configuration requires `…:admin`.
- **FR-17.** Author retirement MUST be a first-class action (`…:admin`): it sets author status to
  `retired`, keeps bylines as plain text, and lists their articles for reassignment — the DB
  equivalent of today's author-registry policy.

## 6. Non-Functional Requirements

- **Performance** — Review queue and health views p95 < 400 ms at 500 articles (server-side
  aggregation, no N+1). The health snapshot is precomputed daily; the view reads the snapshot plus a
  live delta.
- **Security** — Governance data reveals internal process; all views are permission-gated. Link
  health fetches are outbound-only, use a distinct User-Agent, and never follow redirects to internal
  hosts (SSRF guard: block private ranges).
- **Privacy & Compliance** — Reviewer/author names are staff data. Notification content includes
  article titles only. The overrides report is retained as an audit artefact per the admin-audit
  retention policy.
- **Accessibility** — Calendar must be operable without a mouse and readable by screen readers: it is
  a table with dates as row headers, not a canvas; "today" and "scheduled" are text plus visual. The
  health report uses accessible status text, and any chart has an accessible table equivalent
  (per the repo's visualization guidance).
- **Scalability** — Sweep jobs process hundreds of rows; link health is bounded (max 500 links/run,
  8 concurrent, resumable).
- **Reliability** — Sweeps are idempotent; notifications deduplicate per (article, type, day) so a
  retried job does not spam. Link-health failures are advisory and never change article state.
- **Observability** — `marketing_content_reviews_total{action}`, `…_overdue_articles`,
  `…_override_publishes_total`, `…_link_failures_total`, `…_notifications_sent_total{type}`; alert if
  overdue ratio exceeds the configured threshold for 7 consecutive days.
- **Maintainability** — Pillars, floors and intervals are configuration (a settings row seeded from
  `editorial-core.mjs`'s current values), not code constants, so the content team can adjust them
  without a deploy.
- **Internationalization** — Dates and intervals respect the viewer's timezone; digests are sent in
  the recipient's locale; per-locale review intervals are supported for MC.14.
- **Backward compatibility** — `www/scripts/editorial*.mjs` and `check-help-freshness.mjs` keep
  working over remaining file-based content; MC.15 removes their help-center scope when the files go.

## 7. Acceptance Criteria

- **AC-1.** *Given* an author submits an article for review with a reviewer assigned, *when* the
  reviewer opens the workspace, *then* it appears in their queue and they received a notification.
- **AC-2.** *Given* a reviewer requests changes with a note, *when* the author opens the article,
  *then* the status is `changes_requested`, the note is visible, and the author was notified.
- **AC-3.** *Given* an article published today with `kind='doc'` and no explicit due date, *when*
  published, *then* `review_due_on = today + 180 days`.
- **AC-4.** *Given* 12 of 70 help articles are past due, *when* the health view loads, *then* it
  reports "12 of 70 (17%) — above the 10% threshold" with the list grouped by owner.
- **AC-5.** *Given* "Mark reviewed" on an overdue article, *when* clicked, *then* `reviewed_at` is
  today, `review_due_on` moves forward by the interval, a revision is recorded, and no content
  change is required.
- **AC-6.** *Given* three scheduled articles next month, *when* the calendar loads, *then* they appear
  on their dates with links, and keyboard navigation reaches each one.
- **AC-7.** *Given* the six configured pillars, *when* the coverage panel loads, *then* each shows
  published count vs floor and pillars below floor are listed as gaps.
- **AC-8.** *Given* a publish override with justification, *when* the governance report loads, *then*
  the entry appears with actor, rules bypassed, justification and timestamp.
- **AC-9.** *Given* the daily sweep, *when* it runs twice in a day, *then* no duplicate notifications
  are sent (dedupe by article+type+day).
- **AC-10.** *Given* an article containing a dead external link, *when* the weekly link job runs,
  *then* the failure appears in the health view and the article's state is unchanged.
- **AC-11.** *Given* an author is retired, *when* the action completes, *then* their public author
  page 404s, existing bylines render as plain text, and their articles are listed for reassignment.
- **AC-12.** *Given* revision retention configured at 18 months, *when* the prune job runs, *then* no
  ever-published revision and no article's last 20 revisions are deleted.

## 8. Data Model

Migration `482_marketing_content_editorial.sql` (indicative number):

```sql
CREATE TABLE marketing.content_reviews (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    article_id   UUID NOT NULL REFERENCES marketing.content_articles (id) ON DELETE CASCADE,
    revision_no  INTEGER NOT NULL,
    action       TEXT NOT NULL CHECK (action IN ('submitted','approved','changes_requested','reviewed')),
    reviewer_id  UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    actor_id     UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    note         TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_mc_reviews_article ON marketing.content_reviews (article_id, created_at DESC);

CREATE TABLE marketing.content_briefs (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title          TEXT NOT NULL,
    kind           TEXT NOT NULL CHECK (kind IN ('blog','doc')),
    pillar         TEXT NOT NULL DEFAULT '',
    cluster        TEXT NOT NULL DEFAULT '',
    primary_question TEXT NOT NULL DEFAULT '',
    owner_id       UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    target_date    DATE,
    brief_ref      TEXT NOT NULL DEFAULT '',
    article_id     UUID REFERENCES marketing.content_articles (id) ON DELETE SET NULL,
    status         TEXT NOT NULL DEFAULT 'planned'
                   CHECK (status IN ('planned','in_progress','published','dropped')),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE marketing.content_link_health (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    article_id  UUID NOT NULL REFERENCES marketing.content_articles (id) ON DELETE CASCADE,
    url         TEXT NOT NULL,
    status_code INTEGER,
    error       TEXT,
    checked_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (article_id, url)
);

CREATE TABLE marketing.content_overrides (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    article_id    UUID NOT NULL REFERENCES marketing.content_articles (id) ON DELETE CASCADE,
    revision_no   INTEGER NOT NULL,
    actor_id      UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    rules         TEXT[] NOT NULL,
    justification TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE marketing.content_health_snapshots (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    taken_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    payload      JSONB NOT NULL          -- counts by kind/status/pillar, overdue list, link failures
);
```

Settings (platform): `marketing_review_interval_doc_days` (180),
`marketing_review_interval_blog_days` (365), `marketing_stale_threshold_pct` (10),
`marketing_revision_retention_months` (18), `marketing_pillars` (JSONB seeded from
`editorial-core.mjs`).

**Backfill:** MC.6 sets `review_due_on` for imported articles from `reviewDue` where present,
otherwise `content_updated_at + interval`, so the health view is meaningful on day one.

## 9. API Surface

| Verb | Path | Permission | Notes |
|---|---|---|---|
| GET | `/api/v1/admin/marketing/reviews/queue` | `…:review` | `assignedToMe`, cursor |
| POST | `/api/v1/admin/marketing/articles/{id}/review` | `…:review` | `{action, note, reviewerId?}` |
| POST | `/api/v1/admin/marketing/articles/{id}/mark-reviewed` | `…:review` | extends due date |
| GET | `/api/v1/admin/marketing/health` | `…:view` | snapshot + live delta |
| GET | `/api/v1/admin/marketing/calendar?from=&to=` | `…:view` | scheduled + briefs |
| GET/POST | `/api/v1/admin/marketing/briefs` · PATCH/DELETE `/{id}` | `…:view` / `…:author` | planning items |
| GET | `/api/v1/admin/marketing/pillars` | `…:view` | coverage vs floors |
| GET | `/api/v1/admin/marketing/overrides` | `…:review` | governance report |
| GET | `/api/v1/admin/marketing/link-health` | `…:view` | failures by article |
| PATCH | `/api/v1/admin/marketing/settings` | `…:admin` | intervals, thresholds, pillars |
| POST | `/api/v1/admin/marketing/authors/{slug}/retire` | `…:admin` | retirement + reassignment list |

Scheduled jobs (registered in `internal/scheduler/config.go`): `marketing_content_review_sweep`
(`0 7 * * *`), `marketing_content_link_health` (`0 5 * * 1`), `marketing_content_revision_prune`
(`30 3 * * 0`).

## 10. UI / UX

New tabs inside the MC.9 workspace: **Queue**, **Calendar**, **Health**, **Governance**.

1. **Queue** — table of `in_review` items with "Assigned to me" toggle; opening an item lands in the
   editor with the reviewer action bar (Approve / Request changes) and a diff against the last
   published revision.
2. **Calendar** — month grid; each cell lists scheduled articles and planned briefs; a side panel
   shows the selected day; "Add brief" opens a small form.
3. **Health** — three cards (Freshness, Coverage, Links) over a table of affected articles with owner
   and action links; the freshness card states the ratio and threshold in words.
4. **Governance** — chronological list of overrides and taxonomy changes with filters by actor and
   date.
5. **States** — empty queue ("Nothing waiting on you"), empty calendar month, health all-clear
   ("Everything is within its review window"), loading skeletons, error retry, permission-limited
   (tabs hidden rather than disabled).
6. **Responsive** — calendar becomes an agenda list below `md`; health cards stack; tables become
   card lists.
7. **Accessibility** — calendar is a `<table>` with `<th scope="col">` weekday headers and dates as
   row content, each day cell containing a list; "today" is announced in text; status is never
   colour-only; the queue's live region announces "3 articles waiting"; any coverage chart ships with
   an equivalent data table.
8. **Copy & i18n** — `marketingContent.review.*`, `.health.*`, `.calendar.*`, `.governance.*`,
   including the stale-ratio sentence and the override justification prompt.

## 11. AI / ML Considerations

Not AI-touching. Explicit governance stance: if AI assistance is ever added to authoring, the
override report and the review workflow are where its use becomes visible — AI-assisted articles must
carry a disclosure field surfaced in the review queue.

## 12. Integration Points

- **Internal modules:** `internal/service/marketingeditorial` (new), `internal/scheduler/config.go`,
  `internal/workers`, `internal/service/notificationevents` / existing notification pipeline,
  `internal/service/adminaudit`, `internal/repos/marketingcontent`.
- **Client:** new tabs under `pages/admin/marketing-content/`, shared components with MC.9.
- **Superseded scripts:** `www/scripts/editorial.mjs`, `editorial-core.mjs`,
  `check-help-freshness.mjs` — their rules move server-side; the scripts remain for file-based pages
  until MC.15.
- **External:** outbound HTTP for link health (with SSRF guard and robots respect).

## 13. Dependencies & Sequencing

- Must ship after: MC.2 (transitions), MC.10 (editor hosts the review actions), MC.6 (real data for
  health to be meaningful).
- Must ship before: MC.15 (cutover requires the governance replacement to exist before the file
  scripts lose their scope).
- Shared infra: scheduler, notifications, email.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Notification fatigue kills adoption | **H** | M | Digest for due/overdue; per-event notifications only for direct assignments; respects existing notification preferences; measured by opt-out rate |
| Review becomes a bottleneck of one person | M | M | Queue shows unassigned items to all `…:review` holders; "assigned to me" is a filter, not a wall |
| Freshness rule generates busywork ("mark reviewed" clicking) | M | M | Mark-reviewed records a revision and an actor, so the click is accountable; health view shows who marked what without a content change, making rubber-stamping visible |
| Link health false positives (bot-blocking sites) | **H** | L | Advisory only; per-domain allowlist for known blockers; two consecutive failures before flagging |
| SSRF via link health | L | H | Block private/loopback ranges, no redirects to non-public hosts, timeouts, distinct egress user-agent |
| Governance views duplicate the admin audit log | M | L | They are filtered projections of the same audit data, linked from the audit page rather than a parallel store |

## 15. Rollout Plan

- **Feature flag:** `ff_marketing_content`; tabs appear per permission. Jobs are `DefaultEnabled:
  true` but no-op when the flag is off.
- **Sequencing:** review workflow (queue + actions + notifications) → freshness/health → calendar and
  briefs → governance report → retention prune (last, after retention settings are agreed).
- **Dogfood:** run one full editorial cycle in staging — plan a brief, write it, review it, publish
  it, mark another reviewed — before production enable.
- **GA criteria:** all ACs; the content team runs a week of real review through the queue; overdue
  ratio visible and accurate against the file-based checker's result for the same corpus.
- **Rollback:** disable the sweep jobs (admin scheduler UI) and hide the tabs by flag; article data is
  unaffected.

## 16. Test Plan

- **Unit** — due-date computation per kind; stale-ratio math; pillar coverage; dedupe key for
  notifications; retention selection (never prune published/last-20); SSRF guard for link checks.
- **Integration (DB)** — submit→review→approve→publish with review rows and notifications;
  mark-reviewed revision creation; sweep idempotency across two runs; health snapshot generation;
  prune job with a crafted revision history.
- **End-to-end** — `e2e/tests/marketing-content-review.spec.ts`: author submits, reviewer sees queue,
  requests changes, author fixes, reviewer approves, publisher publishes; calendar shows the
  scheduled item; health lists an overdue article and mark-reviewed clears it.
- **Security** — permission matrix per tab and endpoint; SSRF attempts against private ranges;
  override justification required and recorded; notification content leaks no draft body.
- **Accessibility** — axe on all four tabs; calendar keyboard navigation script; screen-reader script
  for the queue live region and health status sentences; chart/table equivalence check.
- **Performance / load** — health view at 500 articles; link-health job over 500 links within the
  window; notification digest generation for 50 recipients.
- **Manual exploratory** — retire an author with published articles; schedule a publish that fails
  validation; simulate a week of overdue growth and verify the alert.

## 17. Documentation & Training

- Help articles: "Reviewing marketing content", "Keeping help articles fresh", "Planning content with
  briefs and the calendar".
- Internal: governance policy — what justifies an override, who may retire an author, how retention
  is configured.
- Update `www/docs/editorial-process.md` to describe the in-product workflow and mark the script-based
  process as applying to file-based pages only.

## 18. Open Questions

1. Do we need inline comments on drafts for real review, or are change notes enough? (Proposed: start
   with notes; measure whether reviewers ask for inline comments within a month.)
2. Should "mark reviewed" require a checklist (links checked, screenshots current, feature still
   exists) rather than a single click? (Proposed: yes for `doc` — a 3-item checklist; decide with the
   content team.)
3. Where do pillars live long-term — settings JSON or a table with its own CRUD? (Proposed: settings
   JSON now; promote to a table if the team edits them often.)
4. Should the overdue threshold breach block publishing new content until remediated? (Proposed: no —
   punishing new work for old debt is counterproductive; alert instead.)

## 19. References

- Files this work touches: `server/internal/service/marketingeditorial/*`,
  `server/internal/scheduler/config.go`, `server/migrations/482_*`,
  `clients/web/src/pages/admin/marketing-content/{queue,calendar,health,governance}.tsx`.
- Superseded logic: `www/scripts/editorial-core.mjs` (pillars, floors, priority score, refresh due),
  `www/scripts/check-help-freshness.mjs` (180-day rule, 10% threshold),
  `docs/plan/seo/briefs/`.
- Related plans: [MC.2](MC.2-authoring-api-and-revisions.md), [MC.10](MC.10-article-editor.md),
  [MC.15](MC.15-rollout-cutover-and-decommission.md), SEO.8 (editorial engine), SEO.16 (governance).
