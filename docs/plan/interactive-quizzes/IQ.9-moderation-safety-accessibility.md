# IQ.9 — Moderation, Safety, Accessibility & Fair Play

> Implementation plan. Source: net-new capability (interactive game-based quizzing). Landscape: [interactive-quizzes/README](README.md). Reuses `contentfilter` for text moderation and the shipped compliance engines under [`../standards/`](../standards/); provides the enforcement hooks that [IQ.3](../../completed/interactive-quizzes/IQ.3-live-game-hosting-engine.md)/[IQ.4](../../completed/interactive-quizzes/IQ.4-player-join-and-gameplay.md) call.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | IQ.9 |
| **Section** | Interactive Quizzes |
| **Severity** | BLOCKER (K12) / MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | MISSING |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Assessment squad + Trust & Safety |
| **Depends on** | IQ.3, IQ.4 |
| **Unblocks** | (gates guest-join and public catalog for IQ.4/IQ.8) |

---

## 1. Problem Statement

The moment students type a **nickname** and answers appear on a class projector, you inherit a safety surface:
inappropriate nicknames on the big screen, students joining games they shouldn't, one clever kid opening two
tabs to farm points, and screen-reader/colour-blind learners locked out of a colour-coded game. IQ.9 makes
Live Quizzes **safe and fair by default** — nickname moderation, join/anti-cheat controls, host safety tools,
and a hardened accessibility layer — and is a **hard gate** before any guest-join or public surface is exposed
to minors.

## 2. Goals

- **Nickname moderation:** filter/deny inappropriate nicknames (reuse `contentfilter`), with a host "mute
  names" and rename/kick control, so nothing offensive hits the projector.
- **Fair play / anti-cheat:** answer shuffling, one-active-session-per-identity, join limits, and detection of
  obvious multi-tab/self-farming, without punishing legitimate reconnects.
- **Host safety tools:** kick/ban a player for the game, lock the lobby, hide names, pause — all
  server-enforced.
- **Accessibility hardening (AA):** answers by shape+label (not colour), full keyboard play, screen-reader
  announcements, reduced-motion, and **no flashing** (photosensitivity) across host/projector/player.
- **Compliance guardrails:** COPPA/FERPA-aware guest handling, age-assurance for under-13 guest play, and
  data-minimised transient guest identities.

## 3. Non-Goals

- The scoring model (IQ.5) or report privacy toggles (IQ.5/IQ.7) — IQ.9 references, doesn't redefine them.
- General platform moderation infrastructure — IQ.9 wires Live Quizzes into the existing `contentfilter` and
  compliance engines rather than building new ones.
- Proctoring-grade anti-cheat (lockdown browser, webcam) — out of scope; the app's `lockdownMode` remains for
  formal exams, not party games.

## 4. Personas & User Stories

- **As a teacher**, I never want an offensive nickname to appear on the projector, so the game stays safe.
- **As a teacher**, I want to kick or rename a disruptive player instantly, so I keep control.
- **As a teacher**, I want it hard for a student to cheat by opening multiple tabs, so scores are fair.
- **As a blind student**, I want to play entirely by keyboard and screen reader, so I'm included.
- **As a colour-blind student**, I want answers I can tell apart without relying on colour.
- **As a student with photosensitive epilepsy**, I need a game with no flashing, so it's safe for me.
- **As a compliance officer**, I need guest play by minors to meet COPPA/age-assurance rules, so we stay legal.

## 5. Functional Requirements

- **FR-1.** Nicknames MUST be validated on join through `contentfilter` (profanity/PII/impersonation lists);
  denied names are rejected with a friendly retry; the filter list is configurable per platform/org.
- **FR-2.** The host MUST have server-enforced controls: **kick** (remove + block re-join for the game),
  **rename** (force a neutral name), **mute names** (show "Player N" instead of nicknames), **lock lobby** (no
  new joins), and **pause**.
- **FR-3.** Anti-cheat: the engine MUST shuffle answer order per player by default (IQ.2 `answer_shuffle`), and
  MUST enforce **one active player session per enrolled identity** per game (a second concurrent join by the
  same user either takes over or is refused — configurable — never both scoring).
- **FR-4.** The system MUST rate-limit and cap **joins per IP/device** and per game to blunt bulk fake joins,
  and MUST distinguish a legitimate **reconnect** (same player token) from a new join (no false positives).
