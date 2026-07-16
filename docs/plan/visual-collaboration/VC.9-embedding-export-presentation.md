# VC.9 — Embedding, Export & Presentation

> Implementation plan. Source: net-new capability (real-time visual collaboration boards). Landscape: [visual-collaboration/README](README.md). Reuses the TipTap editor embed pattern (`clients/web/src/components/editor/extensions/whiteboard-node-view.tsx`).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | VC.9 |
| **Section** | Visual Collaboration Boards |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | MISSING |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Collaboration squad |
| **Depends on** | VC.2, VC.3 |
| **Unblocks** | — |

---

## 1. Problem Statement

A board's value multiplies when it can leave its page: **embedded** inside a lesson or page, **presented**
full-screen for the class, **exported** to PDF/CSV/image for grading or archiving, and **joined instantly**
via a QR code on the projector. Lextures already embeds Whiteboards inside its rich-text editor via a TipTap
node view; VC.9 gives boards the same reach plus a present mode and exports.

## 2. Goals

- **Embed** a board inside course content/pages via a TipTap node view (like the whiteboard node), rendering
  live (for members) or read-only.
- **Present mode**: a distraction-free, full-screen view that steps through cards (slideshow) or shows the
  whole board zoomed for projection.
- **Export**: board → PDF (print-ready), CSV (tabular card data), and image/PNG (snapshot of the surface).
- **Quick-join**: a QR code / short link for a board's share URL so students join from a projected code.
- Keep all of the above bound to the VC.6 access model (embeds/exports never bypass permissions).

## 3. Non-Goals

- The share-link/token mechanics themselves (VC.6 owns tokens/passwords/capabilities; VC.9 renders and
  QR-encodes them).
- Realtime transport (VC.4); embeds reuse it when the viewer is a member.
- Video export / recording of a session (future).
- Public SEO/discoverability of boards (explicitly avoided for privacy).

## 4. Personas & User Stories

- **As an instructor**, I want to embed a live board inside a lesson page so students work without leaving
  the content.
- **As an instructor**, I want a present mode to show the class one card at a time on the projector.
- **As an instructor**, I want to export a board to PDF to attach to my records.
- **As an instructor**, I want to export card data to CSV to analyze responses.
- **As an instructor**, I want a QR code on screen so students join the board instantly from their phones.
- **As a student**, I want an embedded board to behave like the full board (post/react) when I'm a member.

## 5. Functional Requirements

- **FR-1.** The rich-text editor MUST support a **board embed** node (TipTap) that references a board by id;
  the editor slash-menu MUST offer "Insert board", mirroring the whiteboard node integration.
- **FR-2.** A rendered embed MUST enforce VC.6 access: members see the live/interactive board (subject to
  contributor policy); non-members/anonymous see a read-only render only if the board's visibility/link
  permits, else an access-denied placeholder.
- **FR-3.** **Present mode** MUST provide (a) *slideshow* — one card at a time with next/prev/keyboard/auto-
  advance, and (b) *overview* — the whole board zoomed to fit; both hide editing chrome.
- **FR-4.** **PDF export** MUST render the board's cards in reading order (respecting layout: sections in
  order, canvas/timeline/map by a sensible traversal) to a print-ready document, generated server-side for
  fidelity and large boards.
- **FR-5.** **CSV export** MUST emit one row per card with columns: section, author (respecting attribution —
  omit/redact when anonymous), content type, text/body, link, attachment filename, reaction count/avg,
  comment count, created-at.
- **FR-6.** **Image export** MUST produce a PNG snapshot of the current surface (client canvas render is
  acceptable; server render optional for headless).
- **FR-7.** Exports MUST require `item:create` (or board manage) and MUST respect anonymity and hidden/
  removed/pending content (excluded from exports unless a manager explicitly includes moderation data).
- **FR-8.** **Quick-join** MUST render a QR code encoding the board's access URL (a VC.6 share link when
  external, else the in-app board URL); the QR/short link MUST honour link expiry/revoke.
- **FR-9.** Embeds and present mode MUST be responsive and keyboard-navigable.
- **FR-10.** All export/print output MUST be accessible where the format allows (tagged PDF, alt text
  included; CSV is inherently text).

## 6. Non-Functional Requirements

