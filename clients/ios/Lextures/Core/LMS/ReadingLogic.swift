import Foundation

enum ReadingLogic {
    static let reflectionMaxLength = 500
    static let gradeBands = ["", "K-2", "3-5", "6-8", "9-12"]

    static func todayISO() -> String {
        let formatter = DateFormatter()
        formatter.calendar = Calendar(identifier: .gregorian)
        formatter.locale = Locale(identifier: "en_US_POSIX")
        formatter.timeZone = TimeZone(secondsFromGMT: 0)
        formatter.dateFormat = "yyyy-MM-dd"
        return formatter.string(from: Date())
    }

    static func logEntryValid(bookTitle: String?, bookId: String?, logDate: String) -> Bool {
        let title = bookTitle?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        let book = bookId?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        let date = logDate.trimmingCharacters(in: .whitespacesAndNewlines)
        return (!title.isEmpty || !book.isEmpty) && !date.isEmpty
    }

    static func weeklyPages(from entries: [ReadingLogEntry], asOf: Date = Date()) -> Int {
        let cutoff = Calendar.current.date(byAdding: .day, value: -6, to: asOf) ?? asOf
        return entries.reduce(0) { sum, entry in
            guard let date = parseLogDate(entry.logDate), date >= cutoff else { return sum }
            return sum + (entry.pagesRead ?? 0)
        }
    }

    static func readingStreakDays(from entries: [ReadingLogEntry], asOf: Date = Date()) -> Int {
        let dates = Set(entries.compactMap { parseLogDate($0.logDate).map { startOfDay($0) } })
        guard !dates.isEmpty else { return 0 }
        var streak = 0
        var cursor = startOfDay(asOf)
        while dates.contains(cursor) {
            streak += 1
            guard let previous = Calendar.current.date(byAdding: .day, value: -1, to: cursor) else { break }
            cursor = previous
        }
        return streak
    }

    static func resolveOrgId(from courses: [CourseSummary]) -> String? {
        courses.compactMap(\.orgId).first { !$0.isEmpty }
    }

    static func bookClubCourses(from courses: [CourseSummary]) -> [CourseSummary] {
        courses
            .filter { $0.viewerIsStudent && $0.isGroupSpacesEnabled }
            .sorted { $0.displayTitle.localizedCaseInsensitiveCompare($1.displayTitle) == .orderedAscending }
    }

    static func formatLexile(_ level: Int?) -> String? {
        guard let level else { return nil }
        return "Lexile \(level)L"
    }

    static func bookSubtitle(_ book: LibraryBook) -> String? {
        if let author = book.author?.trimmingCharacters(in: .whitespacesAndNewlines), !author.isEmpty {
            return author
        }
        return formatLexile(book.lexileLevel) ?? book.fpBand ?? book.gradeBand
    }

    private static func parseLogDate(_ value: String) -> Date? {
        let formatter = DateFormatter()
        formatter.calendar = Calendar(identifier: .gregorian)
        formatter.locale = Locale(identifier: "en_US_POSIX")
        formatter.timeZone = TimeZone.current
        formatter.dateFormat = "yyyy-MM-dd"
        return formatter.date(from: value)
    }

    private static func startOfDay(_ date: Date) -> Date {
        Calendar.current.startOfDay(for: date)
    }
}