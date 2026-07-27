# CT.19 — Tool: Media Checkpoints (questions injected at timestamps in a video or audio clip)

> Implementation plan. Source: new capability — interactive tools inside content sections. Folder overview: [README](../plan/content_tools/README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | CT.19 |
| **Section** | Content Tools (CT) — tool shelf |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | SHIPPED |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Media / web platform team |
| **Depends on** | CT.1, CT.2, CT.3; shipped `service/video`, `service/captions` |
| **Unblocks** | Flipped-classroom and lecture-capture workflows; watch-time evidence |

---

## 1. Problem Statement

Video is the most-assigned and least-evidenced content in every course: the platform can host and play
it, but has no idea whether anyone watched, understood or fell asleep at minute 4. Instructors compensate
with a follow-up quiz that tests memory of the whole video rather than comprehension at the moment it
mattered. Interpolated testing — a question at the moment the concept lands — is one of the most robust
findings in the media-learning literature, and it also produces the engagement data instructors actually
need. Today an author cannot place a single question inside a video in Lextures.

## 2. Goals

- Let an author attach questions to timestamps in a hosted video or audio clip, played inline.
- Pause at each checkpoint, ask, give feedback, and resume — with the learner's answers stored per
  enrollment.
- Record meaningful watch evidence (segments actually watched, not just "opened"), without pretending
  to be a surveillance tool.
- Preserve accessibility fully: captions, transcript, keyboard control, and a **transcript-only path**
  for learners who cannot use the media.
- Give instructors a drop-off curve plus per-checkpoint results — where attention and understanding fail.

## 3. Non-Goals

- Hosting or transcoding changes (the shipped video pipeline is used as-is).
- Third-party embeds (YouTube/Vimeo) in v1 — privacy review and API differences make them a separate
  story; the config schema reserves the shape.
- Proctoring behaviours (blocking seek, forcing full watch) beyond an optional, clearly-labelled
  "no skipping past unanswered checkpoints" setting.
- Editing or clipping media inside the tool.

## 4. Personas & User Stories

- **As an instructor**, I want two questions inside my 8-minute lecture clip so that students check
  their understanding while it is fresh.
- **As an instructor**, I want to see that 40% of students stopped at 3:20 so that I can fix that part
  of the video.
- **As a student**, I want the video to pause and ask me so that I notice when I have drifted.
- **As a Deaf student**, I want captions and a transcript so that the activity is equivalent.
- **As a student on a phone with limited data**, I want to complete the activity from the transcript if
  I cannot stream.
- **As an instructor**, I want to reset a student's checkpoints so that they can rewatch properly.

## 5. Functional Requirements

- **FR-1.** The author MUST select a media asset from course files (video or audio) and define
  checkpoints, each with a timestamp and a question (types reuse the CT.11 set: `single`, `multi`,
  `true_false`, `short_text`, `numeric`), plus optional per-option feedback.
- **FR-2.** The player MUST pause at each checkpoint and present its question inline over/below the
  player; answering MUST be a server action (scoring server-side, keys never sent early).
- **FR-3.** The author MUST configure: whether a checkpoint is required to continue, attempts per
  question, whether feedback is shown immediately, and whether seeking past unanswered checkpoints is
  prevented.
- **FR-4.** Captions MUST be shown when available and MUST be required for any video used by this tool
  where the platform's caption policy demands it (reusing shipped caption enforcement).
- **FR-5.** A **transcript panel** MUST be available, time-linked (clicking a line seeks), and a
  **transcript-only mode** MUST allow completing every checkpoint question without playing the media.
- **FR-6.** The player MUST be fully keyboard-operable (play/pause, seek by arrow keys, volume, captions,
  speed) with visible focus and standard shortcuts documented in-tool.
- **FR-7.** State MUST record: per-checkpoint answers and attempts, watched segments (coarse, ≥ 5 s
  granularity), furthest position, and completion.
- **FR-8.** Watch tracking MUST be coarse and honest: it records *segments played*, does not attempt to
  detect attention, and its limits are documented to instructors so it is not misused as proof of
  attendance.
- **FR-9.** Scoring MUST be the fraction of checkpoints answered correctly (last attempt by default),
  reported for optional CT.7 bridging; `practiceOnly` mode reports nothing.
- **FR-10.** Playback speed, volume and caption preferences MUST persist per learner (client
  preference, not tool state).
- **FR-11.** The instructor MUST see: a drop-off curve (learners still watching over time), per-checkpoint
  correctness, and the most-missed checkpoint.
- **FR-12.** The tool MUST degrade gracefully when the media fails to load: transcript-only mode is
  offered automatically and the activity remains completable.
- **FR-13.** CT.4 reset MUST clear answers and watch data.
- **FR-14.** `prefers-reduced-motion` MUST suppress any non-essential animation; autoplay MUST never be
  used.

## 6. Non-Functional Requirements

- **Performance** — Checkpoint pause fires within 250 ms of its timestamp; player mount ≤ 120 ms;
  renderer ≤ 38 KB gz (native `<video>` plus a thin controls layer, no heavy player library).
- **Security** — Media served through existing course-file authorization with signed URLs; answer keys
  server-side; watch data cannot be forged into a grade because only checkpoint answers score.
- **Privacy & Compliance** — Watch data is behavioural data about a student: disclosed to learners,
  included in DSAR, retained per policy, never used for automated adverse decisions. K-12 guidance
  explicitly discourages using it as attendance evidence.
- **Accessibility** — WCAG 2.1 AA: captions (1.2.2), audio description support where the asset provides
  it (1.2.5), transcript (1.2.3), keyboard operation (2.1.1), no seizure-inducing content policy,
  contrast on controls, and — the key design decision — a **fully equivalent transcript-only path**.
- **Scalability** — Watch segments stored as compact ranges, capped; drop-off curve aggregated in CT.7.
- **Reliability** — Answers persist independently of playback; a network blip during playback never
  loses an answered checkpoint.
- **Observability** — `lextures_content_tool_checkpoints_total{result}`, drop-off percentiles,
  transcript-only usage rate, media-load failure rate.
- **Maintainability** — Player controls layer is shared with any future media tool.
- **Internationalization** — Caption language selection; RTL layout for transcript; localized controls.
- **Backward compatibility** — Additive; existing video embeds unchanged.

## 7. Acceptance Criteria

- **AC-1.** *Given* a checkpoint at 02:15, *When* playback reaches it, *Then* the media pauses within
  250 ms and the question appears with focus moved to it.
- **AC-2.** *Given* an unanswered required checkpoint, *When* the learner tries to seek past it with
  `preventSkip` on, *Then* the seek is clamped and an explanatory message is announced.
- **AC-3.** *Given* the learner answers, *Then* scoring happens server-side, feedback appears per config,
  and playback resumes on request (never automatically without consent).
- **AC-4.** *Given* the learner uses transcript-only mode, *Then* every checkpoint question is reachable
  and answerable, and completion is recorded identically.
- **AC-5.** *Given* the media fails to load, *Then* transcript-only mode is offered automatically and no
  error state blocks the activity.
- **AC-6.** *Given* the learner watches 0:00–3:00 and 5:00–6:00, *Then* state records those segments at
  ≥ 5 s granularity and the furthest position is 6:00.
- **AC-7.** *Given* 30 learners, *When* the instructor opens insights, *Then* the drop-off curve and
  per-checkpoint correctness match the aggregated state.
- **AC-8.** *Given* keyboard-only operation with a screen reader, *Then* play/pause, seek, captions,
  checkpoint answering and resume are all operable and announced.
- **AC-9.** *Given* a video without captions and a platform policy requiring them, *Then* the author is
  blocked (or warned per setting) at authoring time.
- **AC-10.** *Given* a CT.4 reset, *Then* answers and watch data are cleared and snapshotted.

## 8. Data Model

**No migration.**

```ts
// configSchema
type MediaCheckpointsConfig = {
  media: { source: 'course_file'; fileId: string; kind: 'video' | 'audio'; durationSec: number }
  captionsTrackId?: string
  transcriptSource?: 'captions' | 'inline'
  transcriptMarkdown?: string
  checkpoints: Array<{
    id: string
    atSec: number
    question: {
      type: 'single' | 'multi' | 'true_false' | 'short_text' | 'numeric'
      prompt: string
      options?: Array<{ id: string; text: string; correct: boolean; feedback?: string }>  // correct/feedback x-lex-sensitive
      acceptedAnswers?: string[]        // x-lex-sensitive
      correctValue?: number             // x-lex-sensitive
      tolerance?: { kind: 'absolute' | 'relative'; value: number }
    }
    required: boolean                   // default true
    attempts: number                    // default 2
    showFeedback: boolean               // default true
  }>
  preventSkipPastUnanswered: boolean    // default false
  practiceOnly: boolean                 // default true
}

// stateSchema
type MediaCheckpointsState = {
  v: 1
  answers: Record<string, { attempts: Array<{ value: string | string[] | number; correct: boolean; at: string }>; done: boolean }>
  watchedSegments: Array<[number, number]>   // seconds, merged, coarse
  furthestSec: number
  usedTranscriptOnly?: boolean
  scoreRaw?: number
  scoreMax?: number
  completedAt?: string
}
```

`scoring.mode = 'auto'` unless `practiceOnly`; `capabilities = ['state','scoring','media']`;
`maxStateBytes = 32000`.

## 9. API Surface

**No new routes.**

- `PUT .../state` — watch segments and progress (coalesced, throttled to ≤ 1 write per 15 s).
- `POST .../actions/answerCheckpoint` — `{checkpointId, value, idempotencyKey}` →
  `{correct, feedback?, attemptsRemaining, state}`.
- Insights via CT.7 facets `checkpointId`, `correct`, plus watch-segment aggregation into a drop-off
  curve computed server-side.

## 10. UI / UX

1. Player with standard controls, caption toggle, speed control, and **checkpoint markers** on the
   timeline (distinct shapes for answered/unanswered, not colour-only).
2. At a checkpoint: playback pauses, a question card appears beneath the player (not covering it), focus
   moves to the question, and a **Continue** button appears after answering.
3. Side/below: **Transcript** panel with time-linked lines and a search field; a **Transcript only**
   toggle prominently available.
4. Footer: progress ("2 of 3 checkpoints answered"), furthest position, and caption/speed preferences.

**States** — *Not started*, *Playing*, *At checkpoint*, *Answered (continue)*, *Complete*, *Media
unavailable (transcript offered)*, *Read-only*, *Error*.

**Mobile** — player full width, question card below the fold-safe area, transcript in a bottom sheet;
no autoplay; controls sized ≥ 44 px.

**Accessibility** — native media element semantics preserved; captions on by default when the learner's
preference says so; timeline markers exposed as a list of checkpoints with times; pause and question
appearance announced ("Paused at 2 minutes 15 seconds. Question 2 of 3."); no keyboard trap between
player and question.

**Copy & i18n** — `contentTools.tools.mediaCheckpoints.*`.

**Authoring** — custom editor: pick media → scrub to a moment → **Add checkpoint here** (timestamp
prefilled) → write the question → preview the learner experience end-to-end; caption/transcript status
shown with a warning when missing.

## 11. AI / ML Considerations

None in v1. Reserved: (a) **generate checkpoint questions from the transcript** via `ui.aiAssist`,
reusing quiz-generation prompts and requiring author review — this is the single highest-value AI assist
in the whole shelf because authoring timestamps and questions is the real cost; (b) auto-suggesting
checkpoint *positions* from transcript topic shifts. Both deferred, both disclosed when built.

## 12. Integration Points

- **Internal** — `service/contenttools/tools/mediacheckpoints/`, `service/video`, `service/captions`,
  `service/vttformatter`, `service/filestorage` (signed media URLs),
  `clients/web/src/components/content-tools/tools/media-checkpoints/`, shared media-controls layer.
- **CT.7** — checkpoint facets, drop-off aggregation.
- **Seat time** — watched segments inform `service/seattime` where the org uses it, with the same
  honesty caveats.

## 13. Dependencies & Sequencing

- **Must ship after:** CT.1–CT.3.
- **Must ship before:** nothing.
- **Shared infra needed:** shipped media hosting and captioning.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Watch data misused as attendance/compliance proof | H | H | Documented limits, no attention detection, explicit instructor guidance, no automated adverse decisions (CT.8) |
| Inaccessible for Deaf/HoH or low-bandwidth learners | M | H | Captions required by policy, transcript-only equivalent path, media-failure fallback |
| `preventSkip` feels punitive | M | M | Off by default, clearly labelled at authoring, never blocks the transcript path |
| Checkpoint timing drift across browsers | M | M | `timeupdate` plus a rAF check with a 250 ms tolerance test on Chrome/Safari/Firefox |
| Authoring is slow (scrub + write questions) | H | M | Add-checkpoint-at-playhead flow, keyboard shortcuts, AI generation reserved as the next investment |
| Large media on mobile data | M | M | No autoplay, quality selection from the shipped pipeline, transcript-only path |

## 15. Rollout Plan

- **Feature flag** — course tool allowlist.
- **Sequencing** — media controls layer → checkpoint engine + timing tests → question card (reusing
  CT.11 grading) → transcript panel and transcript-only mode → authoring editor → insights → pilot.
- **Dogfood** — one flipped-classroom unit with three videos.
- **GA criteria** — timing accuracy across browsers; transcript-only equivalence verified; a11y audit
  passed; caption policy enforced.
- **Rollback** — remove from the allowlist; answers preserved.

## 16. Test Plan

- **Unit** — checkpoint firing logic (including seeks backwards/forwards, rapid seeking, speed changes);
  segment merging and coarsening; scoring; skip prevention.
- **Integration** — answer actions and key redaction; state throttling; reset; caption enforcement at
  authoring.
- **End-to-end** — Playwright with a fixture clip: play → checkpoint → answer → resume → complete;
  transcript-only completion; media-load failure fallback; skip prevention.
- **Security** — media URL authorization across courses; payload inspection for keys; forged watch data
  attempting to affect a score (must not).
- **Accessibility** — axe; screen-reader script for the full loop; caption rendering; keyboard-only
  operation; focus movement at checkpoints.
- **Performance** — checkpoint latency across browsers; state write frequency under seeking; chunk budget.
- **Manual exploratory** — long videos, audio-only assets, slow networks, background-tab behaviour,
  Safari/iOS media quirks.

## 17. Documentation & Training

- **Instructor** — authoring checkpoints; what watch data does and does not prove; captions requirement;
  reading the drop-off curve.
- **Student** — controls, transcript mode, that answers save automatically.
- **Admin** — media policy interaction, retention of watch data.

## 18. Open Questions

1. Should third-party embeds (YouTube) be supported later given their tracking implications? Proposed:
   only via a privacy-reviewed proxy pattern, as a separate story.
2. Should audio description be required for videos where it materially matters, or recommended?
   Proposed: recommended with a warning; requiring it needs an authoring pipeline we do not have.
3. Should the drop-off curve be shown to students about themselves? Proposed: show their own watch
   coverage only — useful for review, not comparative.

## 19. References

- Existing files this work touches: `server/internal/service/video/`, `server/internal/service/captions/`,
  `server/internal/service/vttformatter/`, `server/migrations/158_captions.sql`,
  `server/migrations/209_video_captions_accessibility.sql`,
  `clients/web/src/components/content-tools/`.
- External standards: WCAG 2.1 AA — 1.2.2 Captions, 1.2.3 Audio Description or Media Alternative,
  1.2.5, 2.1.1; HTML media element accessibility guidance.
- Related plans: [CT.11](CT.11-tool-inline-questions.md),
  [CT.7](CT.7-analytics-insights-and-gradebook.md),
  [CT.8](CT.8-governance-safety-privacy-accessibility.md).
