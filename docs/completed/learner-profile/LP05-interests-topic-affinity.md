# LP05 — Interests & Topic Affinity (Facet)

> Implementation plan. A facet deriver on top of [LP01](LP01-foundation-derivation-engine.md).
> Primary signal: enrollments, notebooks, feed activity, content read. Follows
> [../_TEMPLATE.md](../_TEMPLATE.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | LP05 |
| **Section** | Learner Profile |
| **Severity** | MINOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | COMPLETED |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Backend platform team |
| **Depends on** | LP01; enrollments, notebooks (262/242), feed (064/241) |
| **Unblocks** | LP07, LP09 (recommendation topic seeds) |

---

## 1. Problem Statement

A learner's interests are implicit in what they choose to engage with: the courses and subjects they
enroll in, the topics they keep notes and flashcards on, the content they read deeply, and the feed
threads they participate in. Lextures never surfaces this. This facet derives **topic affinity** — the
subjects a learner gravitates to and goes deep on — autonomously and with evidence, so the profile can
say "you keep coming back to statistics and ecology," and LP09 can seed recommendations from real
interest rather than only from gaps.

## 2. Goals

- Derive a ranked set of **topics/subjects** the learner engages with, weighted by depth of
  engagement (not just enrollment), across courses, notebooks, deep reads, and feed participation.
- Distinguish **assigned** engagement (required coursework) from **elective/self-directed** interest
  (self-created notebooks, optional reading, feed participation) — the latter is the stronger signal.
- Attach evidence (which courses/notebooks/threads, counts, windows) per LP01.
- Degrade to `insufficient_data` under threshold.

## 3. Non-Goals

- Semantic topic modelling / embeddings of free text (v1 uses existing structured tags/subjects; an
  embedding upgrade is a later option, see Open Questions).
- Mastery of those topics (LP04) or format preference (LP03).
- Any social-graph inference beyond the learner's own feed participation.

## 4. Personas & User Stories

- **As a self-learner**, I want to see the subjects I keep returning to, so I can pick my next course
  with intent.
- **As a student**, I want the platform to notice I take lots of notes on ecology, so what it
  recommends feels like me.
- **As the adaptive layer (LP09)**, I want interest topics to seed recommendations alongside gaps.

## 5. Functional Requirements

- **FR-1.** The `interests` deriver MUST aggregate topic signals from: course/subject enrollment
  (`course.course_enrollments` + course subject/category), notebooks (`student_notebooks`,
  `notebook_tasks`, flashcards) and their topics, deep content reads (high engagement from LP03's
  underlying events), and feed participation (`064_course_feed`, `241` channels).
- **FR-2.** MUST weight **self-directed** signals (self-created notebooks, optional reading, feed
  posts) above **assigned** signals when ranking affinity.
- **FR-3.** MUST produce a ranked, capped list of topics with an affinity score and the mix of signal
  sources behind each.
- **FR-4.** Each topic insight MUST carry evidence naming the contributing courses/notebooks/threads
  and counts, per LP01.
- **FR-5.** MUST return `insufficient_data` when the learner has too few distinct topic signals
  (default: < 2 topics with ≥ 2 signals each).
- **FR-6.** MUST NOT invent topics with no structured source; unlabelled activity contributes to depth
  but not to a named topic.

## 6. Non-Functional Requirements

- **Performance** — Aggregation over a learner's enrollments/notebooks/feed; derive ≤ 700 ms.
- **Security / Privacy** — Self-only via LP01. Feed participation read only for the learner's own
  posts/threads. FERPA/GDPR via LP01/LP08.
- **Accessibility** — Topic chips/lists in LP07 use text, not color alone.
- **Scalability** — Bounded by learner's own artifacts; incremental on new notebook/feed activity.
- **Observability** — LP01 metrics `facet="interests"`.
- **Internationalization** — Topic labels from existing localised subject/tag data.

## 7. Acceptance Criteria

- **AC-1.** *Given* a learner with 3 self-created notebooks on ecology and heavy optional reading in
  ecology, *then* ecology ranks as a top interest with self-directed evidence.
- **AC-2.** *Given* two topics with equal raw counts but one only from required coursework and one
  from self-created notebooks, *then* the self-directed topic ranks higher (FR-2).
- **AC-3.** *Given* activity with no structured topic, *then* no fabricated topic appears (FR-6).
- **AC-4.** *Given* a learner with a single subject, *then* facet is `insufficient_data`.

## 8. Data Model

No new tables — writes LP01 facet tables with `facet_key='interests'`. Reads
`course.course_enrollments` + course subject/category, `student_notebooks`/`notebook_tasks`/flashcards
(262/242/219), feed tables (064/241), and deep-read events (via LP03's engagement source).

## 9. API Surface

Served by LP01 `GET /me/learner-profile/facets/interests`. Example value:
`{ "topics":[{"topic":"Ecology","affinity":0.78,"sources":{"notebooks":3,"reading":12,"feed":4},
"selfDirected":true},{"topic":"Statistics","affinity":0.6,"sources":{"courses":1,"quizzes":20}}] }`.

## 10. UI / UX

Rendered by LP07 as "What you're drawn to" (ranked topic chips with a source breakdown on drill-in).
Empty state = `insufficient_data`.

## 11. AI / ML Considerations

v1 is structured aggregation (no model). A later upgrade could embed notebook/content text to cluster
topics; that upgrade MUST redact PII and only run server-side (deferred — Open Questions).

## 12. Integration Points

- `server/internal/service/learnerprofile/derivers/interests.go` (new).
- Reads enrollments, notebooks (`me_notebook.go`, `student_notebooks.go`), feed (`feedevents`,
  `feed_websocket.go`), course subject/category metadata.

## 13. Dependencies & Sequencing

- After LP01. Parallel with other facets. Feeds LP07, LP09. Weakest-signal facet — lowest priority
  of the five if sequencing is constrained.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Interests dominated by required courses (weak signal) | H | M | Self-directed weighting (FR-2); show source mix so it's honest |
| Topic labels too coarse (course-level only) | M | M | Use notebook/tag topics where available; degrade gracefully |
| Feed content sensitive | M | M | Only the learner's own participation; no content text stored in evidence |

## 15. Rollout Plan

Behind `learner_profile_enabled`. Register deriver → pilot → GA with LP07. Rollback: unregister.

## 16. Test Plan

- **Unit** — self-directed weighting; topic ranking/cap; no-topic exclusion; sufficiency.
- **Integration** — seed notebooks/enrollments/feed → derive → assert ranked topics + evidence.
- **E2E** — seeded learner shows interest topics via profile API.
- **Performance** — derive ≤ 700 ms.

## 17. Documentation & Training

- Student help: "What you're drawn to — how interests are inferred and where they come from."

## 18. Open Questions

1. Is course "subject/category" granular enough, or do we need per-module topic tags?
2. Embedding-based topic clustering of notebook/content text — worth it, and privacy cost?
3. Should completed vs abandoned courses weight interest differently?

## 19. References

- [LP01](LP01-foundation-derivation-engine.md).
- Existing: `student_notebooks.go`, `me_notebook.go`, `notebook_tasks.go`, `064_course_feed.sql`,
  `241_feed_announcements_channel.sql`, `course.course_enrollments`.
