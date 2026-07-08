import XCTest
@testable import Lextures

final class CourseOutcomesLogicTests: XCTestCase {
    func testGradableOptionsSkipsArchivedAndModules() {
        let items = [
            CourseStructureItem(
                id: "m1", sortOrder: 0, kind: "module", title: "Week 1", parentId: nil,
                published: true, dueAt: nil, pointsWorth: nil, pointsPossible: nil, archived: nil, updatedAt: nil
            ),
            CourseStructureItem(
                id: "a1", sortOrder: 1, kind: "assignment", title: "Essay", parentId: "m1",
                published: true, dueAt: nil, pointsWorth: nil, pointsPossible: nil, archived: nil, updatedAt: nil
            ),
            CourseStructureItem(
                id: "q1", sortOrder: 2, kind: "quiz", title: "Quiz 1", parentId: "m1",
                published: true, dueAt: nil, pointsWorth: nil, pointsPossible: nil, archived: true, updatedAt: nil
            ),
        ]
        let options = CourseOutcomesLogic.gradableOptions(from: items)
        XCTAssertEqual(options.count, 1)
        XCTAssertEqual(options.first?.label, "Week 1 — Essay")
    }

    func testDirtyOutcomeIdsDetectsTitleChange() {
        let outcomes = [
            CourseOutcome(
                id: "o1", title: "Original", description: "", sortOrder: 0,
                rollupAvgScorePercent: nil, links: []
            ),
        ]
        var drafts = CourseOutcomesLogic.drafts(from: outcomes)
        XCTAssertTrue(CourseOutcomesLogic.dirtyOutcomeIds(drafts: drafts, outcomes: outcomes).isEmpty)
        drafts["o1"] = .init(title: "Updated", description: "")
        XCTAssertEqual(CourseOutcomesLogic.dirtyOutcomeIds(drafts: drafts, outcomes: outcomes), ["o1"])
    }

    func testValidateCreateTitleRejectsEmpty() {
        XCTAssertEqual(CourseOutcomesLogic.validateCreateTitle("   "), .titleRequired)
        XCTAssertNil(CourseOutcomesLogic.validateCreateTitle("Analyze sources"))
    }

    func testTargetKindMapsQuizScopes() {
        XCTAssertEqual(CourseOutcomesLogic.targetKind(gradableKind: "assignment", quizScopeWhole: true), "assignment")
        XCTAssertEqual(CourseOutcomesLogic.targetKind(gradableKind: "quiz", quizScopeWhole: true), "quiz")
        XCTAssertEqual(CourseOutcomesLogic.targetKind(gradableKind: "quiz", quizScopeWhole: false), "quiz_question")
    }

    func testTruncatedPromptCollapsesWhitespace() {
        let long = String(repeating: "word ", count: 40)
        let result = CourseOutcomesLogic.truncatedPrompt("  \(long)  ")
        XCTAssertTrue(result.hasSuffix("…"))
        XCTAssertFalse(result.contains("\n"))
    }

    func testBuildAddLinkBodyIncludesLevels() {
        let body = CourseOutcomesLogic.buildAddLinkBody(
            structureItemId: "item-1",
            targetKind: "quiz",
            quizQuestionId: nil,
            measurementLevel: "summative",
            intensityLevel: "high"
        )
        XCTAssertEqual(body.structureItemId, "item-1")
        XCTAssertEqual(body.measurementLevel, "summative")
        XCTAssertEqual(body.intensityLevel, "high")
    }
}