# VC.M2 — Mobile Boards: Posts & Multi-Format Cards (View + Compose)

> Implementation plan. Source: mobile parity for board posts. Landscape: [visual-collaboration/README](../../plan/visual-collaboration/README.md). Mirrors web [VC.2](VC.2-posts-and-content-types.md); reuses the board REST post/attachment/unfurl endpoints and native capture (camera, photo library, mic).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | VC.M2 |
| **Section** | Visual Collaboration Boards — Mobile |
| **Severity** | BLOCKER |
| **Markets** | K12 / HE / SL |
| **Status (today)** | DONE |
| **Estimated effort** | L (1–2mo) |
| **Owner (proposed)** | Mobile squad |
| **Depends on** | VC.M1 |
| **Unblocks** | VC.M3, VC.M4, VC.M5, VC.M7 |

---

## 1. Problem Statement

An empty board shell (VC.M1) is not useful until learners can put things on it and read what others posted.
On mobile the defining moment is even sharper: a phone is a camera, a mic, and a keyboard in one — a student
should snap a photo of their handwriting, record a 20-second reflection, or paste a YouTube link in seconds.
VC.M2 renders every board card type and delivers a mobile composer so learners contribute rich media from
their phones.

## 2. Goals

- Render all VC.2 content types as mobile cards: **text/rich-text, image, file, link (with preview), video
  (embed or uploaded), audio, and drawing (read-only)**.
- Provide a mobile **composer** (bottom sheet) that creates at least: text, image (camera or library), link,
  file, and recorded audio; drawing capture is optional/deferred, drawing **display** is required.
- Reuse the board attachment upload endpoints for file-backed cards; surface upload progress and the AV-scan
  ("checking file" / "blocked") states honestly.
- Let authors edit/delete their own cards; staff (`item:create`) edit/delete any.
- Respect the board's posting capability (`canPost` from the VC.6 access resolver, surfaced in VC.M6) —
  read-only viewers get no composer.

## 3. Non-Goals

- Card placement/layout (VC.M3) — VC.M2 renders a simple recency-ordered list until layouts land.
- Real-time propagation (VC.M4) — VC.M2 uses optimistic insert + pull-to-refresh.
- Reactions/comments/grades (VC.M5).
- **Drawing capture** on mobile (create a sketch) — display only in v1; capture is an open question.
- Screen recording capture.

## 4. Personas & User Stories

- **As a student**, I want to snap a photo of my whiteboard work and post it as a card.
- **As a student**, I want to type a quick text note and post it.
- **As a student**, I want to paste a YouTube link and see a playable thumbnail on the card.
- **As a student**, I want to record a short audio reflection with my phone mic.
- **As an instructor**, I want to attach a PDF to a card and delete an off-topic student card.
- **As a student**, I want to read a classmate's drawing card even though I can't draw one on my phone yet.

## 5. Functional Requirements

- **FR-1.** The board detail surface MUST call `GET …/boards/{id}/posts` and render each post by
  `contentType ∈ {text, image, file, link, video, audio, drawing}` in a mobile card.
- **FR-2.** Text/rich-text bodies MUST render through the existing mobile rich-text/markdown renderer used by
  notebooks/syllabus; the client MUST NOT render raw HTML (bodies are server-sanitized, but the client renders
  the safe doc model only).
- **FR-3.** Image cards MUST show the thumbnail with its `altText`; tapping opens a full-screen viewer. File
  cards MUST show a file chip (name, size) with an open/download action.
- **FR-4.** Link cards MUST render the server unfurl (`linkPreview` title/description/image/siteName); when
  unfurl is absent, render a plain tappable link. YouTube/Vimeo/uploaded video MUST render an inline/native
  player; other links open the unfurl card.
- **FR-5.** Drawing cards MUST render the stored sketch **read-only** (rasterize/replay the Whiteboard
  serialization); create/edit of drawings is out of scope for v1.
- **FR-6.** The composer MUST create text, image (camera or photo library), link, file, and audio (mic
  recording) posts via `POST …/boards/{id}/posts` (+ the attachment upload flow for file-backed types),
  rejecting a declared type with missing content client-side before submit.
- **FR-7.** File-backed posts MUST upload via the board attachment endpoint (`POST …/boards/{id}/attachments`,
  presign/TUS) and only attach on success; the composer MUST show upload progress and handle failure/retry.
- **FR-8.** A post whose attachment `scanStatus == 'pending'` MUST show a "checking file" state; `'blocked'`
  MUST show a warning and never fetch the file bytes.
- **FR-9.** Image posts MUST prompt for (or allow adding) **alt text** before/after upload; empty alt text is
  discouraged with a hint, consistent with web VC.2 accessibility.