- **Performance** — PDF/CSV export of a 200-card board completes < 10 s (server job with progress for large
  boards); present-mode navigation is instant.
- **Security** — embeds/exports run through the VC.6 resolver; no export bypasses anonymity or hidden-content
  rules; QR encodes only a permitted URL; server-side PDF render uses an SSRF-safe, sandboxed renderer.
- **Privacy & Compliance** — exports of student work are education records; author redaction under anonymity
  is mandatory; exported files inherit retention rules; align with FERPA and [S02](../standards/S02-data-retention-deletion-engine.md).
- **Accessibility** — present mode is keyboard-operable with visible focus and reduced-motion for auto-
  advance; PDFs are tagged; embeds preserve card semantics.
- **Scalability** — heavy exports offloaded to the job queue; QR generated on demand (cheap).
- **Reliability** — export failures are retryable; partial files never delivered.
- **Observability** — counters for embeds rendered, present sessions, exports by type, QR generations.
- **Maintainability** — one board serializer feeds PDF/CSV/image so formats stay consistent.
- **Internationalization** — exported headers/labels and present-mode chrome localised; RTL-safe.
- **Backward compatibility** — additive; the editor gains one node type.

## 7. Acceptance Criteria

- **AC-1.** *Given* an instructor editing a page, *when* they insert a board embed, *then* the page renders
  the board; a member can interact and a non-member sees read-only or access-denied per VC.6.
- **AC-2.** *Given* present mode slideshow, *when* the instructor presses →, *then* it advances one card;
  keyboard and on-screen controls work; overview shows the full board zoomed to fit.
- **AC-3.** *Given* a board with sections, *when* exported to PDF, *then* cards appear grouped by section in
  order, print-ready, with image alt text included.
- **AC-4.** *Given* an anonymous-attribution board, *when* exported to CSV, *then* author columns are
  redacted/omitted and no author is recoverable.
- **AC-5.** *Given* hidden/pending/removed cards, *when* a standard export runs, *then* they are excluded.
- **AC-6.** *Given* quick-join, *when* the QR is scanned, *then* it opens the correct access URL; if the
  underlying link is revoked, the URL denies access.
- **AC-7.** *Given* image export, *when* triggered, *then* a PNG snapshot of the current surface downloads.

## 8. Data Model

Migration `386_board_exports.sql` (minimal — most work is stateless):

```sql
CREATE TABLE board.export_jobs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    board_id    UUID NOT NULL REFERENCES board.boards (id) ON DELETE CASCADE,
    format      TEXT NOT NULL,               -- pdf|csv|image
    status      TEXT NOT NULL DEFAULT 'pending', -- pending|running|done|failed
    storage_key TEXT,                         -- object-store key of the produced file
    requested_by UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);
CREATE INDEX idx_export_jobs_board ON board.export_jobs (board_id, created_at);
```

- Board **embeds** need no table: the TipTap node stores `{ boardId }` in the document JSON.
- QR/short link reuse VC.6 `board_shares`; no new storage.

## 9. API Surface

| Verb | Path | Auth |
|---|---|---|
| POST | `/boards/{id}/export` — `{format}` → `202` + job id | `item:create` |
| GET | `/boards/{id}/export/{jobId}` (status + download url) | `item:create` |
| GET | `/boards/{id}/qr` (PNG/SVG of the access URL) | member with view; encodes VC.6 URL |
| GET | `/boards/{id}/embed` (read-only render context for non-members via permitted link) | resolver (VC.6) |

- CSV may also stream synchronously for small boards; PDF/image go through the job.
- **OpenAPI**: export + qr endpoints.

## 10. UI / UX

- **Editor node** (`components/editor/extensions/board-node-view.tsx` + slash command): "Insert board" picks
  a board from the course; renders inline with a header + surface; read-only fallback + access-denied state.
- **Present mode** (`components/boards/present-mode.tsx`): full-screen; slideshow controls (prev/next/auto),
  overview toggle, exit; large type; hides editing chrome.
- **Export menu**: PDF / CSV / Image from the board menu; progress + download when ready; error/retry.
- **Quick-join**: a "Share to class" panel shows a large QR + short URL for projection.
- **States**: export-in-progress, export-failed, embed access-denied, present empty board.
- **Mobile**: present mode adapts to portrait; QR panel scales.
- **Accessibility**: present mode keyboard nav + reduced motion; tagged PDF; QR panel has the URL as text
  too.
