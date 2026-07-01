import XCTest
@testable import Lextures

final class GradeCalculatorTests: XCTestCase {
    private let groups = [
        GradeCalculator.GroupWeight(id: "hw", weightPercent: 50, dropLowest: 0, dropHighest: 0, replaceLowestWithFinal: false),
        GradeCalculator.GroupWeight(id: "ex", weightPercent: 50, dropLowest: 0, dropHighest: 0, replaceLowestWithFinal: false),
    ]

    func testStraightPointsWhenNoWeights() {
        let cols = [
            GradeCalculator.ColumnForFinal(id: "a", maxPoints: 100, assignmentGroupId: nil),
            GradeCalculator.ColumnForFinal(id: "b", maxPoints: 50, assignmentGroupId: nil, dueAt: "2000-01-01T00:00:00Z"),
        ]
        let pct = GradeCalculator.computeCourseFinalPercent(
            columns: cols,
            gradesByItemId: ["a": "80", "b": "40"],
            assignmentGroups: []
        )
        XCTAssertEqual(pct ?? 0, (120.0 / 150.0) * 100, accuracy: 0.001)
    }

    func testDropLowestInGroup() {
        let cols = (0 ..< 4).map { index in
            GradeCalculator.ColumnForFinal(
                id: ["a", "b", "c", "d"][index],
                maxPoints: 100,
                assignmentGroupId: "g",
                dueAt: "2000-01-01T00:00:00Z"
            )
        }
        let pct = GradeCalculator.computeCourseFinalPercent(
            columns: cols,
            gradesByItemId: ["a": "60", "b": "70", "c": "80", "d": "90"],
            assignmentGroups: [
                GradeCalculator.GroupWeight(id: "g", weightPercent: 100, dropLowest: 1, dropHighest: 0, replaceLowestWithFinal: false),
            ]
        )
        XCTAssertEqual(pct ?? 0, 80, accuracy: 0.001)
    }

    func testWhatIfIncludesFutureOverride() {
        let future = "2099-01-01T00:00:00Z"
        let cols = [
            GradeCalculator.ColumnForFinal(id: "hw", maxPoints: 100, assignmentGroupId: "ex", dueAt: "2000-01-01T00:00:00Z"),
            GradeCalculator.ColumnForFinal(id: "final", maxPoints: 100, assignmentGroupId: "fi", dueAt: future),
        ]
        let groups = [
            GradeCalculator.GroupWeight(id: "ex", weightPercent: 40),
            GradeCalculator.GroupWeight(id: "fi", weightPercent: 60),
        ]
        let projected = GradeCalculator.computeWhatIfFinalPercent(
            columns: cols,
            actualGrades: ["hw": "80", "final": ""],
            assignmentGroups: groups,
            excusedByItemId: [:],
            whatIfOverrides: ["final": "90"],
            heldItemIds: []
        )
        XCTAssertEqual(projected ?? 0, 0.4 * 80 + 0.6 * 90, accuracy: 0.001)
    }

    func testHeldItemsExcludedFromWhatIfMerge() {
        let held: Set<String> = ["secret"]
        let merged = GradeCalculator.mergeGradesForWhatIf(
            actualGrades: ["secret": "99"],
            overrides: [:],
            heldItemIds: held
        )
        XCTAssertNil(merged["secret"])

        let withOverride = GradeCalculator.mergeGradesForWhatIf(
            actualGrades: ["secret": "99"],
            overrides: ["secret": "70"],
            heldItemIds: held
        )
        XCTAssertEqual(withOverride["secret"], "70")
    }

    func testBuildSectionsGroupsColumns() {
        let response = MyGradesResponse(
            columns: [
                GradeColumn(id: "1", kind: "assignment", title: "A1", assignmentGroupId: "hw"),
                GradeColumn(id: "2", kind: "assignment", title: "Other", assignmentGroupId: nil),
            ],
            assignmentGroups: [AssignmentGroup(id: "hw", name: "Homework", weightPercent: 20)]
        )
        let sections = GradesDisplayLogic.buildSections(from: response)
        XCTAssertEqual(sections.count, 2)
        XCTAssertEqual(sections[0].title, "Homework")
        XCTAssertEqual(sections[0].weightPercent, 20)
    }
}
