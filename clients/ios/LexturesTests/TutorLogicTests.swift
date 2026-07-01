import XCTest
@testable import Lextures

final class TutorLogicTests: XCTestCase {
    func testParseStreamContentEvent() {
        let event = TutorLogic.parseStreamEvent("{\"type\":\"content\",\"text\":\"Hello\"}")
        XCTAssertEqual(event, .content("Hello"))
    }

    func testParseStreamDoneEvent() {
        let event = TutorLogic.parseStreamEvent(
            "{\"type\":\"done\",\"conversationId\":\"c1\",\"citations\":[]}"
        )
        XCTAssertEqual(event, .done(conversationId: "c1", messageId: nil, sessionId: nil, citations: []))
    }

    func testMessageWithContextPrefixesFirstMessage() {
        let result = TutorLogic.messageWithContext(
            "Explain step 3",
            itemTitle: "Photosynthesis",
            itemKind: "content_page",
            includeContext: true
        )
        XCTAssertTrue(result.contains("Photosynthesis"))
        XCTAssertTrue(result.contains("Explain step 3"))
    }

    func testMessageWithContextSkipsWhenDisabled() {
        XCTAssertEqual(
            TutorLogic.messageWithContext("Hi", itemTitle: "Page", itemKind: "quiz", includeContext: false),
            "Hi"
        )
    }

    func testShouldShowFabWhenCourseEnablesTutor() {
        var course = CourseSummary(
            id: "1",
            courseCode: "BIO-101",
            title: "Biology",
            description: ""
        )
        course.aiTutorEnabled = false
        XCTAssertFalse(TutorLogic.shouldShowFab(course: course))
        course.aiTutorEnabled = true
        XCTAssertTrue(TutorLogic.shouldShowFab(course: course))
    }

    func testGracefulHttpMessageBudgetExceeded() {
        let message = TutorLogic.gracefulHttpMessage(statusCode: 402, body: "BUDGET_EXCEEDED")
        XCTAssertFalse(message.isEmpty)
    }
}