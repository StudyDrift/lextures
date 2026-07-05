import XCTest
@testable import Lextures

final class EvaluationLogicTests: XCTestCase {
    private let draftCourseCode = "CS101"
    private let draftWindowId = "win-1"
    private let draftDefaultsSuiteName = "com.lextures.evaluation-drafts"

    override func setUp() {
        super.setUp()
        EvaluationLogic.clearDraft(courseCode: draftCourseCode, windowId: draftWindowId)
        UserDefaults(suiteName: draftDefaultsSuiteName)?
            .removePersistentDomain(forName: draftDefaultsSuiteName)
    }

    override func tearDown() {
        EvaluationLogic.clearDraft(courseCode: draftCourseCode, windowId: draftWindowId)
        super.tearDown()
    }

    func testEvaluationsEnabledRequiresBothFlags() {
        var features = MobilePlatformFeatures()
        XCTAssertFalse(EvaluationLogic.evaluationsEnabled(features))
        features.ffCourseEvaluations = true
        XCTAssertTrue(EvaluationLogic.evaluationsEnabled(features))
        features.ffMobileCourseEvaluations = false
        XCTAssertFalse(EvaluationLogic.evaluationsEnabled(features))
    }

    func testMissingRequiredIndices() {
        let questions = [
            EvaluationQuestion(type: .rating, text: "Q1", options: nil, required: true),
            EvaluationQuestion(type: .openText, text: "Q2", options: nil, required: false),
            EvaluationQuestion(type: .multipleChoice, text: "Q3", options: ["A", "B"], required: true),
        ]
        let missing = EvaluationLogic.missingRequiredIndices(
            questions: questions,
            answers: ["1": "3"]
        )
        XCTAssertEqual(missing, [0, 2])
    }

    func testIsSubmitBlocked() {
        XCTAssertTrue(EvaluationLogic.isSubmitBlocked(status: nil))
        XCTAssertTrue(
            EvaluationLogic.isSubmitBlocked(
                status: EvaluationStatus(windowOpen: true, windowId: "w1", hasSubmitted: true, opensAt: nil, closesAt: nil, questions: nil)
            )
        )
        XCTAssertFalse(
            EvaluationLogic.isSubmitBlocked(
                status: EvaluationStatus(windowOpen: true, windowId: "w1", hasSubmitted: false, opensAt: nil, closesAt: nil, questions: nil)
            )
        )
    }

    func testShouldShowWorkspaceSectionForStudentWithOpenWindow() {
        let course = CourseSummary(
            id: "1",
            courseCode: "demo",
            title: "Demo",
            description: "",
            viewerEnrollmentRoles: ["student"]
        )
        var features = MobilePlatformFeatures()
        features.ffCourseEvaluations = true
        features.ffMobileCourseEvaluations = true
        let status = EvaluationStatus(windowOpen: true, windowId: "w1", hasSubmitted: false, opensAt: nil, closesAt: nil, questions: nil)
        XCTAssertTrue(EvaluationLogic.shouldShowWorkspaceSection(course: course, status: status, features: features))
    }

    func testDraftCacheKeyFormat() {
        XCTAssertEqual(
            EvaluationLogic.draftCacheKey(courseCode: draftCourseCode, windowId: draftWindowId),
            "evaluation:draft:CS101:win-1"
        )
    }
}
