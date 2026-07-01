import Foundation

/// Office-hours slot formatting, filtering, and planner calendar mapping (M7.3).
enum OfficeHoursLogic {
    static func isMyBooking(_ slot: AppointmentSlot) -> Bool {
        slot.status == "booked" && slot.studentId != nil
    }

    static func upcomingAvailableSlots(_ slots: [AppointmentSlot], now: Date = Date()) -> [AppointmentSlot] {
        slots
            .filter { $0.status == "available" }
            .filter { guard let start = LMSDates.parse($0.slotStart) else { return false }; return start >= now }
            .sorted { ($0.slotStart) < ($1.slotStart) }
    }

    static func myBookedSlots(_ slots: [AppointmentSlot], now: Date = Date()) -> [AppointmentSlot] {
        slots
            .filter(isMyBooking)
            .filter { guard let start = LMSDates.parse($0.slotStart) else { return false }; return start >= now }
            .sorted { ($0.slotStart) < ($1.slotStart) }
    }

    static func formatSlotTime(_ slot: AppointmentSlot) -> String {
        guard LMSDates.parse(slot.slotStart) != nil else { return slot.slotStart }
        let startText = DateFormatting.formatDateTime(slot.slotStart)
        guard LMSDates.parse(slot.slotEnd) != nil else { return startText }
        let endText = endTimeOnly(slot.slotEnd)
        return "\(startText) – \(endText)"
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

    static func locationLabel(window: AvailabilityWindow?) -> String? {
        guard let window else { return nil }
        let location = window.location?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        guard !location.isEmpty else {
            return window.isVirtual ? L.text("mobile.officeHours.virtual") : nil
        }
        return window.isVirtual ? "\(L.text("mobile.officeHours.virtual")) · \(location)" : location
    }

    static func windowMap(_ windows: [AvailabilityWindow]) -> [String: AvailabilityWindow] {
        Dictionary(uniqueKeysWithValues: windows.map { ($0.id, $0) })
    }

    static func calendarEvents(
        courseCode: String,
        courseTitle: String,
        slots: [AppointmentSlot],
        windows: [AvailabilityWindow]
    ) -> [PlannerCalendarEvent] {
        let lookup = windowMap(windows)
        return myBookedSlots(slots).compactMap { slot in
            guard let start = LMSDates.parse(slot.slotStart) else { return nil }
            let end = LMSDates.parse(slot.slotEnd)
            return PlannerCalendarEvent(
                id: "office-hours:\(slot.id)",
                title: L.text("mobile.officeHours.calendarTitle"),
                courseCode: courseCode,
                courseTitle: courseTitle,
                startsAt: start,
                endsAt: end,
                allDay: false,
                kind: .officeHours,
                structureKind: nil,
                structureItemId: nil,
                notebookPageId: nil,
                officeHoursSlotId: slot.id,
                meetingId: slot.meetingId
            )
        }
    }

    static func collectCalendarEvents(
        studentCourses: [CourseSummary],
        availabilityByCourseCode: [String: OfficeHoursAvailability]
    ) -> [PlannerCalendarEvent] {
        studentCourses.flatMap { course in
            guard course.isOfficeHoursEnabled,
                  let availability = availabilityByCourseCode[course.courseCode] else { return [PlannerCalendarEvent]() }
            return calendarEvents(
                courseCode: course.courseCode,
                courseTitle: course.displayTitle,
                slots: availability.slots,
                windows: availability.windows
            )
        }
    }
}