- **FR-5.** The system SHOULD surface simple **integrity signals** to the host post-game (e.g. improbably fast
  answers across the board, duplicate device fingerprints) as advisory flags — never auto-punitive.
- **FR-6.** **Accessibility (MUST, AA):** every answer is distinguishable by **shape + text label** (colour is
  redundant); all host/player controls are keyboard-operable with visible focus; countdown and results use
  ARIA live regions; a **reduced-motion** setting removes animations; a **no-flashing** guarantee (no content
  flashes > 3×/s) holds on host, projector, and player.
- **FR-7.** Question/answer **media** MUST support captions/alt (enforced at authoring by IQ.2); the projector
  MUST offer a high-contrast, large-text preset.
- **FR-8.** **Guest play** MUST be off unless explicitly enabled per game and per platform/org policy; when on,
  guest identities are transient (nickname + per-game token only), data-minimised, and excluded from the
  gradebook/learner model (IQ.7 FR-10).
- **FR-9.** For **under-13** contexts, guest join MUST honour the age-assurance/design-code guardrails in
  [S08](../standards/S08-childrens-privacy-age-assurance-design-codes.md) (no open public join for children;
  teacher-mediated only), and default to enrolled-only.
- **FR-10.** All safety actions (kick/ban/rename/mute) and denied nicknames MUST be **audited** and available
  in the game report/admin logs.
- **FR-11.** A **report/abuse** affordance MUST let a host flag content (e.g. an offensive word-cloud/open-text
  answer) for review; open-text answers (type-answer, word-cloud) MUST pass moderation before display on the
  projector.

## 6. Non-Functional Requirements

- **Performance** — nickname/open-text moderation check < 50 ms (cached lists); no perceptible join delay.
- **Security** — all safety controls server-enforced (client cannot bypass a kick/ban/mute); one-session rule
  is server-authoritative; join limits resist trivial IP rotation within reason.
- **Privacy & Compliance** — guest data minimised and short-lived; COPPA/FERPA honoured; device signals used
  only for integrity flags, coarse, and not persisted as fingerprints beyond the game (privacy-preserving).
- **Accessibility** — WCAG 2.1 AA verified by automated (axe) + manual screen-reader + colour-blind + reduced-
  motion + photosensitivity (PEAT-style) checks; this is the story that *owns* the AA sign-off for the section.
- **Scalability** — moderation lists cached; checks O(1); host controls scale to 200-player rooms.
- **Reliability** — safety actions are durable (survive reconnect); a kicked player cannot silently rejoin.
- **Observability** — counters: denied nicknames, kicks/bans, blocked joins, integrity flags; alerts on
  moderation-service failure (fail closed for open-text on the projector).
- **Maintainability** — moderation and a11y are cross-cutting utilities reused by all IQ surfaces.
- **Internationalization** — moderation lists per language; a11y announcements localised; RTL-safe.
- **Backward compatibility** — additive; defaults are the safe/strict settings.

## 7. Acceptance Criteria

- **AC-1.** *Given* a player enters an offensive nickname, *when* they join, *then* it's rejected and never
  shown; the attempt is audited.
- **AC-2.** *Given* a disruptive player, *when* the host kicks them, *then* they're removed and cannot rejoin
  that game (server-enforced), even by reopening the tab.
- **AC-3.** *Given* the host enables "mute names", *when* the projector renders, *then* players show as "Player
  N", not their nicknames.
- **AC-4.** *Given* an enrolled student opens a second tab to double-score, *when* they join again, *then* the
  configured one-session rule prevents a second scoring identity.
- **AC-5.** *Given* a legitimate reconnect (same player token), *when* it happens, *then* it is **not** treated
  as a cheat/new join and the score is preserved.
- **AC-6.** *Given* a colour-blind and a screen-reader user, *when* a question opens, *then* both can identify
  and select answers by shape+label and keyboard, and hear the countdown/result.
- **AC-7.** *Given* reduced-motion / no-flashing settings, *when* transitions play, *then* there is no motion/
  flashing that violates the setting (verified by audit).
- **AC-8.** *Given* an open-text answer with profanity, *when* it would appear in a word cloud, *then* it is
  filtered/withheld from the projector and flaggable.
