# VC.2 — Posts & Multi-Format Content

> Implementation plan. Source: net-new capability (real-time visual collaboration boards). Landscape: [visual-collaboration/README](../../plan/visual-collaboration/README.md). Reuses the file-upload stack (`server/internal/service/filestorage`, TUS, MinIO/S3) and the AV-scan flag.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | VC.2 |
| **Section** | Visual Collaboration Boards |
| **Severity** | BLOCKER |
| **Markets** | K12 / HE / SL |
| **Status (today)** | DONE |
| **Estimated effort** | L (1–2mo) |
| **Owner (proposed)** | Collaboration squad |
| **Depends on** | VC.1 |
| **Unblocks** | VC.3, VC.4, VC.5, VC.7, VC.9 |

---

## 1. Problem Statement

A board is only useful once people can put things on it. The defining capability of the incumbent tools we
are replacing is that a single card can be **any kind of content** — a sticky note, a photo, a document, a
YouTube/Vimeo link that renders inline, an audio clip, a quick sketch — and anyone with access can add one
in seconds. VC.2 delivers the **post (card) model** and the **content-type composer** so every learner can
contribute rich media to a board.

## 2. Goals

- Model a **post** (card) belonging to a board, authored by a user, carrying one primary content block plus
  an optional attachment.
- Support content types: **text/rich-text, image, file/document, link (with preview), video (embed or
  upload), audio (upload or record), and drawing/sketch**.
- Reuse the existing upload pipeline (presigned/TUS uploads, MinIO/S3, AV scanning, storage quota) for all
  file-backed content.
- Generate link previews (title, description, thumbnail) server-side for URL posts.
- Provide a fast composer with drag-and-drop and paste-to-upload.

## 3. Non-Goals

- Card **placement/layout** on the board surface (VC.3 owns position/section/order).
- **Real-time** propagation of new posts (VC.4) — VC.2 works with optimistic add + refetch until VC.4 lands.
- Reactions/comments on posts (VC.5).
- Moderation/approval before a post appears (VC.7) — VC.2 posts appear immediately for authorized authors.
- Screen recording capture (future; note in Open Questions).

## 4. Personas & User Stories

- **As a student**, I want to add a text note to a board so I can share an idea quickly.
- **As a student**, I want to upload a photo of my handwritten work so my group can see it.
- **As a student**, I want to paste a YouTube link and have it show a playable thumbnail.
- **As an instructor**, I want to attach a PDF to a card so the class has the reading in context.
- **As a student**, I want to record a short audio reflection directly in the browser.
- **As a self-learner**, I want to sketch a quick diagram on a card without leaving the board.

## 5. Functional Requirements

- **FR-1.** The system MUST persist a `board.posts` row with `board_id`, `author_id`, `content_type`, a
  `title` (optional), a rich-text `body` (optional), a `link_url` (optional), an `attachment_id`
  (optional), and layout placeholders (`position`, `section_id`, `sort_index`) owned by VC.3.
- **FR-2.** The system MUST support `content_type ∈ {text, image, file, link, video, audio, drawing}`.
- **FR-3.** `POST /api/v1/courses/{code}/boards/{board_id}/posts` MUST create a post; the request MUST be
  rejected (`400`) if the content for the declared `content_type` is missing (e.g. `link` without
  `link_url`).
- **FR-4.** File-backed posts (`image`, `file`, `video`, `audio`) MUST upload through the existing
  `filestorage` service (presigned or TUS), and the stored object MUST be associated to the post only after
  a successful upload; orphaned uploads MUST be reaped by the existing cleanup job.
- **FR-5.** Uploaded attachments MUST pass the AV-scan pipeline when `AvScanningEnabled` is on; a post whose
  attachment is quarantined MUST NOT render its file and MUST show a "scanning/blocked" state.
- **FR-6.** For `link` posts, the system MUST fetch and cache an unfurl (title, description, image, site
  name) via a server-side fetch with SSRF protections; failures MUST degrade to a bare link.
- **FR-7.** For known video providers (YouTube, Vimeo, and direct-uploaded video), the client MUST render an
  inline player; for other links it renders the unfurl card.
