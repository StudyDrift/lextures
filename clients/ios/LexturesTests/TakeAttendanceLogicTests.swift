import XCTest
@testable import Lextures

final class TakeAttendanceLogicTests: XCTestCase {
    private func record(id: String, name: String, status: String) -> AttendanceRecord {
        AttendanceRecord(studentUserId: id, displayName: name, status: status, recordedAt: nil)
    }

    func testMarkAllPresent() {
        let records = [
            record(id: "a", name: "Alex", status: "not_recorded"),
            record(id: "b", name: "Blair", status: "absent"),
        ]
        let draft = TakeAttendanceLogic.markAllPresent(records: records)
        XCTAssertEqual(draft["a"], "present")
        XCTAssertEqual(draft["b"], "present")
    }

    func testSummaryCounts() {
        let records = [
            record(id: "a", name: "Alex", status: "present"),
            record(id: "b", name: "Blair", status: "absent"),
            record(id: "c", name: "Casey", status: "tardy"),
            record(id: "d", name: "Dana", status: "excused"),
            record(id: "e", name: "Eden", status: "not_recorded"),
        ]
        let draft: [String: String] = ["e": "present"]
        let counts = TakeAttendanceLogic.summaryCounts(records: records, draft: draft)
        XCTAssertEqual(counts.present, 2)
        XCTAssertEqual(counts.absent, 1)
        XCTAssertEqual(counts.tardy, 1)
        XCTAssertEqual(counts.excused, 1)
        XCTAssertEqual(counts.notRecorded, 0)
    }

    func testFindTodaysOpenRollCallSession() {
        let sessions = [
            AttendanceSession(
                id: "1",
                title: "Old",
                collectionMethod: "roll_call",
                sessionDate: "2020-01-01",
                status: "open"
            ),
            AttendanceSession(
                id: "2",
                title: "Today",
                collectionMethod: "roll_call",
                sessionDate: TakeAttendanceLogic.todayDateString(),
                status: "open"
            ),
            AttendanceSession(
                id: "3",
                title: "Self",
                collectionMethod: "self_report",
                sessionDate: TakeAttendanceLogic.todayDateString(),
                status: "open"
            ),
        ]
        XCTAssertEqual(TakeAttendanceLogic.findTodaysOpenRollCallSession(sessions: sessions)?.id, "2")
    }

    func testShouldTakeSession() {
        let rollCall = AttendanceSession(
            id: "1",
            title: nil,
            collectionMethod: "roll_call",
            sessionDate: nil,
            status: "open"
        )
        let selfReport = AttendanceSession(
            id: "2",
            title: nil,
            collectionMethod: "self_report",
            sessionDate: nil,
            status: "open"
        )
        XCTAssertTrue(TakeAttendanceLogic.shouldTakeSession(rollCall, isStaff: true))
        XCTAssertFalse(TakeAttendanceLogic.shouldTakeSession(rollCall, isStaff: false))
        XCTAssertFalse(TakeAttendanceLogic.shouldTakeSession(selfReport, isStaff: true))
    }

    func testRecordsPayload() {
        let records = [record(id: "a", name: "Alex", status: "not_recorded")]
        let payload = TakeAttendanceLogic.recordsPayload(records: records, draft: ["a": "absent"])
        XCTAssertEqual(payload.count, 1)
        XCTAssertEqual(payload[0].studentUserId, "a")
        XCTAssertEqual(payload[0].status, "absent")
        XCTAssertEqual(payload[0].source, "instructor")
    }
}