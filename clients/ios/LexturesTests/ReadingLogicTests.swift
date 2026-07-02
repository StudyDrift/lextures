import XCTest
@testable import Lextures

final class ReadingLogicTests: XCTestCase {
    private func parseDate(_ value: String) -> Date {
        let formatter = DateFormatter()
        formatter.calendar = Calendar(identifier: .gregorian)
        formatter.locale = Locale(identifier: "en_US_POSIX")
        formatter.timeZone = TimeZone.current
        formatter.dateFormat = "yyyy-MM-dd"
        return formatter.date(from: value)!
    }

    func testWeeklyPagesCountsLastSevenDays() {
        let entries = [
            ReadingLogEntry(id: "1", logDate: "2026-07-01", pagesRead: 10),
            ReadingLogEntry(id: "2", logDate: "2025-01-01", pagesRead: 99),
            ReadingLogEntry(id: "3", logDate: "2026-07-02", pagesRead: 5),
        ]
        XCTAssertEqual(ReadingLogic.weeklyPages(from: entries, asOf: parseDate("2026-07-02")), 15)
    }

    func testReadingStreakCountsConsecutiveDays() {
        let entries = [
            ReadingLogEntry(id: "1", logDate: "2026-07-02", pagesRead: 1),
            ReadingLogEntry(id: "2", logDate: "2026-07-01", pagesRead: 1),
            ReadingLogEntry(id: "3", logDate: "2026-06-29", pagesRead: 1),
        ]
        XCTAssertEqual(ReadingLogic.readingStreakDays(from: entries, asOf: parseDate("2026-07-02")), 2)
    }

    func testLogEntryValidRequiresTitleOrBookId() {
        XCTAssertFalse(ReadingLogic.logEntryValid(bookTitle: "", bookId: nil, logDate: "2026-07-02"))
        XCTAssertTrue(ReadingLogic.logEntryValid(bookTitle: "Frog and Toad", bookId: nil, logDate: "2026-07-02"))
        XCTAssertTrue(ReadingLogic.logEntryValid(bookTitle: nil, bookId: "book-1", logDate: "2026-07-02"))
    }

    func testBookClubCoursesFiltersStudentGroups() {
        let courses = [
            CourseSummary(
                id: "1", courseCode: "a", title: "Alpha", description: "",
                viewerEnrollmentRoles: ["student"], groupSpacesEnabled: true
            ),
            CourseSummary(
                id: "2", courseCode: "b", title: "Beta", description: "",
                viewerEnrollmentRoles: ["student"], groupSpacesEnabled: false
            ),
        ]
        XCTAssertEqual(ReadingLogic.bookClubCourses(from: courses).count, 1)
        XCTAssertEqual(ReadingLogic.bookClubCourses(from: courses).first?.courseCode, "a")
    }
}