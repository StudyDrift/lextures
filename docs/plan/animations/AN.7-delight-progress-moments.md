# AN.7 — Delight & Progress Moments

> Implementation plan. Source: [docs/plan/animations/README.md](README.md) (Motion & Animation Polish initiative).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | AN.7 |
| **Section** | Motion & Animation Polish |
| **Severity** | MINOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | PARTIAL — intro-completion celebration exists and is reduced-motion aware; most progress/gamification/quiz feedback is static |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Frontend Platform (web) + Mobile + Learning/Gamification |
| **Depends on** | AN.1, AN.6 |
| **Unblocks** | — |

---

## 1. Problem Statement

Learning products earn emotional engagement at moments of achievement, and Lextures has many —
streaks, badges, XP, mastery, quiz answers, course/module completion — but most land silently. The
intro-completion celebration is the one strong example and it already respects reduced motion
([`intro-completion-celebration.tsx`](../../../clients/web/src/components/intro-course/intro-completion-celebration.tsx),
iOS `IntroCompletionCelebrationSheet`). Elsewhere, progress bars/rings snap to their value, a correct
quiz answer just recolors, badges/XP appear without fanfare, and mastery updates are invisible. These
are the highest-delight opportunities in the app: animating them (tastefully, accessibly) makes
progress feel earned and the product feel alive.

## 2. Goals

- Animate progress representations: bars/rings/meters fill from prior→new value; mastery and
  completion updates count/sweep up rather than snap.
- Add tasteful feedback to quiz/assessment answers: correct/incorrect states animate (with haptics
  from AN.6), streaks build, live-quiz leaderboard positions transition.
- Standardize achievement moments — badges, XP, streaks, level-ups, course/module completion — on a
  shared, reusable "delight" primitive (with an optional confetti/burst, capped and reduced-motion
  aware).
- Ensure every delight moment has a calm, non-motion fallback and never blocks progression or input.

## 3. Non-Goals

- Core control feedback (button press, toggle, validation) — that is [AN.6](AN.6-micro-interactions-controls.md).
- Changing gamification rules, scoring, or what earns a badge/XP.
- Heavy Lottie/video pipelines (may evaluate a lightweight burst asset, but default is code-driven
  motion).
- Sound design.

## 4. Personas & User Stories

- **As a K-12 student answering a quiz correctly**, I want a satisfying pop + streak build so it feels
  rewarding.
- **As a self-learner completing a module**, I want the progress ring to sweep up and a modest
  celebration so the milestone feels real.
- **As a student in a live quiz**, I want the leaderboard to animate my position change so competition
  feels dynamic.
- **As a learner earning a badge/XP**, I want it to arrive with a small flourish that I can also see
  later without motion.
- **As a motion-sensitive user or in a serious/exam context**, I want celebrations to be calm or off —
  never flashing, never blocking my next action.

## 5. Functional Requirements

- **FR-1.** Progress bars, rings, and meters MUST animate from the previous value to the new value
  (ease/bubble) instead of snapping; numeric counters MAY count up over ≤ `deliberate`.
- **FR-2.** Mastery/skill updates and course/module completion MUST show a fill/sweep + a modest
  completion flourish.
- **FR-3.** Quiz/assessment answer feedback MUST animate correct (bubble pop + accent) and incorrect
  (single shake/pulse, from AN.6) states, paired with haptics on mobile; feedback MUST NOT delay
  advancing to the next item.
- **FR-4.** Live-quiz leaderboards MUST animate rank/score changes (position transitions from AN.4 +
  score count-up); podium reveals MUST be staggered.
- **FR-5.** Achievement moments (badge earned, XP gained, streak milestone, level-up) MUST use a shared
  `DelightMoment` primitive with an optional confetti/particle burst that is **capped** (particle
  count/duration bounded) and centered on the earned element.
- **FR-6.** Every delight animation MUST have a reduced-motion fallback: progress sets instantly or
  fades, celebrations show a static badge/checkmark with no particles or flashing.
- **FR-7.** No delight animation may flash more than 3× per second (WCAG 2.3.1) and none may block
  input, navigation, or answer submission — all are dismissible/skippable.
- **FR-8.** Delight moments MUST be context-gated: suppressed or toned down in exam/proctored/serious
  contexts and where the org disables gamification.
- **FR-9.** The confetti/burst MUST be performance-bounded (particle cap, auto-stop, transform/opacity)
  and MUST clean up fully (no lingering canvas/DOM).

## 6. Non-Functional Requirements

