import XCTest
@testable import Lextures

final class InsightsLogicTests: XCTestCase {
    func testFormatHoursRoundsSmallValues() {
        XCTAssertEqual(InsightsLogic.formatHours(0.05), "0")
        XCTAssertEqual(InsightsLogic.formatHours(9.4), "9.4")
        XCTAssertEqual(InsightsLogic.formatHours(12), "12")
    }

    func testGoalProgressPercentCapsAt100() {
        XCTAssertEqual(InsightsLogic.goalProgressPercent(progressHours: 5, goalHours: 4), 100)
        XCTAssertEqual(InsightsLogic.goalProgressPercent(progressHours: 2, goalHours: 4), 50)
        XCTAssertNil(InsightsLogic.goalProgressPercent(progressHours: 2, goalHours: nil))
    }

    func testModuleCompletionPercentCountsItems() {
        let snapshot = ModulesProgressSnapshot(
            enrollmentId: "e1",
            modules: [
                ModuleLockState(
                    moduleId: "m1",
                    title: "Module 1",
                    sortOrder: 1,
                    locked: false,
                    complete: false,
                    reason: nil,
                    items: [
                        ItemLockState(itemId: "i1", locked: false, complete: true, reason: nil),
                        ItemLockState(itemId: "i2", locked: false, complete: false, reason: nil),
                    ]
                ),
            ]
        )
        XCTAssertEqual(InsightsLogic.moduleCompletionPercent(snapshot), 50)
    }

    func testJournalEntryValidEnforcesLength() {
        XCTAssertFalse(InsightsLogic.journalEntryValid("   "))
        XCTAssertTrue(InsightsLogic.journalEntryValid("Felt good today"))
        XCTAssertFalse(InsightsLogic.journalEntryValid(String(repeating: "a", count: 281)))
    }

    func testBarWidthPercentUsesMaxMinutes() {
        XCTAssertEqual(InsightsLogic.barWidthPercent(minutes: 30, maxMinutes: 60), 50)
        XCTAssertEqual(InsightsLogic.barWidthPercent(minutes: 90, maxMinutes: 60), 100)
    }
}