- **AC-9.** *Given* an under-13 context, *when* guest join is attempted publicly, *then* it is blocked and only
  teacher-mediated enrolled play is allowed.

## 8. Data Model

Migration `398_interactive_quizzes_safety.sql`:

```sql
ALTER TABLE quizgame.sessions
  ADD COLUMN IF NOT EXISTS allow_guests     BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS lobby_locked     BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS names_muted      BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS one_session_rule TEXT NOT NULL DEFAULT 'takeover', -- takeover | refuse | off
  ADD COLUMN IF NOT EXISTS max_joins_per_ip INTEGER NOT NULL DEFAULT 5;

ALTER TABLE quizgame.session_players
  ADD COLUMN IF NOT EXISTS banned      BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS renamed_by_host BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS join_ip_hash TEXT;         -- salted, transient integrity signal only

CREATE TABLE quizgame.safety_events (
  id         BIGSERIAL PRIMARY KEY,
  session_id UUID NOT NULL REFERENCES quizgame.sessions (id) ON DELETE CASCADE,
  player_id  UUID REFERENCES quizgame.session_players (id) ON DELETE SET NULL,
  actor_id   UUID REFERENCES "user".users (id) ON DELETE SET NULL, -- host/system
  kind       TEXT NOT NULL,     -- nickname_denied | kicked | banned | renamed | muted | integrity_flag | content_flag
  detail     JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_quizgame_safety_session ON quizgame.safety_events (session_id, created_at);
```

- `join_ip_hash` is salted per game and purged with the session (integrity signal, not durable fingerprint).
- Moderation word lists come from `contentfilter` (platform/org-configurable), not stored here.

## 9. API Surface

- **Join (IQ.4) MUST call moderation:** `POST .../join` runs nickname through `contentfilter` and enforces
  join limits + one-session rule before minting a player.
- **Host controls (WS + REST):** `{type:"kick"|"ban"|"rename"|"mute_names"|"lock_lobby"|"pause"}` frames, and
  `POST /live-quizzes/games/{game_id}/players/{player_id}/{kick|ban|rename}`.
- **Open-text moderation:** server filters `type_answer`/`word_cloud` submissions before including them in the
  projector distribution; flagged items go to `safety_events`.
- **Report/abuse:** `POST /live-quizzes/games/{game_id}/flag` `{playerId?, questionIndex?, reason}`.
- **Settings:** game creation accepts `{allowGuests, oneSessionRule, maxJoinsPerIp, ...}`.
- **OpenAPI:** document safety settings, host-control frames, and moderation behaviour.

## 10. UI / UX

- **Host safety panel:** per-player kick/ban/rename; toggles for mute-names, lock-lobby; a small "integrity
  flags" advisory list post-game.
- **Join UX (IQ.4):** inline nickname rejection with guidance; guest-allowed only when enabled.
- **Player a11y layer:** shape+label answer tiles, keyboard shortcuts, ARIA-live countdown/results, a
  per-user reduced-motion toggle (honouring OS `prefers-reduced-motion`), no-flashing transitions.
- **Projector a11y preset:** high-contrast, large text, no flashing; open-text answers shown only after
  moderation.
- **States:** nickname-denied, kicked/banned, lobby-locked, names-muted, content-withheld.
- **Accessibility:** this story's UI *is* the AA reference for the section; every state has a screen-reader
  script.
- **Copy & i18n:** `liveQuiz.safety.*`, `liveQuiz.a11y.*`, `liveQuiz.moderation.*`.

## 11. AI / ML Considerations

Optional: an AI toxicity classifier could augment list-based `contentfilter` for open-text answers (reuse the
AP provider path with PII redaction and a strict cost budget); the list-based filter is the required baseline
and the fail-closed default if the classifier is unavailable.

## 12. Integration Points

- **Reuse:** `contentfilter` (nickname + open-text moderation), the compliance engines
  ([S01](../standards/S01-unified-data-subject-rights-orchestration.md)/[S02](../standards/S02-data-retention-deletion-engine.md)/[S08](../standards/S08-childrens-privacy-age-assurance-design-codes.md)),
  the design system's a11y primitives, audit logging, rate limiter.
