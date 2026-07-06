# C35 — Meetings, office hours & conferences

> CLI parity plan. Source: `registerMeetingRoutes` (`meetings`, `courses/{id}/meetings`), `registerOfficeHoursRoutes` + `office_hours_get.go` (`slots`, `courses/{id}/availability`), `conferences.go` (`conference-slots`), `calendar_http.go` + `registerCalendarFeedRoutes` (`me/calendar-token`). Baseline: none.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | C35 |
| **Section** | Communication |
| **Severity** | MINOR |
| **Markets** | HE / K12 / SL |
| **Status (today)** | MISSING |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | Scheduling / CLI |
| **Depends on** | C40 |
| **Unblocks** | — |

---

## 1. Problem Statement

Meetings, office-hours scheduling, conferences and calendar feeds are UI-only. Instructors cannot bulk-create office-hour slots or export their schedule, and admins cannot provision meetings/conferences for many courses at once.

## 2. Goals

- Create/list/cancel meetings and conferences per course.
- Manage office-hours availability and slots in bulk.
- Export a calendar feed token / iCal for scheduling integration.

## 3. Non-Goals

- Hosting the video call (third-party/browser).
- Real-time booking UX.

## 4. Personas & User Stories

- **As an instructor**, I want `office-hours set --course C --file availability.json` to publish slots.
- **As an instructor**, I want `meetings create --course C --start --duration`.
- **As a user**, I want `calendar token` to get an iCal URL for my calendar app.

## 5. Functional Requirements

- **FR-1.** MUST add `meetings list|create|update|cancel` (`registerMeetingRoutes`).
- **FR-2.** MUST add `office-hours set|list|slots <course>` and `availability get|set` (`office_hours_get.go`).
- **FR-3.** SHOULD add `conferences list|create|slots` (`conferences.go`).
- **FR-4.** SHOULD add `calendar token get|rotate` and `calendar export --ical` (`me/calendar-token`, `calendar_http.go`).

## 6. Non-Functional Requirements

- **Performance** — trivial payloads.
- **Security** — scheduling scope; calendar token is a secret URL → shown once, rotatable.
- **Privacy & Compliance** — calendar may reveal enrollment (FERPA); token rotation supported.
- **Reliability** — slot creation idempotent by time window.
- **Internationalization** — timezone-aware (`--tz`, defaults to profile/locale).
- **Backward compatibility** — additive.

## 7. Acceptance Criteria

- **AC-1.** *Given* an availability file, *When* `office-hours set`, *Then* slots are created and `slots` lists them.
- **AC-2.** *Given* a course, *When* `meetings create --start ... --tz`, *Then* the meeting appears with correct tz.
- **AC-3.** *Given* `calendar token`, *Then* a secret iCal URL prints once.

## 8. Data Model

- None client-side. Document availability JSON.

## 9. API Surface

- `registerMeetingRoutes`; `registerOfficeHoursRoutes` + `slots`/`availability`; `conferences.go`; `me/calendar-token` + `calendar_http.go`.

## 10. UI / UX

- `lextures meetings ...`, `lextures office-hours ...`, `lextures conferences ...`, `lextures calendar ...`.

## 11. AI / ML Considerations

- None (scheduling suggestions, if any, are server-side).

## 12. Integration Points

- Server meeting/office-hours/conference/calendar handlers.

## 13. Dependencies & Sequencing

- After: C40 (tz handling).
- Before: none.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Timezone/DST bugs | M | M | Always send explicit tz; test around DST boundaries |
| Calendar token leakage | M | M | One-time display + `rotate` |

## 15. Rollout Plan

- Ship meetings + office-hours first, then conferences + calendar export.
- Rollback: additive.

## 16. Test Plan

- **Unit** — tz handling; slot window idempotency.
- **Integration** — office-hours set; meeting create.
- **E2E** — publish office hours → list slots.

## 17. Documentation & Training

- "Bulk-publish office hours" recipe.

## 18. Open Questions

1. Are meetings tied to a video provider requiring extra config?

## 19. References

- `registerMeetingRoutes`, `office_hours_get.go`, `conferences.go`, `calendar_http.go`.
- Related: [C34](C34-messaging-broadcasts.md), [C39](C39-profile-account-personas.md).
