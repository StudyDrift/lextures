import XCTest
@testable import Lextures

final class OnboardingModelsTests: XCTestCase {
    func testDecodesOnboardingStatus() throws {
        let json = """
        {"completed":false,"step":2,"shouldShowFlow":true}
        """
        let status = try JSONDecoder().decode(OnboardingStatus.self, from: Data(json.utf8))
        XCTAssertFalse(status.completed)
        XCTAssertEqual(status.step, 2)
        XCTAssertTrue(status.shouldShowFlow)
    }

    func testDecodesDiagnosticQuestions() throws {
        let json = """
        {"questions":[{"id":"q1","prompt":"Which keyword defines a function?","choices":["func","def","fn"]}]}
        """
        let response = try JSONDecoder().decode(DiagnosticQuestionsResponse.self, from: Data(json.utf8))
        XCTAssertEqual(response.questions.count, 1)
        XCTAssertEqual(response.questions[0].id, "q1")
        XCTAssertEqual(response.questions[0].choices.count, 3)
    }

    func testDecodesLearnerGoalsEnvelope() throws {
        let json = """
        {"goals":{"id":"g1","userId":"u1","topic":"python","dailyMinutes":20,\
        "priorKnowledgeLevel":"beginner","diagnosticSkipped":true,"onboardingStep":6,\
        "onboardingCompleted":true,"reminderOptIn":false,"recommendedCourseCode":"PY101",\
        "recommendedCourseTitle":"Python Basics"}}
        """
        let envelope = try JSONDecoder().decode(GoalsEnvelope.self, from: Data(json.utf8))
        XCTAssertEqual(envelope.goals.recommendedCourseCode, "PY101")
        XCTAssertEqual(envelope.goals.recommendedCourseTitle, "Python Basics")
        XCTAssertTrue(envelope.goals.onboardingCompleted)
    }

    func testOnboardingTopicsIncludePython() {
        XCTAssertTrue(OnboardingTopic.all.contains { $0.id == "python" })
    }
}