- **Server new:** moderation hooks in `quizgame_join.go` + WS, `repos/quizgame/safety.go`, integrity-signal
  computation in the report job (IQ.7).
- **Web new:** host safety panel, player a11y layer, projector a11y preset.

## 13. Dependencies & Sequencing

- Must ship after: IQ.3, IQ.4 (there must be a game/join to protect).
- Must ship **before**: enabling guest join (IQ.4) and any public catalog (IQ.8) — hard gate.
- Shared infra: content filter, compliance engines, rate limiter, audit.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Offensive nickname/open-text reaches the projector | M | H | List-based moderation on join + before display; fail-closed; host mute-names; audit |
| Anti-cheat false-positives punish honest reconnects | M | H | Player-token reconnect path distinct from new join; advisory-only integrity flags |
| A11y regressions ship silently | M | H | axe in CI + manual SR/colour-blind/reduced-motion/PEAT checks gate release |
| Guest join exposes minors (COPPA) | M | H | Guest off by default; under-13 → enrolled-only + S08 age-assurance |
| Device fingerprinting overreach | M | M | Salted per-game IP hash, purged with session; coarse signals only |
| Moderation service outage | L | M | Fail closed for open-text on projector; nickname list is local/cached |

## 15. Rollout Plan

- **Flag:** safety controls are always-on within `interactive_quizzes_enabled`; **guest-join** and **public
  catalog** stay disabled until this story ships and passes audit.
- **Sequencing:** migration `398` → moderation on join + host controls + a11y layer → a11y audit → enable
  guest-join sub-flag.
- **Dogfood:** run a game attempting offensive names, multi-tab farming, and a full screen-reader play.
- **GA criteria:** AC-1..AC-9 pass; WCAG 2.1 AA audit signed off; anti-cheat false-positive rate ~0 on
  reconnect tests.
- **Rollback:** disable guest-join; safety controls remain (they harden, not gate, enrolled play).

## 16. Test Plan

- **Unit** — nickname/open-text moderation; one-session rule; join-limit vs reconnect discrimination; kick/ban
  durability.
- **Integration** — kick prevents rejoin; mute-names on projector; open-text withheld until moderated; guest
  gating by policy/age.
- **End-to-end** — Playwright: offensive nickname rejected; host kicks + player can't rejoin; multi-tab
  blocked; reconnect preserved.
- **Security** — client cannot bypass kick/ban/mute; one-session server-enforced; join-limit abuse.
- **Accessibility** — axe on all states; screen-reader full play; colour-blind simulation; reduced-motion;
  PEAT/photosensitivity check.
- **Compliance** — COPPA guest gating; DSAR export includes guest-excluded rationale; retention of safety
  events.
- **Manual** — teacher runs the "chaos" scenario (names, tabs, disruption) and regains control.

## 17. Documentation & Training

- Instructor: "Keep your game safe" (mute names, kick, lock lobby), accessibility features for students.
- Admin: configuring moderation lists; guest-join policy; COPPA/age settings.
- API reference: safety settings + host-control frames.
- Runbook: moderation fail-closed behaviour; integrity-flag interpretation (advisory only); audit locations.

## 18. Open Questions

1. One-session default — `takeover` or `refuse`? (Recommendation: `takeover` for classroom reconnect UX;
   `refuse` where strict integrity is required.)
2. Do we build the AI toxicity classifier now or ship list-based only? (Recommendation: list-based baseline at
   GA; AI augmentation as a follow-up with cost budget.)
3. Should integrity flags ever gate a score automatically? (Recommendation: no — advisory to the host only, to
   avoid unfair automated penalties.)

## 19. References

- Existing files: `server/internal/repos/contentfilter/`, `server/internal/config/config.go` (moderation
  config), the design system's accessibility primitives, `server/internal/repos/adminaudit/`.
- Related plans: [IQ.3](../../completed/interactive-quizzes/IQ.3-live-game-hosting-engine.md), [IQ.4](../../completed/interactive-quizzes/IQ.4-player-join-and-gameplay.md),
  [IQ.7](IQ.7-reports-results-gradebook.md), [IQ.8](IQ.8-library-templates-sharing.md);
  [S08 children's privacy](../standards/S08-childrens-privacy-age-assurance-design-codes.md),
  [S02 retention](../standards/S02-data-retention-deletion-engine.md).
