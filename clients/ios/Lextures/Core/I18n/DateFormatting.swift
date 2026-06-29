import Foundation

/// Locale- and timezone-aware date/number formatting for LMS timestamps (plan M0.4 / M2.1).
enum DateFormatting {
    private static let isoFractional: ISO8601DateFormatter = {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        return formatter
    }()

    private static let iso: ISO8601DateFormatter = {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime]
        return formatter
    }()

    static func parse(_ raw: String?) -> Date? {
        guard let raw, !raw.isEmpty else { return nil }
        return isoFractional.date(from: raw) ?? iso.date(from: raw)
    }

    static func formatAbsoluteShort(_ raw: String?, timeZone: TimeZone = .current) -> String {
        guard let date = parse(raw) else { return L.text("mobile.emDash") }
        let formatter = DateFormatter()
        formatter.locale = LocalePreferences.effectiveLocaleValue()
        formatter.timeZone = timeZone
        formatter.dateStyle = .medium
        formatter.timeStyle = .short
        return formatter.string(from: date)
    }

    static func formatDue(_ raw: String?, timeZone: TimeZone = .current) -> String {
        L.format("mobile.courses.due", formatAbsoluteShort(raw, timeZone: timeZone))
    }

    static func formatDate(_ raw: String?, timeZone: TimeZone = .current) -> String {
        guard let date = parse(raw) else { return L.text("mobile.emDash") }
        let formatter = DateFormatter()
        formatter.locale = LocalePreferences.effectiveLocaleValue()
        formatter.timeZone = timeZone
        formatter.dateStyle = .medium
        formatter.timeStyle = .none
        return formatter.string(from: date)
    }

    static func formatDateTime(_ raw: String?, timeZone: TimeZone = .current) -> String {
        guard let date = parse(raw) else { return "" }
        let formatter = DateFormatter()
        formatter.locale = LocalePreferences.effectiveLocaleValue()
        formatter.timeZone = timeZone
        formatter.dateStyle = .medium
        formatter.timeStyle = .short
        return formatter.string(from: date)
    }

    static func formatRelative(_ raw: String?) -> String {
        guard let date = parse(raw) else { return "" }
        let formatter = RelativeDateTimeFormatter()
        formatter.locale = LocalePreferences.effectiveLocaleValue()
        formatter.unitsStyle = .short
        return formatter.localizedString(for: date, relativeTo: Date())
    }

    static func formatNumber(_ value: Double, maximumFractionDigits: Int = 0) -> String {
        let formatter = NumberFormatter()
        formatter.locale = LocalePreferences.effectiveLocaleValue()
        formatter.numberStyle = .decimal
        formatter.maximumFractionDigits = maximumFractionDigits
        formatter.minimumFractionDigits = 0
        return formatter.string(from: NSNumber(value: value)) ?? "\(value)"
    }

    static func formatPoints(_ value: Double) -> String {
        L.format("mobile.courses.points", formatNumber(value, maximumFractionDigits: 1))
    }
}
