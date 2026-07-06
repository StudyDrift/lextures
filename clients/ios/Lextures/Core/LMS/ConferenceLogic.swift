import Foundation

/// Conference slot formatting and filtering (M10.2, reuses office-hours patterns).
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

    private static func endTimeOnly(_ raw: String) -> String {
        guard let date = DateFormatting.parse(raw) else { return raw }
        let formatter = DateFormatter()
        formatter.locale = LocalePreferences.effectiveLocaleValue()
        formatter.timeZone = .current
        formatter.dateStyle = .none
        formatter.timeStyle = .short
        return formatter.string(from: date)
    }

    static func todayDateString() -> String {
        let formatter = DateFormatter()
        formatter.locale = Locale(identifier: "en_US_POSIX")
        formatter.timeZone = .current
        formatter.dateFormat = "yyyy-MM-dd"
        return formatter.string(from: Date())
    }
}
