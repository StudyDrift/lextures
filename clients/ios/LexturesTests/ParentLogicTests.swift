import XCTest
@testable import Lextures

final class ParentLogicTests: XCTestCase {
    func testChildLabelPrefersDisplayName() {
        let child = ParentChildSummary(
            linkId: "1",
            studentUserId: "s1",
            displayName: "  Alex  ",
            email: "alex@school.edu",
            relationship: "parent",
            status: "active",
            linkedAt: "2026-01-01T00:00:00Z"
        )
        XCTAssertEqual(ParentLogic.childLabel(child), "Alex")
    }

    func testChildLabelFallsBackToEmail() {
        let child = ParentChildSummary(
            linkId: "1",
            studentUserId: "s1",
            displayName: "  ",
            email: "alex@school.edu",
            relationship: "parent",
            status: "active",
            linkedAt: "2026-01-01T00:00:00Z"
        )
        XCTAssertEqual(ParentLogic.childLabel(child), "alex@school.edu")
    }

    func testResolveSelectedChildId() {
        let children = [
            ParentChildSummary(
                linkId: "1",
                studentUserId: "a",
                displayName: "A",
                email: "a@x",
                relationship: "parent",
                status: "active",
                linkedAt: ""
            ),
            ParentChildSummary(
                linkId: "2",
                studentUserId: "b",
                displayName: "B",
                email: "b@x",
                relationship: "parent",
                status: "active",
                linkedAt: ""
            ),
        ]
        XCTAssertEqual(ParentLogic.resolveSelectedChildId(children: children, storedId: "b"), "b")
        XCTAssertEqual(ParentLogic.resolveSelectedChildId(children: children, storedId: "missing"), "a")
        XCTAssertNil(ParentLogic.resolveSelectedChildId(children: [], storedId: "a"))
    }

    func testAttendanceSummary() {
        let records = [
            ParentAttendanceRecord(id: "1", date: "2026-01-01", category: "present"),
            ParentAttendanceRecord(id: "2", date: "2026-01-02", category: "absent"),
            ParentAttendanceRecord(id: "3", date: "2026-01-03", code: "T"),
        ]
        let summary = ParentLogic.attendanceSummary(records)
        XCTAssertEqual(summary.present, 1)
        XCTAssertEqual(summary.absent, 1)
        XCTAssertEqual(summary.tardy, 1)
    }
}