- **Performance** — particle/burst effects are capped and GPU-friendly; progress animation is
  transform/opacity or cheap canvas; 60fps on low-end; effects auto-teardown to avoid leaks.
- **Security** — None.
- **Privacy & Compliance** — None (uses existing achievement data).
- **Accessibility** — no >3Hz flashing (2.3.1); achievements announced via text/live region, not motion
  alone (1.4.1); reduced-motion fallback; nothing blocks input (2.2.2 style pause/skip); color-blind-
  safe correct/incorrect (icon + color).
- **Scalability** — one delight primitive reused across all achievement types.
- **Reliability** — rapid consecutive achievements queue/coalesce rather than stacking chaotically;
  interruption cleans up.
- **Observability** — may reuse existing gamification events; no new PII. Optionally count reduced-
  motion suppressions.
- **Maintainability** — shared `DelightMoment` + progress-animation helpers per platform; feature code
  triggers, doesn't hand-roll.
- **Internationalization** — count-up respects locale number formatting; RTL-safe layout; no text baked
  into effects.
- **Backward compatibility** — un-migrated achievement surfaces keep static behavior; incremental
  adoption.

## 7. Acceptance Criteria

- **AC-1.** *Given* a progress ring/bar updates, *When* the value changes, *Then* it animates from old
  to new value (and reduced-motion sets it instantly).
- **AC-2.** *Given* a correct quiz answer, *When* submitted, *Then* it pops with accent + haptic and
  the next item is reachable without waiting; an incorrect answer shakes once.
- **AC-3.** *Given* a badge/XP/streak is earned, *When* it triggers, *Then* the shared `DelightMoment`
  plays a capped celebration centered on the element, and the earned item remains visible afterward.
- **AC-4.** *Given* a live-quiz leaderboard update, *When* ranks change, *Then* rows transition to new
  positions and scores count up.
- **AC-5.** *Given* reduced motion or an exam/serious context, *When* an achievement occurs, *Then* it
  shows a static, non-flashing indicator with no particles.
- **AC-6.** *Given* several achievements fire in quick succession, *When* they occur, *Then* they
  queue/coalesce and all effects fully tear down (no lingering canvas/DOM, no memory growth).

## 8. Data Model

- No database changes. Consumes existing progress/mastery/gamification/quiz data and events.
- No migration; no backfill.

## 9. API Surface

- No HTTP/WebSocket surface (consumes existing gamification/quiz endpoints & realtime events).
  Internal:
  - Web: `<AnimatedProgress>`/`useCountUp()`, a `<DelightMoment>` component (capped confetti), quiz-
    answer feedback hooks; reuse `book-loader.css`/existing celebration where sensible.
  - iOS: `AnimatedProgressRing`, a `DelightMoment` view, extend `IntroCompletionCelebrationSheet`
    patterns; `Haptics` from AN.6.
  - Android: `AnimatedProgress` composable + `DelightMoment`; `Haptics` from AN.6.

## 10. UI / UX

- **Modified surfaces** — dashboards' progress/stat tiles, mastery/skill views (iOS `Features/Mastery`,
  reading levels), gamification (iOS/Android `features/gamification`, XP/streaks/badges), quiz play
  ([`live-quiz-play-page.tsx`](../../../clients/web/src/pages/lms/live-quiz-play-page.tsx), iOS/Android
  Quiz features), live-quiz leaderboards, course/module completion, intro-course completion (already
  done — align to the shared primitive), portfolio/credential/wallet earn moments.
- **Key flows** — (1) answer a quiz item; (2) complete a module/course; (3) earn a badge/XP/streak;
  (4) live-quiz leaderboard update; (5) mastery level change.
- **Empty/loading/error/offline** — no achievement → no effect; offline-earned achievements celebrate
  on next view (once), not repeatedly.
- **Mobile/responsive** — effects scale to viewport; haptics on mobile; particle cap lower on low-end.
- **Accessibility** — live-region achievement text, icon+color correctness signals, reduced-motion +
  exam-context suppression, no flashing.
- **Copy & i18n** — reuse existing achievement strings; locale-aware number count-up.

## 11. AI / ML Considerations

Not applicable (no model use). If AI-generated praise copy is ever attached to a celebration, it would
follow existing AI-content policies — out of scope here.

## 12. Integration Points

- **Web** — quiz play pages, gamification/progress components, `intro-completion-celebration.tsx`
  (align), `book-loader.css`, dashboards.
- **iOS** — `Features/Gamification`, `Features/Mastery`, `Features/Quiz`, `IntroCompletionCelebrationSheet`.
- **Android** — `features/gamification`, `features/quiz`, progress/mastery composables.
- Consumes AN.1 tokens and AN.6 haptics; leaderboard motion reuses AN.4.

