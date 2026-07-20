# SS.1 — Full-Screen Sharing (Cableless Classroom Presentation)

> Implementation plan. **Status: DONE** (moved from `docs/plan/screenshare/`). Source: net-new capability
> (in-person classroom presentation friction). Landscape: [screenshare/README](README.md). Reuses the
> WebSocket auth/room *transport* proven by Collaborative Documents (`collab_docs_ws.go`, `yrelay`) and the
> live-game room pattern
> ([IQ.3](../interactive-quizzes/IQ.3-live-game-hosting-engine.md)) for **signaling**, but adds a net-new
> **WebRTC media plane** (in-house Pion SFU + TURN). Deliberately **narrower** than
> [6.4 Virtual Classroom](../06-communication-collaboration/6.4-virtual-classroom.md)
> (external A/V providers): SS.1 is one presenter's *entire screen* → the room, sub-second, no provider.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | SS.1 |
| **Section** | Screen Sharing (Cableless Classroom Presentation) |
| **Severity** | MAJOR |
| **Markets** | K12 / HE (SL: study-group use only) |
| **Status (today)** | DONE |
| **Estimated effort** | L (1–2mo) |
| **Owner (proposed)** | Realtime squad + Web + Platform/Infra (TURN) |
| **Depends on** | — (reuses shipped WS transport; no hard blocker) |
| **Unblocks** | SS.M1 (mobile capture), SS.2 (recording/VOD) |

---

## 1. Problem Statement

In most in-person classes today a student who wants to present must physically plug a laptop into the room's
projector/monitor — the wrong dongle, a dead HDMI port, a resolution mismatch, or a blank screen eats the
first five minutes of every session, and only one machine can be connected at a time. There is no way inside
Lextures to put a screen on the classroom display. SS.1 lets any authorized participant share their **entire
screen** straight from the browser to the teacher's display (already driving the projector) and, optionally,
to every student's own device — one click, no cable, instructor in control. The outcome: presentations start
in seconds, hand-offs between presenters are instant, and "it won't project" stops being a class-time tax.

## 2. Goals

- Let a student or instructor **share their entire screen** from a supported desktop browser to the
  classroom display and/or classmates' devices in **one click**, with no cable, dongle, or native install.
- Achieve **sub-second, classroom-scale** delivery: one presenter fanned out to 30+ simultaneous viewers via
  an in-house **SFU** (presenter uploads once; the server forwards), not a peer mesh.
- Keep the **instructor in control**: who may present, single active presenter by default, approve/start/stop
  any share, and instant hand-off between presenters.
- Make sharing **safe and consensual**: an unmissable "you are sharing your entire screen" indicator, a
  one-click Stop, and no server-side recording by default.
- Ship **self-hostable and provider-free** in the media critical path (in-house SFU + self-hosted TURN),
  behind a platform + per-course feature flag, reusing the existing WS/auth/telemetry stack.

## 3. Non-Goals

- **Full multi-party video conferencing** — webcam grids, breakout rooms, raise-hand, gallery view. That is
  [6.4 Virtual Classroom](../06-communication-collaboration/6.4-virtual-classroom.md) (external
  providers). SS.1 shares *screens*, not faces.
- **Mobile-originated entire-screen capture** (iOS ReplayKit broadcast extension, Android MediaProjection) →
  carved out as **SS.M1**. SS.1 web viewers work on mobile browsers, but capture originates on desktop.
- **Recording / VOD / replay** of a shared session → **SS.2**. SS.1 exposes a recording *hook* but persists
  **no media** by default.
- **Remote control / annotation** over the shared screen (mouse takeover, laser pointer, ink) — future.
- **Region/window/tab-only sharing as the headline path.** The feature is *share the entire screen*; window-
  or tab-scoped capture is allowed as a graceful fallback but is not the promoted flow.
- **Replacing native OS casting** (AirPlay / Miracast / Chromecast) — those remain available and untouched.

## 4. Personas & User Stories

- **As a student presenter**, I want to click "Share my screen," pick my whole display, and have it appear on
  the class projector instantly — no cable, no dongle, no asking the teacher to switch inputs.
- **As an instructor (host)**, I want to decide who can present, hand the screen from one student to the next
  in one click, and stop any share immediately if something inappropriate appears.
- **As an audience student**, I want to watch the shared screen on my own laptop/phone (readably, full-screen
  if I want) when I'm at the back of the room or joining from an overflow space.
- **As the classroom display / projector** (a browser tab the teacher opens once on the room PC), I want a
  clean, full-bleed, chrome-free view of whoever is presenting, that survives presenter hand-offs.
- **As a school admin**, I want to enable this per course, know that nothing is recorded by default, and
  deploy the TURN server our network requires — self-hosted, no student media leaving our infra.
