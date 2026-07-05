import XCTest
@testable import Lextures

final class EvaluationLogicTests: XCTestCase {
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
            EvaluationLogic.draftCacheKey(courseCode: "CS101", windowId: "win-1"),
            "evaluation:draft:CS101:win-1"
        )
    }

    func testEvaluationTodoTitleIsLocalized() {
        XCTAssertEqual(L.text("mobile.evaluations.todoTitle"), "Course evaluation")
    }

    func testCollectEvaluationTodos() {
        let course = CourseSummary(
            id: "1",
            courseCode: "BIO101",
            title: "Biology",
            description: "",
            viewerEnrollmentRoles: ["student"],
            calendarEnabled: true
        )
        let status = EvaluationStatus(
            windowOpen: true,
            windowId: "win-1",
            hasSubmitted: false,
            opensAt: nil,
            closesAt: "2026-07-10T23:59:00Z",
            questions: nil
        )
        let todos = PlannerLogic.collectTodos(
            studentCourses: [course],
            structureByCourseCode: [:],
            notebookTasks: [],
            gradesByCourseCode: [:],
            evaluationStatusByCourseCode: ["BIO101": status]
        )
        XCTAssertEqual(todos.count, 1)
        XCTAssertEqual(todos[0].key, PlannerLogic.evaluationTodoKey(courseCode: "BIO101", windowId: "win-1"))
        XCTAssertEqual(todos[0].kind.rawValue, "evaluation")
        XCTAssertEqual(todos[0].evaluationWindowId, "win-1")
        XCTAssertEqual(todos[0].title, L.text("mobile.evaluations.todoTitle"))
    }
}
