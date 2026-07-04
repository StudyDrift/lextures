import Foundation

/// Live virtual-meeting formatting, grouping, and planner calendar mapping (M7.5).
enum LiveMeetingsLogic {
    static let soonWindowSeconds: TimeInterval = 30 * 60

    struct GroupedMeetings: Equatable {
        var live: [VirtualMeeting]
        var upcoming: [VirtualMeeting]
        var past: [VirtualMeeting]
    }

    struct LiveUpcomingItem: Identifiable, Hashable {
        var id: String { "\(courseCode):\(meeting.id)" }
        let courseCode: String
        let courseTitle: String
        let meeting: VirtualMeeting
    }

    static func groupMeetings(_ meetings: [VirtualMeeting]) -> GroupedMeetings {
        GroupedMeetings(
            live: meetings.filter { $0.status == "live" },
            upcoming: meetings.filter { $0.status == "scheduled" },
            past: meetings.filter { $0.status == "ended" }
        )
    }

    static func isLiveOrSoon(_ meeting: VirtualMeeting, now: Date = Date()) -> Bool {
        if meeting.status == "live" { return true }
        guard meeting.status == "scheduled",
              let startRaw = meeting.scheduledStart,
              let start = LMSDates.parse(startRaw) else { return false }
        let diff = start.timeIntervalSince(now)
        return diff >= 0 && diff <= soonWindowSeconds
    }

    static func canJoin(_ meeting: VirtualMeeting, now: Date = Date()) -> Bool {
        guard meeting.status != "cancelled", meeting.status != "ended" else { return false }
        return meeting.status == "live" || isLiveOrSoon(meeting, now: now)
    }

    static func formatMeetingTime(_ meeting: VirtualMeeting) -> String {
        guard let startRaw = meeting.scheduledStart,
              LMSDates.parse(startRaw) != nil else {
            return L.text("mobile.live.noTime")
        }
        let startText = DateFormatting.formatDateTime(startRaw)
        guard let endRaw = meeting.scheduledEnd,
              LMSDates.parse(endRaw) != nil else { return startText }
        let endText = endTimeOnly(endRaw)
        return "\(startText) – \(endText)"
    }

    static func countdownText(scheduledStart: String, now: Date = Date()) -> String? {
        guard let start = LMSDates.parse(scheduledStart) else { return nil }
        let seconds = max(0, Int(start.timeIntervalSince(now)))
        guard seconds > 0 else { return nil }
        let minutes = seconds / 60
        let remainder = seconds % 60
        return L.format("mobile.live.startsIn", "\(minutes)", String(format: "%02d", remainder))
    }

    static func providerLabel(_ provider: String) -> String {
        let key = provider.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        let localized = L.text("mobile.live.provider.\(key)")
        if localized != "mobile.live.provider.\(key)" { return localized }
        return provider.capitalized
    }

    static func statusLabel(_ status: String) -> String {
        switch status {
        case "live": return L.text("mobile.live.status.live")
        case "scheduled": return L.text("mobile.live.status.scheduled")
        case "ended": return L.text("mobile.live.status.ended")
        case "cancelled": return L.text("mobile.live.status.cancelled")
        default: return status.capitalized
        }
    }

    static func calendarEvents(
        courseCode: String,
        courseTitle: String,
        meetings: [VirtualMeeting]
    ) -> [PlannerCalendarEvent] {
        meetings.compactMap { meeting in
            guard meeting.status == "scheduled" || meeting.status == "live",
                  let startRaw = meeting.scheduledStart,
                  let start = LMSDates.parse(startRaw) else { return nil }
            let end = meeting.scheduledEnd.flatMap { LMSDates.parse($0) }
            return PlannerCalendarEvent(
                id: "live-meeting:\(meeting.id)",
                title: meeting.title,
                courseCode: courseCode,
                courseTitle: courseTitle,
                startsAt: start,
                endsAt: end,
                allDay: false,
                kind: .liveMeeting,
                structureKind: nil,
                structureItemId: nil,
                notebookPageId: nil,
                officeHoursSlotId: nil,
                meetingId: meeting.id
            )
        }
    }

    static func collectCalendarEvents(
        studentCourses: [CourseSummary],
        meetingsByCourseCode: [String: [VirtualMeeting]]
    ) -> [PlannerCalendarEvent] {
        studentCourses.flatMap { course in
            guard course.isLiveSessionsEnabled,
                  let meetings = meetingsByCourseCode[course.courseCode] else { return [PlannerCalendarEvent]() }
            return calendarEvents(
                courseCode: course.courseCode,
                courseTitle: course.displayTitle,
                meetings: meetings
            )
        }
    }

    static func collectLiveAndUpcoming(
        courses: [CourseSummary],
        meetingsByCourseCode: [String: [VirtualMeeting]],
        limit: Int = 5,
        now: Date = Date()
    ) -> [LiveUpcomingItem] {
        var items: [LiveUpcomingItem] = []
        for course in courses where course.isLiveSessionsEnabled {
            guard let meetings = meetingsByCourseCode[course.courseCode] else { continue }
            for meeting in meetings where meeting.status == "live" || isLiveOrSoon(meeting, now: now) {
                items.append(
                    LiveUpcomingItem(
                        courseCode: course.courseCode,
                        courseTitle: course.displayTitle,
                        meeting: meeting
                    )
                )
            }
        }
        return items
            .sorted { lhs, rhs in
                let left = LMSDates.parse(lhs.meeting.scheduledStart) ?? .distantFuture
                let right = LMSDates.parse(rhs.meeting.scheduledStart) ?? .distantFuture
                if lhs.meeting.status == "live", rhs.meeting.status != "live" { return true }
                if rhs.meeting.status == "live", lhs.meeting.status != "live" { return false }
                return left < right
            }
            .prefix(limit)
            .map { $0 }
    }

    static func meetingIcalURL(meetingId: String) -> URL? {
        AppConfiguration.apiURL(path: "/api/v1/meetings/\(meetingId)/ical")
    }

    private static func endTimeOnly(_ raw: String) -> String {
        guard let date = DateFormatting.parse(raw) else { return raw }
        let formatter = DateFormatter()
        formatter.locale = LocalePreferences.effectiveLocaleValue()
        formatter.timeZone = .current
        formatter.dateStyle = .none
        formatter.timeStyle = .short
        return formatter.string(from: date)
    }
}