- **FR-8.** `drawing` posts MUST store a sketch payload reusing the Whiteboard canvas serialization
  (`clients/web/src/lib/whiteboard/serialize.ts` / `types.ts`), so a card can hold a small drawing.
- **FR-9.** The system MUST provide `GET .../posts` (list for a board), `GET .../posts/{post_id}`,
  `PATCH .../posts/{post_id}` (edit own post; instructors may edit any), and `DELETE .../posts/{post_id}`.
- **FR-10.** Authors MUST be able to edit and delete their own posts; users with
  `course:{code}:item:create` MUST be able to edit/delete any post on boards in their course.
- **FR-11.** The composer MUST accept drag-and-drop files and clipboard-pasted images, enforcing per-file
  size and MIME allow-lists consistent with the course files feature.
- **FR-12.** Text/rich-text bodies MUST be sanitized on write (no raw HTML injection) and support a safe
  subset (bold, italic, lists, links, code) consistent with the existing rich-text editor.

## 6. Non-Functional Requirements

- **Performance** — post list p95 < 250 ms for 200 posts (thumbnails via CDN/presigned GET); composer add
  feels instant via optimistic insert.
- **Security** — attachments are course-scoped; presigned URLs are short-lived; link unfurl runs through an
  SSRF-safe fetcher (block private IP ranges, cap redirects, cap body size); rich text sanitized.
- **Privacy & Compliance** — attachments and post bodies are education records; deletion/export must reach
  the object store (align with [S02 retention](../../plan/standards/S02-data-retention-deletion-engine.md)).
- **Accessibility** — every media card has alt-text (image `alt` required or prompted), captions surfaced
  for video where present, and audio has a transcript affordance; composer is keyboard-operable.
- **Scalability** — posts partition naturally by board; attachment bytes live in object storage, not the DB.
- **Reliability** — create is transactional (post row + attachment link); upload failures leave no dangling
  post.
- **Observability** — counters per content type (`board.post.created{type}`), upload success/failure, unfurl
  latency/failure.
- **Maintainability** — attachment handling reuses `filestorage`/`coursefiles`; no new storage client.
- **Internationalization** — composer copy and content-type labels externalised; unfurl respects the
  target's language where available.
- **Backward compatibility** — additive tables; no change to existing posts (none exist yet).

## 7. Acceptance Criteria

- **AC-1.** *Given* an authorized member, *when* they submit a text card, *then* it persists and appears in
  the board's post list.
- **AC-2.** *Given* an image drag-dropped into the composer, *when* upload completes, *then* a card renders
  the image thumbnail and stores the attachment id.
- **AC-3.** *Given* a pasted YouTube URL, *when* the post is created, *then* the card shows the video title
  and thumbnail and plays inline on click.
- **AC-4.** *Given* an arbitrary article URL, *when* the post is created, *then* the card shows the unfurled
  title/description/image, and if unfurl fails it shows a plain clickable link.
- **AC-5.** *Given* AV scanning is on and an uploaded file is quarantined, *then* the card shows a blocked
  state and never serves the file.
- **AC-6.** *Given* a student authored a post, *when* another student tries to edit it, *then* the API
  returns `403`; *when* the instructor edits it, *then* it succeeds.
- **AC-7.** *Given* a `link` content_type with no `link_url`, *when* create is called, *then* it returns
  `400`.
- **AC-8.** *Given* a drawing card, *when* saved and reloaded, *then* the sketch re-renders from the stored
  payload.

## 8. Data Model

Migration `379_board_posts.sql`:

```sql
CREATE TABLE board.posts (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    board_id      UUID NOT NULL REFERENCES board.boards (id) ON DELETE CASCADE,
    author_id     UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    content_type  TEXT NOT NULL,               -- text|image|file|link|video|audio|drawing
    title         TEXT NOT NULL DEFAULT '',
    body          JSONB,                        -- sanitized rich-text doc (nullable)
    link_url      TEXT,
    link_preview  JSONB,                        -- {title, description, image, siteName, fetchedAt}
    drawing_data  JSONB,                        -- Whiteboard serialization for drawing cards
    attachment_id UUID REFERENCES board.post_attachments (id) ON DELETE SET NULL,
    -- layout columns (owned/used by VC.3)
    section_id    UUID,
    sort_index    DOUBLE PRECISION NOT NULL DEFAULT 0,
    position      JSONB,                        -- {x, y, w, h} for freeform layouts
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_posts_board ON board.posts (board_id);

CREATE TABLE board.post_attachments (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    board_id      UUID NOT NULL REFERENCES board.boards (id) ON DELETE CASCADE,
    storage_key   TEXT NOT NULL,               -- object-store key (filestorage)
    file_name     TEXT NOT NULL,
    mime_type     TEXT NOT NULL,
    size_bytes    BIGINT NOT NULL,
    alt_text      TEXT NOT NULL DEFAULT '',
    scan_status   TEXT NOT NULL DEFAULT 'pending', -- pending|clean|blocked (AV pipeline)
    uploaded_by   UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_post_attachments_board ON board.post_attachments (board_id);
```

- The `attachment_id` FK is added after `post_attachments` exists (declare tables in dependency order or use
  a deferred constraint).
- **Backfill**: none.

## 9. API Surface

| Verb | Path | Auth |
|---|---|---|
| GET | `/boards/{board_id}/posts` | course access |
| POST | `/boards/{board_id}/posts` | post-permitted member (see VC.6) |
| GET | `/boards/{board_id}/posts/{post_id}` | course access |
| PATCH | `/boards/{board_id}/posts/{post_id}` | author or `item:create` |
| DELETE | `/boards/{board_id}/posts/{post_id}` | author or `item:create` |
| POST | `/boards/{board_id}/attachments` | post-permitted member (presign/TUS init) |
| POST | `/boards/{board_id}/link-preview` | post-permitted member (unfurl a URL) |

```ts
type Post = {
  id: string; boardId: string; authorId: string | null
  contentType: 'text'|'image'|'file'|'link'|'video'|'audio'|'drawing'
  title: string
  body?: RichTextDoc
  linkUrl?: string
  linkPreview?: { title?: string; description?: string; image?: string; siteName?: string }
  drawingData?: unknown
  attachment?: { id: string; url: string; fileName: string; mimeType: string; sizeBytes: number; altText: string; scanStatus: 'pending'|'clean'|'blocked' }
  sectionId?: string; sortIndex: number; position?: { x: number; y: number; w: number; h: number }
  createdAt: string; updatedAt: string
}
```

- **Rate limits**: per-user post-create limiter to blunt spam/flooding; unfurl endpoint tightly limited.
- **OpenAPI**: register post + attachment + unfurl schemas.

## 10. UI / UX

- **Composer** (`clients/web/src/components/boards/post-composer.tsx`): a "+" button opens a compact
  composer with a content-type switcher (text / image / link / file / video / audio / draw). Drag-and-drop
  and paste route to upload. Recording uses `MediaRecorder` for audio.
- **Post card** (`components/boards/post-card.tsx`): renders by content type — rich text, `<img>` with alt,
  file chip with download, link-preview card, inline video player, audio player with transcript link, or a
  read-only sketch canvas.
- **States**: uploading (progress), scanning (spinner + "checking file"), blocked (warning), unfurl-pending
  (skeleton), error (retry).
- **Mobile**: composer is a bottom sheet; cards are full-width; camera/mic capture supported.
- **Accessibility**: image posts require/prompt alt text; player controls are native/labelled; composer
  fully keyboard-operable; focus returns to the new card after add.
- **Copy & i18n**: `boards.compose.*`, `boards.post.*` keys.

## 11. AI / ML Considerations

Optional (behind flag, not required for GA): auto-suggest `alt_text` for images and auto-transcribe audio to
seed a transcript, reusing the existing captioning/transcription path (`AutoCaptioningEnabled`). Cost budget
and PII redaction follow the AI provider standards; fallback is manual entry. Out of scope if the flag is
off.

## 12. Integration Points