- **As a self-learner**, I want to share my screen to a peer study group I've invited (same mechanism, no
  classroom).

## 5. Functional Requirements

Numbered, testable, RFC 2119 language.

- **FR-1.** `POST /api/v1/courses/{course_code}/screen-share/sessions` MUST create a `screenshare.sessions`
  row (status `open`) for the course when the platform flag **and** the course flag are on and the caller has
  host permission (instructor/TA) or the course allows student-initiated shares; it MUST return
  `{sessionId, joinToken, turn}` where `turn` is a set of **ephemeral** ICE (STUN/TURN) credentials (FR-11).
- **FR-2.** The system MUST expose a signaling WebSocket `GET /api/v1/courses/{course_code}/screen-share/sessions/{session_id}/ws`
  that upgrades and authenticates via the **existing handshake** (first text frame `{"authToken":…}` →
  `JWTSigner`), then verifies enrollment/role (`enrollment.UserHasAccess`) and joins the caller to the
  session's in-process **signaling room** with a role of `host`, `presenter`, `viewer`, or `display`.
- **FR-3.** The server MUST run an in-house **SFU** (Selective Forwarding Unit): the active presenter
  publishes **one** upstream WebRTC track (screen video + optional audio); the SFU forwards RTP to every
  subscribed viewer/display. Clients MUST NOT mesh (no presenter→N direct uploads).
- **FR-4.** The presenter client MUST capture the screen via `navigator.mediaDevices.getDisplayMedia`,
  requesting `{ video: { displaySurface: "monitor" }, audio: <optional> }`; on the returned track the client
  MUST inspect `getSettings().displaySurface` and, when it is not `"monitor"` (browser gave a window/tab),
  surface a "You're sharing a window/tab, not your whole screen — continue or reshare?" prompt (FR-14).
- **FR-5.** The media plane MUST be end-to-end encrypted in transit via **DTLS-SRTP** (WebRTC default);
  signaling MUST be over WSS. The SFU MUST negotiate via SDP offer/answer relayed over the signaling WS
  (`offer`, `answer`, `ice-candidate` frames) with Trickle ICE.
- **FR-6.** The system MUST enforce a **single active presenter** per session by default. A `presenter-request`
  from a non-host MUST be queued and require host `presenter-grant`; a host MAY grant/revoke, and MAY set the
  session to `free_for_all` (any enrolled user may self-promote) or `host_only`.
- **FR-7.** The host MUST be able to **stop** the active presenter's share and **hand off** to the next
  presenter without tearing down the session; on hand-off the SFU MUST swap the forwarded upstream track and
  notify all subscribers (`presenter-changed`), with viewers re-subscribing to the new track.
