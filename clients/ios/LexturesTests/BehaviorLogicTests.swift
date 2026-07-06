import XCTest
@testable import Lextures

final class BehaviorLogicTests: XCTestCase {
    func testStudentRosterFiltersLearners() {
        let enrollments = [
            CourseEnrollment(id: "1", userId: "s1", displayName: "Alex", role: "student"),
            CourseEnrollment(id: "2", userId: "t1", displayName: "Teacher", role: "teacher"),
            CourseEnrollment(id: "3", userId: "s2", displayName: nil, role: "learner"),
        ]
        XCTAssertEqual(BehaviorLogic.studentRoster(from: enrollments).map(\.userId), ["s1", "s2"])
    }

    func testPositiveAndNegativeCategories() {
        let categories = [
            BehaviorCategory(id: "1", orgId: "o", name: "Respect", type: "positive", color: nil, active: true),
            BehaviorCategory(id: "2", orgId: "o", name: "Tardy", type: "negative", color: nil, active: true),
            BehaviorCategory(id: "3", orgId: "o", name: "Old", type: "positive", color: nil, active: false),
        ]
        XCTAssertEqual(BehaviorLogic.positiveCategories(categories).map(\.id), ["1"])
        XCTAssertEqual(BehaviorLogic.negativeCategories(categories).map(\.id), ["2"])
    }

    func testAwardPayloadBuildsOnePerStudent() {
        let payload = BehaviorLogic.awardPayload(
            studentIds: ["a", "b"],
            categoryId: "cat",
            note: " nice "
        )
        XCTAssertEqual(payload.count, 2)
        XCTAssertEqual(payload[0].points, 1)
        XCTAssertEqual(payload[0].note, "nice")
    }

    func testHallPassCountdownUsesApprovedAt() {
        let approved = "2026-07-06T12:00:00.000Z"
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        let now = formatter.date(from: "2026-07-06T12:02:00.000Z")!
        let pass = HallPass(
            id: "p1",
            sectionId: "s1",
            studentId: "u1",
            destination: "bathroom",
            status: "approved",
            estimatedMins: 5,
            requestedAt: approved,
            approvedAt: approved,
            returnedAt: nil,
            approvedBy: nil,
            overdue: false
        )
        let countdown = BehaviorLogic.hallPassCountdown(pass: pass, now: now)
        XCTAssertNotNil(countdown)
        XCTAssertEqual(countdown?.remainingSeconds, 180)
    }

    func testIsActiveHallPass() {
        let requested = HallPass(
            id: "1", sectionId: "s", destination: "office", status: "requested", requestedAt: ""
        )
        let returned = HallPass(
            id: "2", sectionId: "s", destination: "office", status: "returned", requestedAt: ""
        )
        XCTAssertTrue(BehaviorLogic.isActiveHallPass(requested))
        XCTAssertFalse(BehaviorLogic.isActiveHallPass(returned))
    }
}