- **Copy & i18n**: `boards.embed.*`, `boards.present.*`, `boards.export.*` keys.

## 11. AI / ML Considerations

Not AI-touching. (Optional future: an AI "summary slide" appended to present/export — deferred.)

## 12. Integration Points

- **Reuse**: TipTap editor extension pattern (`whiteboard-node-view.tsx`, `whiteboard-tip-tap.ts`,
  slash-menu registration in `block-editor/markdown-body-slash.ts`), `filestorage` for export files, job
  queue for heavy exports, VC.6 resolver + `board_shares` for QR, VC.4 for live embeds.
- **New**: `server/internal/repos/board/exports.go`, `server/internal/service/boardexport/` (PDF/CSV/image
  renderers), `server/internal/httpserver/board_export_http.go`,
  `clients/web/src/components/editor/extensions/board-node-view.tsx`,
  `clients/web/src/components/boards/present-mode.tsx`.
- **Server PDF**: use an approved headless renderer already vetted for CSP/sandbox; if none exists,
  server-side HTML→PDF via a sandboxed converter (documented dependency).

## 13. Dependencies & Sequencing

- Must ship after: VC.2 (content), VC.3 (layout/order for export/present). QR depends on VC.6 for external
  links (in-app URL works without it). Live embeds enhanced by VC.4.
- Must ship before: nothing hard.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Server PDF renderer = SSRF/RCE surface | M | H | Sandboxed, network-restricted renderer; render from serialized data, not arbitrary URLs |
| Export leaks anonymous author / hidden content | M | H | Single serializer applies VC.6/VC.7 rules; explicit redaction tests |
| Large-board export times out | M | M | Job queue + progress; paginate PDF |
| Embedded board bypasses permissions | L | H | Embed always routes through the resolver; tests for non-member render |

## 15. Rollout Plan

- **Flag**: gated by `visual_boards_enabled`; embed node appears in the editor only when the course flag is
  on.
- **Sequencing**: migration `386` → ship CSV + image + present mode → add PDF (server renderer) → add embed
  node → add QR (after VC.6 external links).
- **Rollback**: hide export/embed/present entrypoints; boards still fully usable.

## 16. Test Plan

- **Unit** — serializer→CSV columns; anonymity/hidden redaction; export-job state machine.
- **Integration** — PDF/CSV/image generation; embed resolver (member vs non-member); QR encodes correct URL;
  revoked link denies.
- **End-to-end** — Playwright: insert embed in a page; present slideshow + overview; export each format;
  scan-equivalent QR open.
- **Security** — PDF renderer sandbox; no author/hidden leakage in any export; embed permission checks.
- **Accessibility** — present mode keyboard + reduced motion; tagged PDF; QR text alternative.
- **Performance** — 200-card PDF/CSV timing; job progress.
- **Manual** — projector present mode; mobile QR join.

## 17. Documentation & Training

- End-user: embedding a board in a page; presenting; exporting; sharing via QR.
- Instructor: what exports include and how anonymity is handled.
- API reference: export + qr endpoints; editor node.

## 18. Open Questions

1. Do we need server-side image render (headless) or is client canvas capture sufficient for GA?
   (Recommendation: client capture for GA; server render if fidelity/headless needed.)
2. Should present mode auto-advance be a per-board setting or session-only? (Recommendation: session-only
   control.)
3. Include moderation data (hidden/pending) in a special "instructor export"? (Recommendation: yes, gated to
   managers, clearly labelled.)

## 19. References

- Existing files: `clients/web/src/components/editor/extensions/whiteboard-node-view.tsx`,
  `clients/web/src/components/editor/extensions/whiteboard-tip-tap.ts`,
  `clients/web/src/components/editor/block-editor/markdown-body-slash.ts`, `filestorage`, job queue.
- Related plans: [VC.2](../../completed/visual-collaboration/VC.2-posts-and-content-types.md), [VC.3](../../completed/visual-collaboration/VC.3-board-layouts-and-arrangement.md),
  [VC.4](VC.4-realtime-collaboration-and-presence.md), [VC.6](VC.6-sharing-access-contributors.md),
  [VC.7](VC.7-moderation-safety-governance.md).