- **FR-10.** Authors MUST be able to edit (`PATCH …/posts/{id}`) and delete (`DELETE …/posts/{id}`) their own
  posts; users with `item:create` MUST be able to edit/delete any; unauthorized edit/delete controls MUST be
  hidden and the API-level `403` handled gracefully.
- **FR-11.** The composer MUST be hidden entirely when the resolved board capability `canPost` is false (see
  VC.M6); until VC.M6 lands, gate on the course create-permission the app already knows.

## 6. Non-Functional Requirements

- **Performance** — post list p95 render < 300 ms for 200 posts; images lazy-load thumbnails; audio/video
  stream, never fully buffered before play.
- **Security** — attachments fetched only via the server-issued presigned/authorized URLs; blocked files never
  requested; the client never bypasses the AV state.
- **Privacy & Compliance** — media and bodies are education records; downloaded media respects OS file
  protections; deletion propagates server-side (S02 retention).
- **Accessibility** — image alt text surfaced to VoiceOver/TalkBack; audio cards expose a transcript
  affordance where present; native player controls; composer fully operable with the on-screen keyboard and
  assistive tech; camera/mic permission prompts explained.
- **Scalability** — list virtualized; attachment bytes stay in object storage.
- **Reliability** — two-phase upload (upload → attach) so a failed upload leaves no dangling post; optimistic
  insert rolls back on error.
- **Observability** — reuse app error logging for upload failures; count composer opens by type if the app
  has lightweight analytics.
- **Internationalization** — composer/content-type labels externalised; RTL-correct card layouts.
- **Backward compatibility** — additive; unknown future content types render a safe "unsupported card, open on
  web" fallback rather than crashing.

## 7. Acceptance Criteria

- **AC-1.** *Given* a board with posts, *when* it opens, *then* each content type renders in its correct card
  form (text, image+alt, file chip, link unfurl, inline video, audio player, read-only drawing).
- **AC-2.** *Given* the composer, *when* a student takes a photo and adds alt text, *then* an image card
  appears after upload and stores the attachment id.
- **AC-3.** *Given* a pasted YouTube URL, *when* posted, *then* the card shows title + thumbnail and plays
  inline.
- **AC-4.** *Given* an uploaded file that is AV-quarantined, *then* the card shows a blocked state and the app
  never requests the file bytes.
- **AC-5.** *Given* a student's own post, *when* they edit/delete it, *then* it succeeds; *when* they try to
  edit another student's post, *then* no control is shown and the API `403` is handled.
- **AC-6.** *Given* an audio recording, *when* posted, *then* an audio card with playable controls appears.
- **AC-7.** *Given* an unknown/unsupported content type, *when* rendered, *then* a safe fallback card is shown
  (no crash).
- **AC-8.** *Given* both platforms, *when* CI builds run, *then* iOS build and Android compile are green.

## 8. Data Model

No server schema change — VC.2's `board.posts` / `board.post_attachments` already exist. Client models mirror
the web `Post` shape:

```kotlin
@Serializable data class BoardPost(
  val id: String, val boardId: String, val authorId: String? = null,
  val contentType: String,                       // text|image|file|link|video|audio|drawing
  val title: String = "",
  val body: JsonElement? = null,                 // sanitized rich-text doc
  val linkUrl: String? = null,
  val linkPreview: LinkPreview? = null,
  val drawingData: JsonElement? = null,
  val attachment: PostAttachment? = null,        // {id,url,fileName,mimeType,sizeBytes,altText,scanStatus}
  val sectionId: String? = null, val sortIndex: Double = 0.0, val position: PostPosition? = null,
  val eventDate: String? = null, val lat: Double? = null, val lng: Double? = null,
  val createdAt: String, val updatedAt: String,
)
```
(iOS: the equivalent `Codable` structs in `Core/LMS/LMSBoardModels.swift`.)

## 9. API Surface

No new endpoints. Mobile consumes web VC.2's routes:

| Verb | Path | Auth |
|---|---|---|
| GET | `/boards/{id}/posts` | course access |
| POST | `/boards/{id}/posts` | post-permitted member |
| GET | `/boards/{id}/posts/{postId}` | course access |
| PATCH | `/boards/{id}/posts/{postId}` | author or `item:create` |
| DELETE | `/boards/{id}/posts/{postId}` | author or `item:create` |
| POST | `/boards/{id}/attachments` | post-permitted member (presign/TUS init) |
| POST | `/boards/{id}/link-preview` | post-permitted member (unfurl a URL) |

## 10. UI / UX

- **Composer bottom sheet** (iOS `Features/Boards/BoardComposerView.swift`, Android
  `features/boards/BoardComposer.kt`): content-type switcher (text / photo / link / file / audio; draw shown
  disabled with "create on web" hint in v1), native pickers (camera, photo library, document picker), and a
  `MediaRecorder`/`AVAudioRecorder` mic capture with a level meter.