- **FR-8.** While a client is publishing, that client MUST display an **always-visible sharing indicator**
  ("You are sharing your ENTIRE screen") and a one-click **Stop sharing** control; stopping (or the browser's
  native stop-sharing UI, detected via the track's `ended` event) MUST propagate `presenter-stop` to the SFU
  and room within 500 ms.
- **FR-9.** The SFU MUST support **bandwidth adaptation** so a weak viewer link degrades gracefully: the
  presenter SHOULD publish **simulcast** (or the SFU MUST apply per-subscriber quality selection / PLI/FIR
  keyframe requests) so one slow viewer never stalls the presenter or the room.
- **FR-10.** The system MUST tolerate **NAT/firewall** environments: every peer MUST be offered STUN **and**
  TURN (UDP + TCP/TLS 443 fallback) so school networks that block UDP still relay; a peer that cannot make a
  direct/`srflx` candidate MUST fall back to `relay` via TURN.
- **FR-11.** TURN credentials MUST be **short-lived and per-user**, minted server-side using the coturn REST
  time-limited-credential scheme (HMAC of `expiry:userId` with a shared secret); they MUST expire (default
  ≤ 12 h) and MUST NOT be reusable to relay arbitrary traffic beyond their TTL.
- **FR-12.** The system MUST tolerate **reconnection**: a viewer or presenter whose WS drops MUST be able to
  rejoin the same session with its join token, run **ICE restart**, and resume; the host MUST be able to
  reopen the console and see the current presenter and participant list (DB + room state are the truth).
- **FR-13.** A **display** role (the projector tab) MUST render full-bleed with no answer/host controls, MUST
  auto-follow presenter hand-offs, and MUST auto-reconnect on flaps so an unattended room PC recovers without
  a human touching it.
- **FR-14.** The system MUST obtain **explicit consent** before capture (the browser's own picker is the
  consent gate) and MUST NOT auto-start capture; K12 orgs MAY restrict student-initiated sharing to
  `host_only` via course policy (FR-6).
- **FR-15.** Starting/joining MUST be **refused** with a clear error when the platform flag or course flag is
  off, the caller lacks enrollment, the session is `ended`, or the viewer cap (NFR Scalability) is exceeded.
- **FR-16.** The server MUST persist **session and participant metadata and an append-only event log**
  (`screenshare.sessions`, `.participants`, `.events`) for audit and observability, but MUST persist **no
  media frames** by default (recording is SS.2, gated by its own flag + consent).
- **FR-17.** The host MUST be able to **end** the session (`POST …/sessions/{id}/end`), which stops any active
  share, closes all peer connections, marks the session `ended`, and expires the join token.

## 6. Non-Functional Requirements

- **Performance** — Glass-to-glass latency p95 **< 1 s** on same-LAN/typical school Wi-Fi (target < 500 ms
  direct, < 1.5 s over TURN relay). Signaling join → first video frame p95 **< 3 s**. Support ≥ **30 viewers**
  per session and many concurrent sessions per instance; a single presenter uploads once (SFU fan-out), so
  presenter uplink is bounded regardless of viewer count.
- **Security** — Signaling token-verified + role-checked; media DTLS-SRTP encrypted; TURN credentials
  ephemeral/HMAC-scoped (FR-11); only enrolled users may subscribe; a viewer can never *become* presenter
  without a host grant (FR-6); presenter identity is server-stamped from the JWT, never client-claimed. Threat
  model notes: TURN relay-abuse (mitigated by TTL + per-user creds + `no-multicast`), signaling flooding
  (per-conn rate/size caps as in the WS relay), and unauthorized subscribe (enrollment check on join).
- **Privacy & Compliance** — Sharing an *entire* screen can expose notifications, tabs, or another student's
  PII → **FERPA** relevant. Mitigations: unmissable sharing indicator (FR-8), one-click stop, host kill
  (FR-7), **no recording by default** (FR-16), and course policy to force `host_only` for minors (**COPPA**/
  K12). Data-subject export/deletion of session *metadata* rides
  [S01](../../plan/standards/S01-unified-data-subject-rights-orchestration.md)/[S02](../../plan/standards/S02-data-retention-deletion-engine.md);
  no media is stored to export. Consent is captured by the browser picker and logged as an event.
- **Accessibility (WCAG 2.1 AA)** — Screen pixels can't be read by assistive tech, so the UI MUST announce
  state via ARIA live regions ("Ada is now presenting", "Sharing stopped"); the viewer MUST offer keyboard-
  operable Play/Pause/Fullscreen/Mute controls with visible focus; the sharing indicator MUST not rely on
  colour alone; no flashing (photosensitivity) on state transitions; reduced-motion honored on the display
  view. Presenter **audio**, when shared, satisfies more of the room than video alone.
- **Scalability** — SFU rooms are single-process (mirrors collab-doc/IQ.3 room model). Horizontal path:
  **sticky routing by `session_id`** for GA; cross-instance signaling bridged via **Redis pub/sub or
  RabbitMQ** (both already in the stack) and, when a session outgrows one node, an SFU-to-SFU relay
  (cascade). Default **viewer cap 50/session** (configurable) to bound fan-out.
- **Reliability** — DB + in-process room are the durable/authoritative signaling state; media is ephemeral by
  design. ICE restart on network change; presenter drop pauses the room (viewers see "presenter reconnecting")
  with a grace window before auto-stop; display tab auto-reconnects; session auto-ends and finalizes on
  abandonment via a background reaper. Idempotent presenter-grant (a repeated grant is a no-op).
- **Observability** — Gauges: `screenshare_active_sessions`, `viewers_per_session`, `presenter_connected`,
  `turn_relay_sessions`. Counters: `ice_connection_failures`, `turn_allocations`, `reconnects`,
  `presenter_handoffs`, `subscribe_denied`. Histograms: `join_to_first_frame_ms`, peer RTT, presenter/viewer
  bitrate. Traces around session create, WS join, SDP negotiate, presenter change. Alert on ICE-failure rate
  and TURN saturation.
- **Maintainability** — Extract shared WS helpers already used by collab docs / IQ into the reusable room
  package (`yrelay`); keep the **SFU as its own package** (`server/internal/screenshare/sfu`, Pion-based) with
  the presenter-arbitration **state machine as a pure, unit-testable reducer** separate from transport.
- **Internationalization** — Signaling frames carry structured data (ids/enums), never localized strings; all
  UI copy externalized under `screenShare.*`; presenter display names respect locale/RTL.
- **Backward compatibility** — Additive: new tables, new flag columns, new routes, new go deps (Pion). No
  change to existing WS, collab, or quiz behaviour; flags default **off**.

## 7. Acceptance Criteria

Given/When/Then; each maps to ≥ 1 automated test.

- **AC-1.** *Given* the platform + course flags are on and a presenter clicks "Share screen" and selects their
  whole monitor, *when* negotiation completes, *then* the projector (`display` role) shows their live screen
  and viewers see it within p95 < 3 s of join.
- **AC-2.** *Given* a presenter selects a single window/tab instead of the whole screen, *when* the client
  reads `displaySurface !== "monitor"`, *then* it warns them it isn't the entire screen and offers to reshare
  before publishing.
- **AC-3.** *Given* a share is live, *when* the presenter clicks Stop (or the browser's native stop), *then*
  all viewers and the display stop within 500 ms and see "Sharing ended," and no media persists anywhere.
- **AC-4.** *Given* a school network that blocks UDP, *when* a viewer joins, *then* the connection succeeds via
  a TURN **relay** candidate over TCP/TLS 443 and video plays.
- **AC-5.** *Given* two students both request to present, *when* the host grants the second, *then* the first
  is stopped, the display and all viewers follow the hand-off with no session teardown, and only one upstream
  track is ever forwarded.
- **AC-6.** *Given* a viewer's Wi-Fi drops mid-session, *when* it reconnects, *then* it rejoins the same
  session (ICE restart) and resumes the live view without a full page reload.
- **AC-7.** *Given* 30 viewers subscribed and one on a throttled link, *when* that viewer's bandwidth drops,
  *then* only that viewer degrades (lower quality) while the presenter and the other 29 are unaffected.
- **AC-8.** *Given* the course flag is off (or the caller is not enrolled), *when* create/join is attempted,
  *then* it is refused with a clear, non-leaking error.
- **AC-9.** *Given* a K12 course set to `host_only`, *when* a student attempts to self-promote to presenter,
  *then* it is denied and only host-granted presenters can share.
- **AC-10.** *Given* the projector tab is left open unattended and the room Wi-Fi flaps, *when* the network
  recovers, *then* the display auto-reconnects and re-follows the current presenter with no human input.
- **AC-11.** *Given* the host ends the session, *when* end is confirmed, *then* every peer connection closes,
  the join token is invalid, and a new join is refused.

## 8. Data Model

Migration `430_screen_share_sessions.sql` (+ `430_..._down.sql`), plus a per-course flag column and the
platform flag. **No media is stored** — only session/participant/event metadata.

```sql
CREATE SCHEMA IF NOT EXISTS screenshare;

CREATE TYPE screenshare.session_status AS ENUM ('open','presenting','ended','abandoned');
CREATE TYPE screenshare.present_policy AS ENUM ('host_only','request','free_for_all');
CREATE TYPE screenshare.participant_role AS ENUM ('host','presenter','viewer','display');

CREATE TABLE screenshare.sessions (
  id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  course_id      UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
  host_id        UUID REFERENCES "user".users (id) ON DELETE SET NULL,
  title          TEXT,
  status         screenshare.session_status NOT NULL DEFAULT 'open',
  policy         screenshare.present_policy NOT NULL DEFAULT 'request',
  present_audio  BOOLEAN NOT NULL DEFAULT FALSE,   -- allow presenter to share system/tab audio
  viewer_cap     INTEGER NOT NULL DEFAULT 50,
  active_presenter_id UUID REFERENCES "user".users (id) ON DELETE SET NULL,
  settings       JSONB NOT NULL DEFAULT '{}'::jsonb,
  join_token_hash TEXT NOT NULL,                   -- hashed; raw returned once at create (FR-1)
  started_at     TIMESTAMPTZ,
  ended_at       TIMESTAMPTZ,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_screenshare_sessions_course ON screenshare.sessions (course_id, created_at DESC);
CREATE INDEX idx_screenshare_sessions_active
  ON screenshare.sessions (course_id) WHERE status IN ('open','presenting');

CREATE TABLE screenshare.participants (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  session_id   UUID NOT NULL REFERENCES screenshare.sessions (id) ON DELETE CASCADE,
  user_id      UUID REFERENCES "user".users (id) ON DELETE SET NULL,  -- NULL only for anon display links
  role         screenshare.participant_role NOT NULL,
  connected    BOOLEAN NOT NULL DEFAULT TRUE,
  joined_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  left_at      TIMESTAMPTZ,
  UNIQUE (session_id, user_id, role)
);
CREATE INDEX idx_screenshare_participants_session ON screenshare.participants (session_id);

CREATE TABLE screenshare.events (
  id         BIGSERIAL PRIMARY KEY,
  session_id UUID NOT NULL REFERENCES screenshare.sessions (id) ON DELETE CASCADE,
  seq        INTEGER NOT NULL,
  type       TEXT NOT NULL,   -- session_open, join, present_request, present_grant, present_stop,
                              -- present_change, consent_given, ice_failed, reconnect, session_end
  actor_id   UUID REFERENCES "user".users (id) ON DELETE SET NULL,
  payload    JSONB NOT NULL DEFAULT '{}'::jsonb,   -- never raw media; e.g. {displaySurface:"monitor"}
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (session_id, seq)
);
CREATE INDEX idx_screenshare_events_session ON screenshare.events (session_id, seq);

-- Per-course feature flag (matches interactive_quizzes_enabled / visual_boards_enabled pattern).
ALTER TABLE course.courses ADD COLUMN IF NOT EXISTS screen_share_enabled BOOLEAN NOT NULL DEFAULT FALSE;
```

- **Platform flag:** add `ScreenShareEnabled` (default **false**) to `platformconfig` (`features.go`,
  `applyPlatformBools`) and its settings column, exactly like `InteractiveQuizzesEnabled`.
- **Consent as data:** the `consent_given` event records `{displaySurface}` returned by the picker — the
  audit trail that the user chose to share, without storing what was shared.
- **Retention/backfill:** ended sessions retained for audit, then aged per S02; **no rows to backfill** (new
  feature); the new course column defaults off so existing courses are unaffected.
- **Idempotency:** `events.(session_id, seq)` and `participants.(session_id, user_id, role)` unique
  constraints make re-delivered joins/grants safe.

## 9. API Surface

| Verb | Path | Auth scope |
|---|---|---|
| POST | `/api/v1/courses/{course_code}/screen-share/sessions` | host perm, or enrolled if course `policy` allows student start → `{sessionId, joinToken, turn}` |
| GET | `/api/v1/courses/{course_code}/screen-share/sessions/{id}` | enrolled |
| POST | `/api/v1/courses/{course_code}/screen-share/sessions/{id}/end` | host |
| POST | `/api/v1/courses/{course_code}/screen-share/sessions/{id}/presenter` | host — `{action:"grant"\|"revoke", userId}` |
| POST | `/api/v1/courses/{course_code}/screen-share/sessions/{id}/turn` | enrolled → fresh ephemeral ICE creds (FR-11) |
| GET (WS) | `/api/v1/courses/{course_code}/screen-share/sessions/{id}/ws` | first-frame `authToken` → JWT + enrollment/role |

- **ICE credential response** (`turn` / `/turn`): `{ iceServers: [{urls:["stun:…:3478"]}, {urls:["turn:…:3478?transport=udp","turn:…:443?transport=tcp","turns:…:5349"], username:"<expiry>:<userId>", credential:"<hmac>"}], ttlSeconds }`.
- **Signaling WS frames (JSON, text):**
  - client→server: `{type:"join", role}`, `{type:"offer"|"answer", sdp}`, `{type:"ice-candidate", candidate}`,
    `{type:"present-request"}`, `{type:"present-stop"}`, `{type:"quality", preferred}`, `{type:"ping"}`.
  - server→client: `{type:"joined", selfRole, participants}`, `{type:"offer"|"answer", sdp}` (SFU negotiation),
    `{type:"ice-candidate", candidate}`, `{type:"present-changed", presenterId|null}`,
    `{type:"present-grant"|"present-revoke"}`, `{type:"participant", op:"add"|"remove", …}`,
    `{type:"error", code, message}`. Every server frame carries a monotonically increasing `seq` for gap
    detection/replay on reconnect (as in IQ.3).
- **Roles at join:** `host` (course permission), `presenter` (host-granted or self per policy), `viewer`
  (enrolled), `display` (read-only projector; may use a signed, course-scoped display token so the room PC
  needn't log in a person). No role can subscribe without passing the enrollment/token check.
- **Rate-limit / quota:** session-create and `/turn` are per-user rate-limited (reuse `ratelimit`); signaling
  frames are size/rate-capped per connection (reuse the WS relay caps); viewer cap enforced at join (FR-15).
- **OpenAPI:** document REST endpoints + the WS handshake and frame catalogue + the ICE-credential shape (as
  collab docs / IQ.3 are documented).

## 10. UI / UX

- **Presenter flow (web):** on a course's Screen Share panel, "Share my screen" →
  `getDisplayMedia({video:{displaySurface:"monitor"}})` opens the browser picker (the consent gate) → on the
  returned track, verify it's the whole monitor (else AC-2 prompt) → negotiate with SFU → a persistent
  **sharing bar** appears ("You are sharing your **entire screen** to {course}") with a red **Stop sharing**
  button; the track's `ended` event (native stop) also tears down.
- **Display / projector view** `clients/web/src/pages/lms/screen-share-present-page.tsx`: full-bleed, chrome-
  free `<video autoplay muted playsinline>`, a small "{name} is presenting" pill, "Waiting for a presenter…"
  empty state, and silent auto-reconnect. Opened once on the room PC; a QR/short join hint shows in the empty
  state so students can also view on their devices.
- **Instructor console** (course page section): current presenter, request queue with grant/deny, one-click
  Stop, presenter-policy selector (`host_only` / `request` / `free_for_all`), audio-allowed toggle, End
  session. Live participant/viewer count.
- **Audience viewer** (embedded on the course page and mobile web): the video with keyboard-operable
  Play/Pause/Mute/Fullscreen, a connection-quality chip, and a "presenter changed" toast.
- **States:** no-session / start; connecting; **waiting-for-approval** (present-request pending);
  presenting; viewer-watching; **reconnecting** (banner + backoff); presenter-paused ("presenter reconnecting
  …"); stopped/ended; **error** (flag off, not enrolled, cap reached, unsupported browser). Unsupported-
  browser state explains that entire-screen capture needs a desktop Chromium/Firefox/Safari and offers the
  audience view instead.
- **Mobile / responsive:** viewing works on mobile web (the video is responsive, fullscreen-capable);
  *capturing* is desktop-only in SS.1 (mobile capture = SS.M1) — the Share button is hidden/disabled with an
  explanation on unsupported platforms.
- **Accessibility annotations:** ARIA live announcements for present/stop/hand-off; visible focus + keyboard
  operation of all controls; sharing indicator uses icon + text (not colour alone); reduced-motion on the
  display; no flashing on transitions; the "you are sharing" bar is an `role="status"` region.
- **Copy & i18n:** `screenShare.start.*`, `screenShare.present.*`, `screenShare.console.*`,
  `screenShare.state.*`, `screenShare.consent.*`, `screenShare.error.*` in `Localizable`/web i18n catalogues.

## 11. AI / ML Considerations

Not AI-touching in v1. *(Future, out of scope: optional on-device OCR / live alt-text of the shared screen to
improve accessibility for blind viewers, and auto-generated session summaries — both would ride the existing
BYOK provider path and are noted only as a later enhancement, not part of SS.1.)*

## 12. Integration Points

- **Reuse:** WS upgrade + first-frame `authToken` handshake and the room registry from
  `server/internal/httpserver/collab_docs_ws.go` + `server/internal/yrelay` (generalized signaling room);
  `JWTSigner`; `enrollment.UserHasAccess` / `UserHasEnrollmentRole`; `course` repo; `ratelimit`;
  `server/internal/telemetry`; the platform + per-course feature-flag plumbing (`platformconfig/features.go`,
  `repos/course/features.go`).
- **New backend deps:** `github.com/pion/webrtc/v4` (SFU media plane) and TURN — either a **self-hosted
  coturn** container (recommended; REST time-limited creds) or `github.com/pion/turn/v4` embedded. Redis /
  RabbitMQ (already present) for cross-instance signaling fan-out.
- **New server code:** `server/internal/screenshare/sfu/` (Pion SFU: publish/subscribe, keyframe/PLI, simulcast
  layer select), `server/internal/screenshare/engine/` (pure presenter-arbitration reducer),
  `server/internal/httpserver/screenshare_ws.go` + `screenshare_sessions.go` + `screenshare_turn.go`,
  `server/internal/repos/screenshare/{sessions,participants,events}.go`, and a background reaper for
  abandoned/idle sessions (`server/internal/background/`).
- **New web code:** `clients/web/src/lib/screen-share-realtime.ts` (WS + `RTCPeerConnection` lifecycle,
  ICE-restart reconnect, `getDisplayMedia` capture), `screen-share-present-page.tsx` (display),
  `screen-share-console.tsx` (host), and a course-page audience/presenter section.
- **Infra:** TURN/STUN service added to `docker-compose.*.yml` and `iac/`/`deploy/` (ports 3478 UDP/TCP,
  5349 TLS, 443 TCP/TLS fallback), with the shared secret wired via config/env.
- **Relationship to existing flags:** distinct from `live_sessions_enabled` (synchronous sessions scheduling)
  and from 6.4's external-provider `virtual_meetings`; SS.1 neither reads nor writes those.

## 13. Dependencies & Sequencing

- **Must ship after:** nothing hard — reuses already-shipped WS transport, auth, and flag plumbing. Requires
  the **TURN/STUN infra** to be provisioned (a same-milestone infra task, not a prior feature).
- **Must ship before:** **SS.M1** (mobile ReplayKit/MediaProjection capture + mobile viewer consume this
  protocol and SFU) and **SS.2** (recording taps the SFU's forwarded track).
- **Shared infra needed:** WebSocket (present), a self-hosted **TURN/STUN** server (new), the Pion SFU
  process (new), Redis or RabbitMQ for multi-instance signaling (present), background job runner (present),
  telemetry (present).

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| School networks block UDP → no media | H | H | Mandatory TURN over TCP/TLS **443** fallback (FR-10); relay-candidate test in CI-adjacent network profiles |
| No self-hosted TURN → NAT traversal fails in prod | M | H | TURN is a **hard dependency**, shipped in compose + `iac/`; readiness check fails without it; document in admin runbook |
| Build-vs-buy: hand-rolled Pion SFU underestimated | M | H | Scope SFU to **one presenter, N viewers** only (no mixing/MCU); simulcast optional-at-first; keep LiveKit (self-hostable, Pion-based) as the documented fallback (Open Q 1) |
| Entire-screen capture exposes private/other-student content (FERPA) | M | H | Unmissable indicator + one-click stop + host kill (FR-7/8); `host_only` policy for minors; **no recording default** (FR-16); consent event logged |
| Single-process SFU won't fan out across instances | M | M | Sticky routing by `session_id` for GA; Redis/RabbitMQ signaling bridge + SFU cascade specified before multi-node |
| Presenter uplink saturates on big screens / 4K | M | M | Cap capture resolution/framerate, simulcast + per-subscriber quality (FR-9); presenter bitrate telemetry + alerts |
| Browser inconsistency (Safari `getDisplayMedia`, no `displaySurface` guarantees) | M | M | Feature-detect; graceful window/tab fallback (AC-2); documented supported-browser matrix; audience-only path on unsupported |
| TURN relay abused as an open proxy | L | H | Short-TTL per-user HMAC creds (FR-11); coturn `no-multicast`, quota, and deny-private-ranges config |
| Thundering herd of 30 renegotiations on hand-off | M | M | SFU swaps upstream, viewers keep their downstream PC and just re-subscribe; single `present-changed` fan-out; load-test hand-offs |

## 15. Rollout Plan

- **Flags:** platform `ScreenShareEnabled` (default off) **and** per-course `screen_share_enabled` (default
  off) — both must be on. An internal `screen_share_turn_ready` readiness gate refuses session-create until a
  TURN endpoint is configured and health-checked.
- **Sequencing:** migration `430` → provision + health-check TURN/STUN → deploy SFU + signaling behind flags
  (off) → enable in one internal course → **dogfood in a real classroom on the real projector** → load test
  30 viewers incl. a UDP-blocked client → GA per org.
- **Dogfood / pilot:** run a genuine class presentation on the room PC's projector; force presenter and
  display Wi-Fi drops; do 3 presenter hand-offs; verify no recording artifacts exist anywhere.
- **GA criteria:** AC-1..AC-11 pass; 30-viewer load test meets latency targets; TURN-relay path verified on a
  UDP-blocked network; reconnect + hand-off scenarios green; admin runbook published.
- **Rollback:** flip `screen_share_enabled` (course) or `ScreenShareEnabled` (platform) off — in-flight
  sessions end gracefully, peers disconnect cleanly, metadata retained; TURN/SFU can be scaled to zero. No
  schema rollback needed (additive; `430_..._down.sql` exists if required).

## 16. Test Plan

- **Unit** — presenter-arbitration reducer (grant/revoke/hand-off/stop, illegal transitions rejected);
  TURN-credential HMAC generation/expiry; `displaySurface` guard logic; join-token hashing; viewer-cap
  enforcement; SFU subscribe/unsubscribe bookkeeping.
- **Integration** — WS signaling handshake + role gating; SFU SDP negotiate publish→subscribe; ICE via a test
  TURN (relay candidate forced); presenter hand-off swaps upstream with one `present-changed`; reconnect/ICE-
  restart resume; flag-off / not-enrolled / cap-exceeded refusals.
- **End-to-end (Playwright, multi-context)** — presenter + display + 2 viewers: share whole screen → all see
  it → hand off to viewer #1 → stop → "ended". Use Chromium fake-capture flags
  (`--use-fake-device-for-media-stream`, `--auto-select-desktop-capture-source=…`,
  `--use-fake-ui-for-media-stream`) so `getDisplayMedia` runs headless; assert frames render (canvas sample).
- **Security** — non-enrolled subscribe denied; TURN cred expiry rejected after TTL; viewer cannot self-
  promote under `host_only` (AC-9); spoofed presenter identity ignored (server-stamped); signaling flood
  rate-limited; TURN not usable to relay private-range traffic.
- **Accessibility** — axe on console/viewer/display; keyboard-only presenter start/stop and viewer controls;
  screen-reader announces present/stop/hand-off; no-flash transitions; reduced-motion honored.
- **Performance / load** — 30-viewer fan-out latency + presenter bitrate stability; one throttled viewer
  degrades alone (AC-7); many concurrent sessions/instance; TURN relay saturation profile.
- **Manual exploratory** — real projector on a real room PC; corporate/school NAT (UDP blocked); presenter
  Safari/Firefox/Chromium matrix; unplug Wi-Fi mid-share; leave display tab unattended through a network flap;
  confirm no media is written to disk/object storage anywhere.

## 17. Documentation & Training

- **End-user (student):** "Present without cables — share your whole screen in one click," incl. the browser
  picker walkthrough, the sharing indicator, and how to stop; troubleshooting ("I only shared a window").
- **Instructor:** "Run screen sharing in class" — open the projector tab, set who can present, hand off
  between students, stop a share, end the session.
- **Admin:** "Enable Screen Sharing" (platform + course flags) and **"Deploy the TURN/STUN server"**
  (ports, TLS/443 fallback, shared secret, quotas, why it's required on school networks).
- **API reference:** REST endpoints + WS frame catalogue + ICE-credential contract + reconnection semantics.
- **Runbook:** SFU room registry + reaper, TURN operations and quota, ICE-failure debugging, multi-instance
  sticky routing + signaling bridge, and documented scaling limits (viewers/session, sessions/instance).

## 18. Open Questions

1. **SFU: build vs. buy.** Hand-rolled **Pion SFU** (chosen for self-host/provider-free ethos and full
   control) vs. self-hosting **LiveKit** (also Pion-based, batteries-included SFU/TURN/simulcast, faster to
   GA) vs. WebRTC **mesh** for tiny rooms only. *Recommendation: in-house Pion SFU scoped to one-presenter/
   N-viewers for GA; keep LiveKit as the documented fallback if SFU effort overruns.*
2. **TURN: coturn vs. `pion/turn`.** *Recommendation: self-hosted **coturn** (mature, REST time-limited creds,
   TLS/443) shipped in compose + `iac/`; `pion/turn` embedded only for local/dev.*
3. **System/tab audio.** Include presenter audio sharing in v1 (per-course `present_audio` toggle) or defer?
   *Recommendation: ship the toggle **off by default**; it's low marginal cost once the media plane exists.*
4. **Entire-screen enforcement.** Browsers can't *force* whole-monitor selection — SS.1 requests+validates and
   warns (AC-2). Do any K12 orgs need a *hard* block on window/tab sharing? *Recommendation: warn-only for GA;
   add a course policy to hard-reject non-`monitor` surfaces if demand appears.*
5. **Display authentication.** Signed course-scoped **display token** (room PC needn't log in a human) vs.
   requiring an instructor login on the projector tab. *Recommendation: signed display token, revocable, so
   unattended room PCs are practical.*
6. **Guest / non-enrolled viewers** (e.g., an observer, a parent night). Enrolled-only for GA, or allow a
   host-issued view link? *Recommendation: enrolled-only for GA; revisit with SS.2.*
7. **Default viewer cap** per session (fan-out cost). *Recommendation: 50, configurable per platform.*
8. **Horizontal fan-out for GA** — sticky single-instance vs. land the Redis/RabbitMQ signaling bridge + SFU
   cascade now. *Recommendation: sticky single-instance for GA; land the bridge before multi-node.*

## 19. References

- **Existing files this work reuses/touches:** `server/internal/httpserver/collab_docs_ws.go` (WS upgrade +
  first-frame `authToken` handshake + room registry), `server/internal/yrelay/room.go` (room/registry to
  generalize for signaling), `server/internal/repos/enrollment/enrollment.go`
  (`UserHasAccess`/`UserHasEnrollmentRole`), `server/internal/repos/platformconfig/features.go` +
  `server/internal/repos/course/features.go` (flag plumbing), `server/internal/telemetry`,
  `server/internal/ratelimit`, `clients/web/src/lib/live-quiz-realtime.ts` (WS-reducer hook pattern),
  `clients/web/src/pages/lms/live-quiz-present-page.tsx` (projector-view pattern). New go dep:
  `github.com/pion/webrtc/v4`.
- **External standards:** W3C **Screen Capture** (`getDisplayMedia`) & **WebRTC 1.0**; **DTLS-SRTP** (RFC
  5763/5764); **ICE** (RFC 8445), **STUN** (RFC 5389), **TURN** (RFC 5766/8656); coturn REST time-limited
  credentials; **WCAG 2.1 AA**; FERPA/COPPA guidance for classroom media.
- **Related plans:** [screenshare/README](README.md);
  [6.4 Virtual Classroom](../06-communication-collaboration/6.4-virtual-classroom.md)
  (complementary external-provider A/V — explicitly not this);
  [IQ.3 Live Game Hosting Engine](../interactive-quizzes/IQ.3-live-game-hosting-engine.md) and
  [VC.4 Realtime Collaboration & Presence](../visual-collaboration/VC.4-realtime-collaboration-and-presence.md)
  (WS transport/room reuse); SS.M1 (mobile capture) and SS.2 (recording) as downstream follow-ons.
