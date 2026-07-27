# CT.6 — Content Tools: Grounded Context Service & Web-Link Ingestion

> Implementation plan. Source: new capability — interactive tools inside content sections. Folder overview: [README](../../plan/content_tools/README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | CT.6 |
| **Section** | Content Tools (CT) |
| **Severity** | BLOCKER (for every AI-backed tool) |
| **Markets** | K12 / HE / HS |
| **Status (today)** | DONE |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | AI platform team |
| **Depends on** | CT.1, CT.3 |
| **Unblocks** | CT.10 (Ask Questions), CT.20 (Explain It Back), CT.18 hints, and every future AI tool |

---

## 1. Problem Statement

An AI tool placed inside a section is only useful if it knows *what the section says*. Today a model
call would receive nothing but the student's question: the tutor (`service/aitutor`) is scoped to a
course title, and `service/notebookrag` indexes notebooks, not the activity a learner is currently
reading. Worse, the most load-bearing context in a real lesson is often **behind the links the author
pasted into the content** — a standards page, a news article, a lab protocol — and nothing in the
platform fetches those. Every AI tool would otherwise reimplement context assembly, link fetching and
SSRF defence, badly and differently. This story ships the shared **context service**: one place that
turns "this tool instance" into a grounded, cited, budgeted, safe context pack.

## 2. Goals

- Assemble a **context pack** for any tool instance: the section it lives in, the surrounding
  activity, the course frame, attached course files, and the links the author included.
- Ingest **web links** safely: SSRF-guarded fetch, main-content extraction, caching, revalidation,
  and per-course allow/deny policy.
- Expose ingestion as a **model tool call** (`fetch_link`) so a model pulls only the links it needs,
  with a server-orchestrated fallback for providers without tool-calling.
- Make every AI answer **citable**: each returned passage carries a source (section, file, link) that
  tools can render, so students can verify rather than trust.
- Enforce disclosure, budgets and PII handling in one place, so no tool can call a model unmediated.

## 3. Non-Goals

- General web search (nothing crawls beyond links the author explicitly placed in the content, plus
  course files). A search-backed tool is a separate, policy-heavy story.
- Replacing `service/notebookrag` or the course-wide tutor — CT.6 is activity-scoped and reuses their
  chunking/embedding utilities where they fit.
- Per-tool prompt design — each AI tool owns its own prompt and evals; CT.6 owns the *context* and the
  *rails*.
- Multimodal ingestion of video/audio transcripts (deferred; CT.19 supplies its own captions context).

## 4. Personas & User Stories

- **As a student**, I want to ask a question about *this page* and get an answer that quotes the page
  and the article my teacher linked, so that I trust and can check the answer.
- **As an instructor**, I want the AI grounded in what I actually assigned so that it does not
  contradict my materials or wander off-syllabus.
- **As an instructor**, I want to see which links were ingested and re-ingest one when the source
  changes so that stale content is my choice, not a mystery.
- **As a district IT admin**, I want link fetching restricted to public HTTP(S) hosts with no access
  to internal networks so that a pasted URL cannot become an SSRF probe.
- **As a privacy officer**, I want student text redacted before it leaves our infrastructure, and I
  want a record of what was sent, so that a subject-access request can be answered honestly.
- **As a finance owner**, I want per-course budgets so that a class of 400 asking unlimited questions
  cannot produce a surprise invoice.

## 5. Functional Requirements

- **FR-1.** The system MUST provide `contenttools/context.Build(ctx, instanceID, opts)` returning a
  **context pack**: ordered, typed, token-budgeted segments with source attribution.
- **FR-2.** The pack MUST include, in priority order: the instance's own section body, sibling sections
  of the same activity, the activity title/description, the module and course titles, instructor-pinned
  notes from the tool config, referenced course files (text-extractable types), and referenced links.
- **FR-3.** Link discovery MUST parse the host activity's Markdown for external `http(s)` links and
  MUST also honour explicit links listed in the tool's config.
- **FR-4.** Link fetching MUST be SSRF-guarded: scheme allowlist (`https`, `http` only), DNS resolution
  pinned and re-checked after redirects, private/link-local/metadata ranges denied (IPv4 and IPv6),
  redirect depth ≤ 3, response ≤ 5 MB, timeout ≤ 10 s, and no credentials, cookies or custom auth.
- **FR-5.** Fetch MUST honour `robots.txt` for the fetching agent and MUST send a descriptive
  `User-Agent` identifying Lextures with a contact URL.
- **FR-6.** Fetched HTML MUST be reduced to main content (boilerplate stripped), converted to text,
  chunked, and cached with `etag`/`last-modified` for conditional revalidation.
- **FR-7.** The cache MUST be keyed by URL hash + extraction version, scoped per organization, with a
  default TTL of 7 days and instructor-triggered re-ingest.
- **FR-8.** PDFs and plain text MUST be supported alongside HTML; unsupported types MUST be recorded
  as "not ingestible" with a reason surfaced to the instructor, never silently skipped.
- **FR-9.** The system MUST expose a model-facing tool `fetch_link(url)` returning extracted passages,
  usable by providers that support tool calling.
- **FR-10.** For providers without tool calling, the system MUST fall back to **server-orchestrated
  retrieval**: rank links by relevance to the learner's query, pre-fetch the top *k* (default 3), and
  inject their passages into the prompt — producing the same citations either way.
- **FR-11.** Retrieval MUST return passages with stable citation handles
  (`{kind: 'section'|'file'|'link', id, title, url?, loc}`) that a tool renders as footnotes.
- **FR-12.** Every model call made through CT.6 MUST pass `aigateway.Evaluate` with the tool's declared
  `ai.featureId` first; a denial MUST return a typed, user-friendly error and MUST NOT call a provider.
- **FR-13.** Student-authored text MUST be PII-redacted (reusing `aitutor.RedactPII`) before leaving
  the platform, and the redacted form MUST be what is logged.
- **FR-14.** The system MUST enforce budgets at three levels: per-request token cap, per-user daily
  call cap, and per-course monthly spend cap; exceeding a cap returns a typed error naming the limit.
- **FR-15.** Every call MUST be logged to `analytics.ai_usage_log` with feature id, instance id, token
  counts and estimated cost, consistent with every other AI feature.
- **FR-16.** Course/org policy MUST be able to deny link ingestion entirely, restrict it to an allowlist
  of hosts, or allow all public hosts (default: allow public hosts, deny on org opt-out).
- **FR-17.** Instructors MUST be able to inspect the ingested corpus for an activity (per link: status,
  fetched-at, size, extraction quality, error) and force re-ingest or exclude a link.
- **FR-18.** Context assembly MUST respect the **served** content variant when ACE is active, so an
  AI tool grounds on the text that learner actually saw.

## 6. Non-Functional Requirements

- **Performance** — Warm context pack build p95 ≤ 150 ms. Cold link ingestion is asynchronous:
  a first request returns available context immediately and marks pending links, never blocking a
  student behind a slow third-party site. Cached link retrieval p95 ≤ 60 ms.
- **Security** — SSRF defence is the central control and is tested adversarially (DNS rebinding,
  redirect-to-internal, IPv6-mapped IPv4, decimal/octal IP forms). Fetching runs from an egress path
  with no access to cluster-internal services; extraction never executes scripts.
- **Privacy & Compliance** — PII redaction before egress; AI disclosure through `aigateway`; COPPA
  gating for under-13 accounts; retention of prompts/completions per org policy; the ingested corpus is
  instructor-visible, and its provenance is recorded for the RoPA (S05) and DPIA (S06).
- **Accessibility** — Citations render as accessible footnote links with descriptive text (never
  "[1]" alone as the accessible name); pending/failed ingestion states are announced.
- **Scalability** — Extraction runs on the job queue with per-org concurrency caps; cache is shared
  across courses in an org, so a widely-assigned article is fetched once.
- **Reliability** — Link failures degrade gracefully: the pack is built from what is available and the
  answer states which sources were unavailable. Circuit breaker per host after repeated failures.
- **Observability** — `lextures_content_tool_context_build_seconds`,
  `…_link_fetch_total{outcome}`, `…_link_cache_hits_total`, `…_ai_calls_total{tool_id,outcome}`,
  `…_ai_budget_denials_total{level}`. Alerts on fetch failure rate > 25% and on budget denials spiking.
- **Maintainability** — One context service; tools receive a typed pack and never call a fetcher.
  Extraction version is recorded so a parser upgrade can invalidate cache deterministically.
- **Internationalization** — Extraction preserves the source language; the pack records `lang` per
  segment so a tool can instruct the model to answer in the learner's locale.
- **Backward compatibility** — Additive. `aiprovider` gains an optional tool-calling capability;
  providers lacking it keep working through the orchestrated fallback (FR-10).

## 7. Acceptance Criteria

- **AC-1.** *Given* a section containing two external links, *When* a context pack is built, *Then*
  both links appear as discovered sources and the section body is the first segment.
- **AC-2.** *Given* a link to `http://169.254.169.254/…`, `http://localhost/…`, or a public URL that
  redirects to a private range, *When* ingestion runs, *Then* every case is refused, logged, and shown
  to the instructor as "blocked: private network".
- **AC-3.** *Given* a cached link with an unchanged `ETag`, *When* revalidation runs, *Then* the
  provider returns 304 and no re-extraction occurs.
- **AC-4.** *Given* a provider with tool-calling, *When* the model calls `fetch_link`, *Then* passages
  are returned and the final answer carries citations for the links actually used.
- **AC-5.** *Given* a provider without tool-calling, *When* the same question is asked, *Then* the
  orchestrated fallback injects the top-ranked passages and the answer carries the same citation shape.
- **AC-6.** *Given* `aigateway` denies the feature for a COPPA-flagged user, *When* a tool action runs,
  *Then* no provider call is made and the tool shows an explanatory, non-alarming message.
- **AC-7.** *Given* a course at its monthly AI budget, *When* a student asks a question, *Then* the
  response is a typed budget error naming the limit, the instructor is notified once per day, and no
  provider call occurs.
- **AC-8.** *Given* a student's message containing an email address and a phone number, *When* it is
  sent to the provider, *Then* both are redacted in the payload and in the stored log.
- **AC-9.** *Given* a link that returns a 500 for three consecutive attempts, *When* a fourth build
  occurs, *Then* the host breaker is open, no fetch is attempted, and the pack marks that source
  unavailable.
- **AC-10.** *Given* ACE is serving a rewritten variant of the page, *When* a pack is built, *Then* it
  contains the variant text the learner was served, not the base text.
- **AC-11.** *Given* an instructor excludes a link, *When* a pack is next built, *Then* that link is
  absent and no fetch is attempted.
- **AC-12.** *Given* a 30 MB PDF link, *When* ingestion runs, *Then* it is refused at the size cap with
  a clear instructor-visible reason.

## 8. Data Model

Migration `server/migrations/454_content_tool_context.sql` (+ `.down.sql`).

```sql
-- 454_content_tool_context.sql

-- Org-scoped cache of ingested external sources (shared across courses in the org).
CREATE TABLE IF NOT EXISTS course.content_tool_link_sources (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id            UUID REFERENCES tenant.organizations (id) ON DELETE CASCADE,
    url_hash          TEXT NOT NULL,          -- sha256 of the normalized URL
    url               TEXT NOT NULL,
    final_url         TEXT,                   -- after redirects
    content_type      TEXT,
    title             TEXT,
    lang              TEXT,
    extracted_text    TEXT,
    extraction_version INTEGER NOT NULL DEFAULT 1,
    byte_size         INTEGER,
    etag              TEXT,
    last_modified     TEXT,
    status            TEXT NOT NULL DEFAULT 'pending'
                        CHECK (status IN ('pending','ready','blocked','failed','unsupported')),
    error             TEXT,
    fetched_at        TIMESTAMPTZ,
    expires_at        TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (org_id, url_hash, extraction_version)
);
CREATE INDEX IF NOT EXISTS idx_ctls_expires ON course.content_tool_link_sources (expires_at);

-- Chunked passages for retrieval + citation.
CREATE TABLE IF NOT EXISTS course.content_tool_link_chunks (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id   UUID NOT NULL REFERENCES course.content_tool_link_sources (id) ON DELETE CASCADE,
    ordinal     INTEGER NOT NULL,
    text        TEXT NOT NULL,
    token_count INTEGER NOT NULL DEFAULT 0,
    UNIQUE (source_id, ordinal)
);

-- Which sources an activity uses, plus instructor overrides.
CREATE TABLE IF NOT EXISTS course.content_tool_activity_sources (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id         UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    structure_item_id UUID REFERENCES course.course_structure_items (id) ON DELETE CASCADE,
    source_id         UUID REFERENCES course.content_tool_link_sources (id) ON DELETE CASCADE,
    origin            TEXT NOT NULL CHECK (origin IN ('body_link','config_link','course_file')),
    course_file_id    UUID,
    excluded          BOOLEAN NOT NULL DEFAULT FALSE,
    excluded_by       UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (structure_item_id, source_id, course_file_id)
);
CREATE INDEX IF NOT EXISTS idx_ctas_item ON course.content_tool_activity_sources (structure_item_id, excluded);

-- Per-course AI budget for content tools (spend guardrail).
ALTER TABLE course.content_tool_settings
    ADD COLUMN IF NOT EXISTS monthly_ai_token_budget BIGINT NOT NULL DEFAULT 0,   -- 0 = org default
    ADD COLUMN IF NOT EXISTS daily_ai_calls_per_user INTEGER NOT NULL DEFAULT 50,
    ADD COLUMN IF NOT EXISTS link_ingestion_mode TEXT NOT NULL DEFAULT 'public'
        CHECK (link_ingestion_mode IN ('off','allowlist','public'));
```

**Backfill** — none. **Purge** — sources past `expires_at` are revalidated or evicted nightly.

## 9. API Surface

| Verb | Path | Auth scope |
|---|---|---|
| `GET` | `.../content-tools/context/sources?itemId=` | instructor |
| `POST` | `.../content-tools/context/sources/{source_id}/reingest` | instructor |
| `PATCH` | `.../content-tools/context/sources/{source_id}` (`{excluded}`) | instructor |
| `POST` | `.../content-tools/context/preview` | instructor (dry-run pack, token counts) |

Student-facing access is **only** through tool actions (`POST .../actions/{action}` from CT.3) — there
is no public "ask the model" endpoint, so every call is bound to an instance, a manifest and a budget.

```ts
type ContextSegment = {
  kind: 'section' | 'activity' | 'course' | 'file' | 'link' | 'note'
  id: string
  title: string
  url?: string
  lang?: string
  text: string
  tokens: number
}
type ContextPack = {
  instanceId: string
  segments: ContextSegment[]
  pendingSources: Array<{ url: string; status: 'pending' | 'blocked' | 'failed'; reason?: string }>
  totalTokens: number
  variantId?: string            // set when ACE served a rewritten variant
}
type Citation = { kind: 'section' | 'file' | 'link'; id: string; title: string; url?: string; loc?: string }
```

- **Rate limits** — re-ingest 10/min/user; context preview 20/min/user.
- **OpenAPI** — instructor routes documented; the model-facing `fetch_link` tool is documented in the
  developer guide, not the HTTP spec.

## 10. UI / UX

**Instructor — "Sources" panel** (new tab in the content-page editor sidebar, and a section in the
Content Tools insights view):

1. Table of discovered sources: title, host, origin (body link / tool config / course file), status
   chip (Ready · Pending · Blocked · Failed · Unsupported), fetched-at, size.
2. Row actions: **Re-ingest**, **Exclude**, **Open original**, **View extracted text** (drawer).
3. Header shows the total token budget of the pack and a warning when it exceeds the model window,
   naming what will be dropped first.
4. Course settings gain **Link ingestion**: Off / Allowlist / Public, plus AI budget fields.

**Student** — no dedicated UI; citations render inside the consuming tool (CT.10) as numbered
footnote links with the source title as accessible text, and a "Sources" disclosure listing them.

**States** — *Empty*: "No external sources found in this activity." *Pending*: animated chip plus
"Answers may not include this source yet." *Blocked*: plain-language reason ("This link points to a
private network address and cannot be used"). *Error*: retry action with last error text.

**Mobile / responsive** — sources table collapses to cards; extracted-text drawer is full-screen.

**Accessibility** — status conveyed by text plus icon (never colour alone); the extracted-text drawer
is a labelled dialog; citation links carry descriptive accessible names.

**Copy & i18n** — `contentTools.context.*`, with reason strings enumerated (not free-text errors).

## 11. AI / ML Considerations

- **Models** — the course's configured provider/model via `aiprovider`; no CT.6-specific model choice.
- **Capability** — `aiprovider` gains an optional `ToolCallingCompleter` interface
  (`Complete(ctx, msgs, tools, opts)`), implemented first for providers that support it; the resolver
  reports the capability so CT.6 selects tool-calling or the orchestrated fallback per call.
- **Prompts** — CT.6 owns only the *context envelope* (source blocks with stable ids and an
  instruction to cite by id); the *task* prompt belongs to the calling tool.
- **Eval metric** — citation faithfulness (does every claim map to a provided segment?) measured on a
  50-item golden set per release; groundedness regression gates prompt changes.
- **Fallback path** — provider error → one retry with backoff → typed `provider_unavailable` returned
  to the tool, which preserves state and offers retry.
- **PII redaction** — `aitutor.RedactPII` applied to learner text pre-egress; instructor content is not
  redacted (it is course material) but is never mixed with learner PII in logs.
- **Cost budget** — per-request token cap (default 8k context / 1k completion), per-user daily cap,
  per-course monthly cap; all recorded in `analytics.ai_usage_log`.

## 12. Integration Points

- **Internal** — new `server/internal/service/contenttools/context/`,
  `service/aigateway` (gate), `service/aiprovider` (capability), `service/notebookrag` (chunking
  utilities), `service/aitutor` (PII redaction), `service/adaptivecontent` (served variant),
  `service/filestorage` + `service/officepreview` (course-file text extraction),
  `internal/background` (ingestion jobs), `internal/ratelimit`.
- **Egress** — ingestion uses a dedicated HTTP client with the SSRF guard and its own proxy/egress
  identity; documented in `deploy/` and `iac/`.
- **Events** — ingestion outcomes appended to `content_tool_events` for instructor visibility.

## 13. Dependencies & Sequencing

- **Must ship after:** CT.1, CT.3 (actions are the only student entry point).
- **Must ship before:** CT.10, CT.20, and any tool declaring `ai` in its manifest.
- **Shared infra needed:** job queue, egress path, `aigateway`, AI usage logging.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| SSRF via a pasted link | M | H | Layered guard (scheme, DNS pin, range deny, redirect re-check, size/time caps), adversarial test suite, isolated egress |
| Copyright/ToS concerns from caching third-party text | M | M | Cache only what the author linked, honour `robots.txt`, store extraction not verbatim rendering, org-level off switch, documented retention |
| Hallucinated citations | M | H | Citations restricted to provided segment ids; post-validation drops citations that do not resolve; faithfulness eval gates releases |
| Runaway AI spend in a large course | M | H | Three-level budgets with typed denials, instructor notification, admin dashboards (CT.7) |
| Slow third-party sites block students | M | M | Async ingestion, never inline-blocking; pack degrades with a stated gap |
| Prompt injection from an ingested page ("ignore previous instructions") | H | M | Source text is delimited and labelled untrusted; system prompt states that source content is data, not instruction; injection-canary evals in the golden set |
| Stale cached sources contradict an updated page | M | L | TTL + conditional revalidation + instructor re-ingest |

## 15. Rollout Plan

- **Feature flag** — `content_tool_settings.link_ingestion_mode` per course (default `public`, org can
  force `off`), plus ops kill-switch `CONTENT_TOOLS_LINK_INGEST_KILL_SWITCH`.
- **Sequencing** — migration `454_*` → SSRF-guarded fetcher + extraction + cache → context builder →
  `aiprovider` tool-calling capability + fallback → instructor Sources panel → enable for the pilot
  course with CT.10.
- **Dogfood** — pilot course with 5 heavily-linked pages; instructors validate extraction quality.
- **GA criteria** — SSRF suite green, faithfulness eval ≥ target, fetch success ≥ 90% on the dogfood
  corpus, zero budget incidents.
- **Rollback** — set ingestion `off` (context degrades to on-platform sources only, tools still work);
  kill-switch for total stop.

## 16. Test Plan

- **Unit** — URL normalization and hashing; SSRF guard truth table (private v4/v6, decimal/octal/hex
  forms, IPv6-mapped, DNS rebinding simulation, redirect chains); extraction on fixture pages; chunking
  and token counting; budget arithmetic; citation resolution.
- **Integration** — ingest → cache → revalidate (200/304) → re-ingest; pack assembly ordering and
  truncation under a token budget; ACE variant selection; `aigateway` denial paths; usage-log writes.
- **End-to-end** — Playwright: instructor sees discovered sources, excludes one, and a CT.10 answer
  no longer cites it; blocked-link messaging.
- **Security** — the adversarial SSRF suite as a CI gate; prompt-injection canaries; PII-redaction
  assertions on the exact provider payload; verification that no student route reaches a provider
  outside a tool action.
- **Accessibility** — axe on the Sources panel; citation link naming with a screen reader.
- **Performance / load** — 200 concurrent pack builds with warm cache p95 ≤ 150 ms; ingestion
  throughput and per-org concurrency caps under load.
- **Manual exploratory** — paywalled pages, JS-only pages, huge PDFs, non-English sources, redirect
  loops, sites blocking our agent.

## 17. Documentation & Training

- **Instructor** — "How the AI knows about your page": what is ingested, how to check and control it,
  why some links cannot be used.
- **Admin** — link-ingestion policy, budgets, retention of ingested text, what leaves the network.
- **Developer** — building an AI tool on the context pack; the citation contract; injection-safe
  prompt patterns; the `fetch_link` tool schema.
- **API reference** — instructor context routes.
- **Runbook** — kill-switch, cache eviction, host breaker reset, budget raise procedure.

## 18. Open Questions

1. Should link ingestion be org-opt-in rather than opt-out for K-12 tenants? Proposed: opt-out
   globally, but default `allowlist` for orgs with COPPA-flagged populations — confirm with CT.8.
2. Do we store extracted text indefinitely for reproducibility of a graded interaction, or expire it?
   Proposed: expire on TTL, but pin a snapshot when a source is cited in a *graded* interaction.
3. Should embeddings back retrieval, or is lexical ranking enough at activity scope (typically < 20
   sources)? Proposed: lexical + recency first; add embeddings only if evals demand it.
4. Which provider gets tool-calling first, and is the fallback ever *preferable* (cheaper, more
   predictable)? Proposed: measure both in dogfood and pick per-provider defaults.

## 19. References

- Existing files this work touches: `server/internal/service/aiprovider/*`,
  `server/internal/service/aigateway/service.go`, `server/internal/service/aitutor/aitutor.go`,
  `server/internal/service/notebookrag/`, `server/internal/service/adaptivecontent/`,
  `server/migrations/454_content_tool_context.sql`.
- External standards: RFC 9309 (robots.txt), RFC 7232 (conditional requests), RFC 3986,
  OWASP SSRF Prevention Cheat Sheet, OWASP LLM Top 10 (LLM01 prompt injection).
- Related plans: [CT.3](CT.3-student-runtime-and-state-persistence.md),
  [CT.8](../../plan/content_tools/CT.8-governance-safety-privacy-accessibility.md),
  [CT.10](../../plan/content_tools/CT.10-tool-ask-questions.md),
  [AC.3 content generation](../adaptive/AC.3-content-generation-engine.md),
  [S06 DPIA](../../plan/standards/S06-dpia-pia-algorithmic-impact.md),
  [S13 EU AI Act](../../plan/standards/S13-eu-ai-act-high-risk.md).