## 13. Dependencies & Sequencing

- Must ship **after**: AN.1 and AN.6 (uses press/haptics helpers); leaderboard reuses AN.4.
- Must ship **before**: nothing.
- Shared infra: existing gamification/quiz data & realtime; no new backend.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Celebrations feel childish/annoying for HE/adult contexts | M | M | UI-mode + org gamification settings gate intensity; restrained default for HE/SL |
| Flashing/particles trigger photosensitivity | L | H | Hard 3Hz cap, reduced-motion off-switch, particle cap, QA against WCAG 2.3.1 |
| Confetti canvas leaks memory / drops frames | M | M | Particle cap, auto-teardown, perf traces, leak tests (AC-6) |
| Celebration blocks quiz flow / answer submission | M | H | Non-blocking, skippable, runs alongside advance (FR-3/FR-7) |
| Repeated achievements stack into chaos | M | M | Queue/coalesce; single at a time (AC-6) |

## 15. Rollout Plan

- **Feature flag** — `ff_motion_delight` (default off → on after QA); respects existing gamification
  enable/disable and exam contexts.
- **Sequencing** — land progress animation + shared `DelightMoment` behind flag → quiz answer feedback
  → badges/XP/streaks → leaderboards → completion moments → enable by default per market/UI-mode.
- **Dogfood** — internal; taste review across K-12 vs HE tone; photosensitivity check.
- **GA criteria** — WCAG 2.3.1 verified, reduced-motion + exam suppression verified, no leaks, non-
  blocking confirmed, tone approved per market.
- **Rollback** — flip `ff_motion_delight` off.

## 16. Test Plan

- **Unit** — progress helper interpolates old→new and sets instantly under reduced motion; count-up
  respects locale; delight queue coalesces.
- **Integration** — iOS/Android quiz correct/incorrect feedback + haptic; leaderboard rank transition;
  completion flourish.
- **End-to-end** — Playwright: progress ring animates on change; badge earn plays capped celebration
  and tears down; reduced-motion emulation → static; exam context → suppressed.
- **Security** — n/a.
- **Accessibility** — no >3Hz flashing (automated frame-rate/flash check + manual), live-region
  announcements, color+icon correctness, reduced-motion fallback; axe clean.
- **Performance / load** — particle burst within budget on low-end; memory stable across many
  celebrations (leak test); 60fps.
- **Manual exploratory** — rapid correct answers, back-to-back badges, offline-earned replay-once, RTL,
  K-12 vs HE tone.

## 17. Documentation & Training

- **Internal** — "Adding a delight moment" recipe; the intensity/tone matrix per market/UI-mode;
  accessibility rules (flashing cap, reduced-motion, non-blocking).
- **End-user / admin** — note the org-level gamification/celebration controls and the reduce-motion
  setting.
- **Runbook** — `ff_motion_delight` scope and kill-switch; how exam-context suppression is wired.

## 18. Open Questions

1. Celebration tone/intensity per market and UI mode (K-2 playful vs HE restrained) — design + product.
2. Do we invest in a lightweight burst asset, or keep everything code-driven?
3. Which contexts count as "serious/exam" for suppression, and is that an existing flag we can read?
4. Should offline-earned achievements celebrate once on reconnect, or stay silent?

## 19. References

- Existing: [`clients/web/src/components/intro-course/intro-completion-celebration.tsx`](../../../clients/web/src/components/intro-course/intro-completion-celebration.tsx),
  `clients/ios/Lextures/Features/IntroCourse/IntroCompletionCelebrationSheet.swift`,
  [`clients/web/src/pages/lms/live-quiz-play-page.tsx`](../../../clients/web/src/pages/lms/live-quiz-play-page.tsx),
  [`clients/web/src/components/quiz/book-loader.css`](../../../clients/web/src/components/quiz/book-loader.css),
  iOS `Features/Gamification` & `Features/Mastery`, Android `features/gamification` & `features/quiz`.
- Standards: WCAG 2.3.1 (Three Flashes), 1.4.1 (Use of Color), 2.2.2 (Pause/Stop/Hide), 4.1.3;
  Apple HIG "Feedback"; Material 3 motion.
- Related plans: [AN.1](../../completed/animations/AN.1-motion-foundation-tokens.md), [AN.4](../../completed/animations/AN.4-lists-collections-motion.md),
  [AN.6](AN.6-micro-interactions-controls.md).