- **Post cards**: type-specific renderers under a shared `BoardPostCard` — rich text, image with alt, file
  chip, link-preview card, inline video, audio player + transcript link, read-only sketch.
- **States**: uploading (progress bar), scanning (spinner + "checking file"), blocked (warning), unfurl
  pending (skeleton), edit/delete overflow on own/managed cards.
- **Accessibility**: alt-text prompt on image; native player controls; focus returns to the new card after
  add; permission-explainer dialogs before camera/mic.
- **Copy & i18n**: `boards.compose.*`, `boards.post.*` keys in the mobile locale catalog.

## 11. AI / ML Considerations

Optional, mirrors web VC.2 (off by default): suggest image `altText` and auto-transcribe recorded audio via
the existing captioning/transcription path. Fallback is manual entry; out of scope if the flag is off.

## 12. Integration Points

- **Reuse**: mobile networking client; the notebook/syllabus rich-text renderer for bodies; native camera /
  photo / document / mic APIs; the web-side unfurl + AV pipeline (server-side, unchanged).
- **Depends on / verify**: a mobile **attachment upload** path. If the apps lack a reusable presigned/TUS
  uploader today, VC.M2 MUST build a small board-attachment uploader against `POST …/boards/{id}/attachments`
  (this is called out as Open Question 1 / a sub-task, not assumed).
- **New (iOS)**: `Core/LMS/LMSAPIBoardPosts.swift`, `Features/Boards/{BoardComposerView,BoardPostCard,
  BoardPostDetailView}.swift` → regenerate the Xcode project.
- **New (Android)**: `core/lms/BoardPostsApi.kt`, `features/boards/{BoardComposer,BoardPostCard,
  BoardPostDetail}.kt`.

## 13. Dependencies & Sequencing

- Must ship after: VC.M1.
- Must ship before: VC.M3 (arrange needs posts), VC.M4 (sync needs posts), VC.M5, VC.M7.
- Shared infra: object storage + AV scan (server, existing); native capture permissions.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| No reusable mobile uploader exists | M | H | Build a focused board-attachment uploader (presign/TUS) as an explicit sub-task; verify before committing to file cards |
| Rich-text/drawing model mismatch on native | M | M | Reuse the notebook renderer for bodies; rasterize drawings read-only rather than re-implement the canvas |
| Camera/mic permission friction | M | M | Explainer dialogs; graceful denial (hide capture options) |
| Large media on cellular | M | M | Warn on large uploads; lazy-load thumbnails; stream playback |
| Blocked-file bytes fetched by accident | L | H | Central guard: never build a media URL when `scanStatus != 'clean'` |

## 15. Rollout Plan

- **Flag**: gated by `visualBoardsEnabled`; audio recording can hide behind a small client capability check.
- **Sequencing**: ship view-all + text/link first → image (camera/library) → file → audio; drawing stays
  read-only.
- **Rollback**: hide the composer (view-only) via a client kill-switch if upload issues arise; content remains
  server-side.

## 16. Test Plan

- **Unit** — content-type → card mapping; missing-content validation; AV-state gating; alt-text handling.
- **Integration** — upload → attach → render happy path; AV-blocked path; author vs manager edit/delete authz.
- **End-to-end (device)** — add each supported type; paste YouTube; record audio; edit/delete own; instructor
  deletes a student card.
- **Security** — blocked file never fetched; edit/delete authz; presigned URL expiry.
- **Accessibility** — alt-text flow; player labels; keyboard/AT-only composer; camera/mic permission paths.
- **Manual** — flaky-network upload; low-storage device; unsupported-type fallback.

## 17. Documentation & Training

- End-user: "Add cards from your phone" (photo, link, file, audio).
- Instructor: managing student cards on mobile; alt-text expectations.
- Update the mobile READMEs' feature list.

## 18. Open Questions

1. Does a reusable mobile upload path exist, or must VC.M2 build the board-attachment uploader? (Assume build
   until verified in-repo.)
2. Do we attempt **drawing capture** on mobile v1, or display-only? (Recommendation: display-only; capture is a
   fast-follow.)
3. Inline video for uploaded (non-YouTube) files — native player vs. link-out on constrained devices?
   (Recommendation: native player with graceful link-out fallback.)

## 19. References

- Web plan: [VC.2](../../completed/visual-collaboration/VC.2-posts-and-content-types.md); web card renderer
  `clients/web/src/components/boards/post-card.tsx`, composer `post-composer.tsx`.
- Existing mobile: notebook rich-text renderer, `clients/mobile/locales/*.json`, `docs/MOBILE_PLAN.md`
  (upload caveats).
- Related mobile plans: [VC.M3](VC.M3-mobile-layouts-and-arrangement.md), [VC.M4](VC.M4-mobile-realtime-and-presence.md).
