import Foundation

/// Academic advising helpers (M7.8).
enum AdvisingLogic {
    static func advisingEnabled(_ features: MobilePlatformFeatures) -> Bool {
        features.ffAdvisingIntegration && features.ffMobileAdvising
    }

    static func notesCacheKey() -> String { "advising:notes" }

    static func degreeProgressCacheKey() -> String { "advising:degree-progress" }

    static func visibleNotes(_ notes: [AdvisingNote]) -> [AdvisingNote] {
        notes.filter(\.visibleToStudent)
    }

    static func sortedNotes(_ notes: [AdvisingNote]) -> [AdvisingNote] {
        visibleNotes(notes).sorted { lhs, rhs in
            parseDate(lhs.createdAt) > parseDate(rhs.createdAt)
        }
    }

    static func advisorFromNotes(_ notes: [AdvisingNote]) -> AdvisingAdvisorInfo? {
        guard let newest = sortedNotes(notes).first else { return nil }
        let display = advisorLabel(
            displayName: newest.advisorDisplayName,
            email: newest.advisorEmail
        )
        return AdvisingAdvisorInfo(displayName: display, email: newest.advisorEmail)
    }

    static func advisorLabel(displayName: String?, email: String?) -> String {
        let trimmedName = displayName?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        if !trimmedName.isEmpty { return trimmedName }
        let trimmedEmail = email?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        if !trimmedEmail.isEmpty { return trimmedEmail }
        return L.text("mobile.advising.advisorFallback")
    }

    static func appointmentURL(progress: DegreeProgress?, config: MyAdvisingConfig?) -> String? {
        let fromProgress = progress?.appointmentUrl?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        if !fromProgress.isEmpty { return fromProgress }
        let fromConfig = config?.appointmentUrl?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        return fromConfig.isEmpty ? nil : fromConfig
    }

    static func canBookAppointment(isOnline: Bool, appointmentURL: String?) -> Bool {
        guard let appointmentURL, !appointmentURL.isEmpty else { return false }
        return isOnline
    }

    static func formatNoteDate(iso: String) -> String {
        let date = parseDate(iso)
        guard date != .distantPast else { return iso }
        return date.formatted(date: .abbreviated, time: .shortened)
    }

    static func formatAuditDate(iso: String) -> String {
        let date = parseDate(iso)
        guard date != .distantPast else { return iso }
        return date.formatted(date: .abbreviated, time: .shortened)
    }

    private static func parseDate(_ iso: String) -> Date {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        if let date = formatter.date(from: iso) { return date }
        formatter.formatOptions = [.withInternetDateTime]
        return formatter.date(from: iso) ?? .distantPast
    }
}