- **Reuse**: `server/internal/service/filestorage`, `server/internal/httpserver/tus.go`,
  `server/internal/httpserver/course_file_upload.go`, `server/internal/repos/coursefiles`, AV-scan flag,
  `clients/web/src/lib/whiteboard/serialize.ts` (drawing cards).
- **New**: `server/internal/repos/board/posts.go`, `board/attachments.go`, `board/unfurl.go`
  (SSRF-safe fetcher), `server/internal/httpserver/board_posts_http.go`,
  `clients/web/src/components/boards/*`.
- **Events**: post lifecycle telemetry; VC.4 will subscribe to these to broadcast.

## 13. Dependencies & Sequencing

- Must ship after: VC.1.
- Must ship before: VC.3 (needs posts to arrange), VC.4 (needs posts to sync), VC.5, VC.7, VC.9.
- Shared infra: object storage, AV scan, job queue (orphan reaper, unfurl fetch).

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| SSRF via link unfurl | M | H | Dedicated fetcher: DNS-pin, block private/link-local ranges, cap redirects/size/time |
| Attachment upload orphans | M | L | Two-phase (upload → attach); reuse existing orphan reaper |
| Rich-text XSS | M | H | Server-side sanitize on write + safe render; reuse existing editor's schema |
| Media size blowing quota | M | M | Enforce per-file limits; integrate storage-quota checks (VC.10) |
| Provider embed policy changes | L | M | Use oEmbed/thumbnail URLs, degrade to link card |

## 15. Rollout Plan

- **Flag**: gated by VC.1's `visual_boards_enabled`; no separate flag. Video/audio recording can hide behind
  a sub-flag if browser support is a concern.
- **Sequencing**: migration `379` → deploy → dogfood text+image first → enable link/video/audio/draw.
- **Rollback**: disable the course flag; attachments remain in object storage for retention rules.

## 16. Test Plan

- **Unit** — content-type validation; unfurl parser; sanitizer; drawing (de)serialize round-trip.
- **Integration** — upload → attach → serve happy path; AV-blocked path; authz for edit/delete; unfurl
  SSRF guard rejects private IPs.
- **End-to-end** — Playwright: add each content type; paste image; paste YouTube link; record audio; edit &
  delete own post; instructor deletes a student post.
- **Security** — SSRF suite; XSS payloads in body/title; oversize/disallowed MIME rejected; presigned URL
  expiry.
- **Accessibility** — axe on composer + each card type; alt-text enforcement; keyboard-only add flow.
- **Performance** — 200-post board render; thumbnail lazy-loading.
- **Manual** — flaky-network upload; mobile camera/mic.

## 17. Documentation & Training

- End-user: "Add cards to a board" (all content types) help article.
- Instructor: managing/editing student cards; alt-text expectations.
- API reference: posts, attachments, link-preview endpoints.
- Runbook: unfurl fetcher config and SSRF allow/deny lists.

## 18. Open Questions

1. Do we support **screen recording** capture in v1, or link-to-video only? (Recommendation: defer.)
2. Should multiple attachments per card be allowed, or one primary + inline body media? (Recommendation:
   one primary attachment for v1; rich body may embed images.)
3. Max file size per content type — align with course files limits or set board-specific caps? (Defer to
   VC.10 quotas.)

## 19. References

- Existing files: `server/internal/service/filestorage/*`, `server/internal/httpserver/tus.go`,
  `server/internal/httpserver/course_file_upload.go`, `server/internal/repos/coursefiles/*`,
  `clients/web/src/lib/whiteboard/serialize.ts`, `clients/web/src/lib/whiteboard/types.ts`.
- Related plans: [VC.1](VC.1-foundation-and-feature-flag.md), [VC.3](../../plan/visual-collaboration/VC.3-board-layouts-and-arrangement.md),
  [VC.4](../../plan/visual-collaboration/VC.4-realtime-collaboration-and-presence.md), [VC.7](../../plan/visual-collaboration/VC.7-moderation-safety-governance.md),
  [VC.10](../../plan/visual-collaboration/VC.10-admin-analytics-quotas-lifecycle.md).
