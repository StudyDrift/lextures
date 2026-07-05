import XCTest
@testable import Lextures

final class AdvisingLogicTests: XCTestCase {
    func testAdvisingEnabled() {
        var features = MobilePlatformFeatures()
        XCTAssertFalse(AdvisingLogic.advisingEnabled(features))
        features.ffAdvisingIntegration = true
        XCTAssertTrue(AdvisingLogic.advisingEnabled(features))
        features.ffMobileAdvising = false
        XCTAssertFalse(AdvisingLogic.advisingEnabled(features))
    }

    func testVisibleNotesFiltersStaffOnly() {
        let notes = [
            AdvisingNote(
                id: "1",
                studentId: "s",
                advisorId: "a",
                content: "visible",
                visibleToStudent: true,
                createdAt: "2026-01-02T12:00:00Z",
                advisorEmail: nil,
                advisorDisplayName: nil
            ),
            AdvisingNote(
                id: "2",
                studentId: "s",
                advisorId: "a",
                content: "hidden",
                visibleToStudent: false,
                createdAt: "2026-01-03T12:00:00Z",
                advisorEmail: nil,
                advisorDisplayName: nil
            ),
        ]
        XCTAssertEqual(AdvisingLogic.visibleNotes(notes).count, 1)
        XCTAssertEqual(AdvisingLogic.sortedNotes(notes).first?.id, "1")
    }

    func testAppointmentURLPrefersProgress() {
        let progress = DegreeProgress(
            configured: true,
            completionPercent: 50,
            remainingRequiredCount: 2,
            remainingRequirements: nil,
            atRisk: false,
            lastUpdated: nil,
            stale: false,
            appointmentUrl: "https://progress.example/book",
            recentNotesCount: nil
        )
        let config = MyAdvisingConfig(appointmentUrl: "https://config.example/book")
        XCTAssertEqual(
            AdvisingLogic.appointmentURL(progress: progress, config: config),
            "https://progress.example/book"
        )
    }

    func testCanBookAppointmentRequiresOnline() {
        XCTAssertFalse(AdvisingLogic.canBookAppointment(isOnline: false, appointmentURL: "https://example.com"))
        XCTAssertTrue(AdvisingLogic.canBookAppointment(isOnline: true, appointmentURL: "https://example.com"))
        XCTAssertFalse(AdvisingLogic.canBookAppointment(isOnline: true, appointmentURL: nil))
    }

    func testCacheKeys() {
        XCTAssertEqual(AdvisingLogic.notesCacheKey(), "advising:notes")
        XCTAssertEqual(AdvisingLogic.degreeProgressCacheKey(), "advising:degree-progress")
    }
}
