import Foundation

enum InsightsLogic {
    static let journalMaxLength = 280

    static func formatHours(_ hours: Double) -> String {
        if hours < 0.1 { return "0" }
        return hours >= 10 ? String(format: "%.0f", hours) : String(format: "%.1f", hours)
    }

    static func hoursFromSeconds(_ seconds: Int) -> Double {
        Double(seconds) / 3600.0
    }

    static func goalProgressPercent(progressHours: Float, goalHours: Float?) -> Int? {
        guard let goalHours, goalHours > 0 else { return nil }
        let pct = Int((Double(progressHours) / Double(goalHours)) * 100.0)
        return min(100, max(0, pct))
    }

    static func maxAllocationMinutes(_ rows: [StudyTimeAllocationRow]) -> Double {
        max(1, rows.map(\.minutes).max() ?? 1)
    }

    static func barWidthPercent(minutes: Double, maxMinutes: Double) -> Double {
        guard maxMinutes > 0 else { return 0 }
        return min(100, max(0, (minutes / maxMinutes) * 100))
    }

    static func moduleCompletionPercent(_ snapshot: ModulesProgressSnapshot) -> Int {
        var total = 0
        var complete = 0
        for module in snapshot.modules {
            if let items = module.items, !items.isEmpty {
                total += items.count
                complete += items.filter(\.complete).count
            } else {
                total += 1
                if module.complete { complete += 1 }
            }
        }
        guard total > 0 else { return 0 }
        return min(100, max(0, (complete * 100) / total))
    }

    static func journalEntryValid(_ text: String) -> Bool {
        let trimmed = text.trimmingCharacters(in: .whitespacesAndNewlines)
        return !trimmed.isEmpty && trimmed.count <= journalMaxLength
    }

    static func latestCoachingTip(from response: CoachingTipsResponse) -> CoachingTip? {
        response.latest ?? response.history?.first
    }

    static func formatJournalDate(_ iso: String) -> String {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        let date = formatter.date(from: iso) ?? ISO8601DateFormatter().date(from: iso)
        guard let date else { return iso }
        return date.formatted(date: .abbreviated, time: .shortened)
    }
}