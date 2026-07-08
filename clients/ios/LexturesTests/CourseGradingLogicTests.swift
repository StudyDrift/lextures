import XCTest
@testable import Lextures

final class CourseGradingLogicTests: XCTestCase {
    func testWeightTotalSumsGroups() {
        let groups = [
            CourseGradingLogic.EditableAssignmentGroup(clientKey: "a", id: "1", name: "Exams", sortOrder: 0, weightPercent: "50"),
            CourseGradingLogic.EditableAssignmentGroup(clientKey: "b", id: "2", name: "Homework", sortOrder: 1, weightPercent: "20"),
        ]
        XCTAssertEqual(CourseGradingLogic.weightTotal(groups), 70)
    }

    func testHasWeightWarningWhenNotOneHundred() {
        XCTAssertTrue(CourseGradingLogic.hasWeightWarning(80))
        XCTAssertFalse(CourseGradingLogic.hasWeightWarning(100))
        XCTAssertFalse(CourseGradingLogic.hasWeightWarning(99.995))
    }

    func testValidateBandsRejectsOutOfOrder() {
        let bands = [
            CourseGradingLogic.GradingSchemeBand(clientKey: "a", label: "A", minPct: "50", gpa: "4"),
            CourseGradingLogic.GradingSchemeBand(clientKey: "b", label: "B", minPct: "40", gpa: "3"),
        ]
        XCTAssertNotNil(CourseGradingLogic.validateBands(bands))
    }

    func testValidateBandsRequiresLowestZero() {
        let bands = [
            CourseGradingLogic.GradingSchemeBand(clientKey: "a", label: "A", minPct: "90", gpa: "4"),
            CourseGradingLogic.GradingSchemeBand(clientKey: "b", label: "B", minPct: "10", gpa: "1"),
        ]
        XCTAssertNotNil(CourseGradingLogic.validateBands(bands))
    }

    func testValidateBandsAcceptsValidScale() {
        let bands = CourseGradingLogic.defaultBands()
        XCTAssertNil(CourseGradingLogic.validateBands(bands))
    }

    func testGradableRowsSkipsModules() {
        let items = [
            CourseStructureItem(
                id: "m1", sortOrder: 0, kind: "module", title: "Week 1", parentId: nil,
                published: true, dueAt: nil, pointsWorth: nil, pointsPossible: nil, archived: nil, updatedAt: nil
            ),
            CourseStructureItem(
                id: "a1", sortOrder: 1, kind: "assignment", title: "Essay", parentId: "m1",
                published: true, dueAt: nil, pointsWorth: nil, pointsPossible: nil, archived: nil, updatedAt: nil
            ),
        ]
        let rows = CourseGradingLogic.gradableRows(from: items)
        XCTAssertEqual(rows.count, 1)
        XCTAssertEqual(rows.first?.moduleTitle, "Week 1")
    }

    func testIsSettingsDirtyDetectsWeightChange() {
        var baseline = CourseGradingLogic.FormBaseline(
            gradingScale: "letter_standard",
            groups: CourseGradingLogic.defaultGroups(),
            schemeType: "points",
            bands: CourseGradingLogic.defaultBands(),
            passMinPct: "60",
            completeMinPct: "50"
        )
        var current = baseline
        XCTAssertFalse(CourseGradingLogic.isSettingsDirty(current: current, baseline: baseline))
        current.groups[0].weightPercent = "80"
        XCTAssertTrue(CourseGradingLogic.isSettingsDirty(current: current, baseline: baseline))
    }

    func testBuildPutSettingsBodyUsesTrimmedNames() {
        let form = CourseGradingLogic.FormBaseline(
            gradingScale: "percent",
            groups: [
                .init(clientKey: "g1", id: nil, name: "  Labs  ", sortOrder: 0, weightPercent: "25"),
            ],
            schemeType: "points",
            bands: CourseGradingLogic.defaultBands(),
            passMinPct: "60",
            completeMinPct: "50"
        )
        let body = CourseGradingLogic.buildPutSettingsBody(form: form)
        XCTAssertEqual(body.gradingScale, "percent")
        XCTAssertEqual(body.assignmentGroups.first?.name, "Labs")
        XCTAssertEqual(body.assignmentGroups.first?.weightPercent, 25)
    }
}