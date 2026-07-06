import Foundation

/// A parent's booked conference slot with teacher and child context (M10.2).
struct ParentConferenceBooking: Hashable, Identifiable {
    var slot: ConferenceSlot
    var teacher: ConferenceTeacher
    var studentId: String
    var childName: String
    var availability: ConferenceAvailability?

    var id: String { slot.id }
}

/// Conference slot formatting, filtering, and planner calendar mapping (M10.2).
enum ConferenceLogic {
    static func isMyBooking(_ slot: ConferenceSlot, parentId: String?, studentId: String) -> Bool {
        slot.status == "booked"
            && slot.bookedForChild == studentId
            && (parentId == nil || slot.bookedByParent == nil || slot.bookedByParent == parentId)
    }

    static func upcomingAvailableSlots(_ slots: [ConferenceSlot], now: Date = Date()) -> [ConferenceSlot] {
        slots
            .filter { $0.status == "open" }
            .filter { guard let start = LMSDates.parse($0.startAt) else { return false }; return start >= now }
            .sorted { $0.startAt < $1.startAt }
    }

    static func myBookedSlots(
        _ slots: [ConferenceSlot],
        parentId: String?,
        studentId: String,
        now: Date = Date()
    ) -> [ConferenceSlot] {
        slots
            .filter { isMyBooking($0, parentId: parentId, studentId: studentId) }
            .filter { guard let start = LMSDates.parse($0.startAt) else { return false }; return start >= now }
            .sorted { $0.startAt < $1.startAt }
    }

    static func formatSlotTime(_ slot: ConferenceSlot) -> String {
        guard LMSDates.parse(slot.startAt) != nil else { return slot.startAt }
        let startText = DateFormatting.formatDateTime(slot.startAt)
        guard LMSDates.parse(slot.endAt) != nil else { return startText }
        let endText = endTimeOnly(slot.endAt)
        return "\(startText) – \(endText)"
    }

    static func locationLabel(availability: ConferenceAvailability?) -> String? {
        guard let availability else { return nil }
        let location = availability.location?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        let videoLink = availability.videoLink?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        let isVirtual = !videoLink.isEmpty
        guard !location.isEmpty else {
            return isVirtual ? L.text("mobile.parent.conferences.virtual") : nil
        }
        return isVirtual ? "\(L.text("mobile.parent.conferences.virtual")) · \(location)" : location
    }

    static func isJoinWindow(_ slot: ConferenceSlot, availability: ConferenceAvailability?) -> Bool {
        guard let videoLink = availability?.videoLink?.trimmingCharacters(in: .whitespacesAndNewlines),
              !videoLink.isEmpty,
              let start = LMSDates.parse(slot.startAt) else { return false }
        let end = LMSDates.parse(slot.endAt) ?? start.addingTimeInterval(15 * 60)
        let now = Date()
        let openFrom = start.addingTimeInterval(-10 * 60)
        return now >= openFrom && now <= end
    }

    static func icalURL(for slotId: String) -> URL? {
        AppConfiguration.apiURL(path: "/api/v1/conference-slots/\(slotId)/ical")
    }

    static func todayDateString() -> String {
        let formatter = DateFormatter()
        formatter.locale = Locale(identifier: "en_US_POSIX")
        formatter.timeZone = .current
        formatter.dateFormat = "yyyy-MM-dd"
        return formatter.string(from: Date())
    }

    static func scanDates(dayCount: Int = 21, from: Date = Date()) -> [String] {
        let formatter = DateFormatter()
        formatter.locale = Locale(identifier: "en_US_POSIX")
        formatter.timeZone = .current
        formatter.dateFormat = "yyyy-MM-dd"
        let calendar = Calendar.current
        return (0 ..< dayCount).compactMap { offset in
            guard let date = calendar.date(byAdding: .day, value: offset, to: from) else { return nil }
            return formatter.string(from: date)
        }
    }

    static func loadParentBookings(
        children: [(studentId: String, childName: String)],
        accessToken: String,
        dates: [String]? = nil
    ) async -> [ParentConferenceBooking] {
        let scan = dates ?? scanDates()
        var results: [ParentConferenceBooking] = []
        for child in children {
            guard let teachers = try? await LMSAPI.fetchParentConferenceTeachers(
                studentId: child.studentId,
                accessToken: accessToken
            ) else { continue }
            for teacher in teachers {
                for date in scan {
                    guard let response = try? await LMSAPI.fetchConferenceSlots(
                        teacherId: teacher.teacherId,
                        date: date,
                        accessToken: accessToken
                    ) else { continue }
                    let booked = myBookedSlots(response.slots ?? [], parentId: nil, studentId: child.studentId)
                    for slot in booked {
                        results.append(
                            ParentConferenceBooking(
                                slot: slot,
                                teacher: teacher,
                                studentId: child.studentId,
                                childName: child.childName,
                                availability: response.availability
                            )
                        )
                    }
                }
            }
        }
        return results.sorted { $0.slot.startAt < $1.slot.startAt }
    }

    static func calendarEvents(from bookings: [ParentConferenceBooking]) -> [PlannerCalendarEvent] {
        bookings.compactMap { booking in
            guard let start = LMSDates.parse(booking.slot.startAt) else { return nil }
            let end = LMSDates.parse(booking.slot.endAt)
            let teacherName = ParentLogic.teacherLabel(booking.teacher)
            return PlannerCalendarEvent(
                id: "conference:\(booking.slot.id)",
                title: L.format("mobile.parent.conferences.calendarTitle", teacherName),
                courseCode: nil,
                courseTitle: booking.childName,
                startsAt: start,
                endsAt: end,
                allDay: false,
                kind: .conference,
                structureKind: nil,
                structureItemId: nil,
                notebookPageId: nil,
                conferenceSlotId: booking.slot.id,
                videoLink: booking.availability?.videoLink
            )
        }
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
