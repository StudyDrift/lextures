# C33 — Accessibility, media & localization

> CLI parity plan. Source: `accessibility_http.go` (`accessibility`), `alt_text_http.go` (`alt-text`), `registerCaptionRoutes`/`registerCaptionAccessibilityRoutes`, `registerTTSRoutes` (`tts`), `stt`, `registerReadingLevelRoutes`, `registerTranscodeRoutes`, `registerTusRoutes`, `registerTranslationRoutes` (`translate`, `translation-memory`), `course_translation.go` (`courses/{id}/translations`, `translation-coverage`), `settings/locale`, `timezones`. Baseline: none.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | C33 |
| **Section** | Accessibility & localization |
| **Severity** | MINOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | MISSING |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Accessibility / CLI |
| **Depends on** | C02, C40 |
| **Unblocks** | — |

---

## 1. Problem Statement

Accessibility tooling (alt-text generation, captions, TTS/STT, reading-level), media transcoding, resumable (TUS) uploads, and course translation are UI-only. Content teams cannot batch-generate alt text/captions for a media library, transcode uploads, or manage per-course translations at scale — key for WCAG and multilingual compliance.

## 2. Goals

- Batch-generate and manage alt text, captions and translations across a course.
- Trigger and monitor media transcode jobs; use resumable TUS uploads for large media.
- Run accessibility checks and pull reading-level analysis.

## 3. Non-Goals

- Manual caption editing UX (browser).
- Building the ASR/MT models (server/provider).

## 4. Personas & User Stories

- **As a content team**, I want `alt-text generate --course C` to caption all images.
- **As an accessibility lead**, I want `captions generate --item I` and `captions upload --file .vtt`.
- **As a localizer**, I want `translations generate --course C --to es` + `translations coverage`.
- **As a media engineer**, I want `media transcode <item> --wait` and `media upload --tus bigfile.mp4`.

## 5. Functional Requirements

- **FR-1.** MUST add `alt-text generate|list|set` (`alt_text_http.go`; batch per course, `--wait`).
- **FR-2.** MUST add `captions generate|list|upload|delete` (caption routes; `.vtt`/`.srt`).
- **FR-3.** MUST add `translations generate|list|coverage|set <course> --to <locale>` (`course_translation.go`).
- **FR-4.** SHOULD add `accessibility check <course>` (`accessibility_http.go`) reporting WCAG issues.
- **FR-5.** SHOULD add `media transcode <item> --wait`, `media upload --tus <file>` (`registerTranscodeRoutes`, `registerTusRoutes`), `tts synth --file text --out audio`, `reading-level <item>`.

## 6. Non-Functional Requirements

- **Performance** — batch/transcode async with `--wait`; TUS uploads resumable/chunked.
- **Security** — content scope; provider keys server-side.
- **Privacy & Compliance** — WCAG 2.1 AA is the target these features serve; alt-text/caption generation may send media to AI providers (disclosure via C29).
- **Reliability** — generation idempotent; skip already-captioned items with `--skip-existing`.
- **Internationalization** — locale codes validated against `settings/locale`.
- **Backward compatibility** — additive; complements existing `files upload`.

## 7. Acceptance Criteria

- **AC-1.** *Given* a course of images, *When* `alt-text generate --wait`, *Then* alt text is produced with a summary.
- **AC-2.** *Given* a video item, *When* `captions generate --wait`, *Then* a caption track is attached.
- **AC-3.** *Given* a course, *When* `translations coverage --json`, *Then* per-locale coverage % prints.

## 8. Data Model

- None client-side.

## 9. API Surface

- `alt_text_http.go`; caption routes; `course_translation.go` + `registerTranslationRoutes`; `accessibility_http.go`; `registerTranscodeRoutes`; `registerTusRoutes`; `registerTTSRoutes`; `registerReadingLevelRoutes`.

## 10. UI / UX

- `lextures alt-text|captions|translations|accessibility|media|tts ...`.
- Batch/transcode use the shared `--wait` primitive.

## 11. AI / ML Considerations

- Alt-text, captions (ASR), translation (MT), TTS are AI-backed server-side; CLI triggers/reads. Cost surfaced where server provides it; opt-out honored.

## 12. Integration Points

- Server accessibility/media/translation handlers; TUS pipeline (existing `files upload`); jobs (C18).

## 13. Dependencies & Sequencing

- After: C02 (items to caption/translate), C40.
- Before: none.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Large media transcode timeouts | M | M | Async + `--wait --timeout`; TUS resume |
| MT/ASR quality | M | L | Mark generated content as machine-generated; human-review flag |

## 15. Rollout Plan

- Ship alt-text + captions + translations first (WCAG value), then transcode/TUS/TTS.
- Rollback: additive.

## 16. Test Plan

- **Unit** — locale validation; `.vtt` parse; skip-existing.
- **Integration** — generate job; coverage shape.
- **E2E** — generate alt text for a course → verify.

## 17. Documentation & Training

- "Batch-caption a media library for WCAG" recipe.

## 18. Open Questions

1. Are generation endpoints per-item or per-course batch?
2. Does translation cover structured content (pages) or only strings?

## 19. References

- `alt_text_http.go`, `course_translation.go`, `accessibility_http.go`, caption/transcode/TUS routes.
- Related: [C02](C02-modules-course-structure.md), [C05](C05-content-extras.md), [C29](C29-compliance-privacy.md).
