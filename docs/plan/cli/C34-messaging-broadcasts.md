# C34 — Messaging, broadcasts & notifications

> CLI parity plan. Source: `registerCommunicationRoutes` (`communication`), `broadcasts_http.go` (`orgs/{orgId}/broadcasts`), `me/notifications`, `me/notification-preferences`, `me/push-subscriptions`, `me/device-tokens`, `push`. Baseline: `feed` (channels/post/recent).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | C34 |
| **Section** | Communication |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | PARTIAL (course `feed` only) |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Communication / CLI |
| **Depends on** | C40 |
| **Unblocks** | — |

---

## 1. Problem Statement

Beyond course feed channels, the CLI has no access to the inbox/messaging system, org-wide broadcasts/announcements, or notification management. Admins cannot send an org broadcast (e.g. emergency notice) or an instructor an announcement from a script, and users can't manage notification preferences programmatically.

## 2. Goals

- Send and read direct/inbox messages.
- Send org-wide or course broadcasts/announcements from automation.
- Read and manage notifications and notification preferences.

## 3. Non-Goals

- Real-time chat UX.
- Being an email server (server handles delivery).

## 4. Personas & User Stories

- **As an admin**, I want `broadcasts send --org O --file notice.md --audience students`.
- **As an instructor**, I want `announcements post --course C --file update.md` (via feed/communication).
- **As a user**, I want `messages send --to U --subject ... --body ...` and `messages list`.
- **As a user**, I want `notifications list` and `notification-prefs set`.

## 5. Functional Requirements

- **FR-1.** MUST add `messages list|get|send|reply` (`registerCommunicationRoutes` inbox).
- **FR-2.** MUST add `broadcasts list|send|status` (`broadcasts_http.go`; `--audience`, `--org`, `--schedule`).
- **FR-3.** SHOULD add `announcements post|list` (course-scoped) bridging feed/communication.
- **FR-4.** SHOULD add `notifications list|read|clear` and `notification-prefs get|set` (`me/notifications`, `me/notification-preferences`).
- **FR-5.** MAY add `push test` (`me/push-subscriptions`, `me/device-tokens`).

## 6. Non-Functional Requirements

- **Performance** — inbox/broadcast lists paginated.
- **Security** — messaging/broadcast scope; broadcast to large audiences requires elevated role + `--yes`.
- **Privacy & Compliance** — messages are FERPA-adjacent; export gated by `--yes`.
- **Reliability** — send idempotent by client-supplied idempotency key to avoid duplicate blasts.
- **Backward compatibility** — existing `feed` unchanged; may alias `announcements` to feed post.

## 7. Acceptance Criteria

- **AC-1.** *Given* a notice file, *When* `broadcasts send --audience students --yes`, *Then* a broadcast id returns and `status` shows delivery counts.
- **AC-2.** *Given* a recipient, *When* `messages send`, *Then* it appears in the recipient's inbox.
- **AC-3.** *Given* re-sent with same idempotency key, *Then* no duplicate broadcast.

## 8. Data Model

- None client-side.

## 9. API Surface

- `registerCommunicationRoutes`; `broadcasts_http.go`; `me/notifications`, `me/notification-preferences`, `me/push-subscriptions`, `me/device-tokens`.

## 10. UI / UX

- `lextures messages ...`, `lextures broadcasts ...`, `lextures announcements ...`, `lextures notifications ...`.

## 11. AI / ML Considerations

- None (compose-assist, if any, is separate).

## 12. Integration Points

- Server communication/broadcast/notification handlers; course feed (existing).

## 13. Dependencies & Sequencing

- After: C40 (idempotency key helper).
- Before: none.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Accidental mass broadcast | M | H | `--yes` + audience preview + idempotency key |

## 15. Rollout Plan

- Ship broadcasts + messages first, then notifications/prefs.
- Rollback: additive.

## 16. Test Plan

- **Unit** — audience flags; idempotency key.
- **Integration** — broadcast send/status; message send.
- **E2E** — send broadcast to a test org → verify delivery counts.

## 17. Documentation & Training

- "Send an org-wide announcement from CI" recipe.

## 18. Open Questions

1. Are announcements a distinct entity or feed posts with `--announce`?

## 19. References

- `registerCommunicationRoutes`, `broadcasts_http.go`; `clients/cli/cmd/feed.go`.
- Related: [C13](C13-groups-collaboration.md), [C39](C39-profile-account-personas.md).
