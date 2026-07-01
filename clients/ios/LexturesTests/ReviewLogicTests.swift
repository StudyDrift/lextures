import XCTest
@testable import Lextures

final class ReviewLogicTests: XCTestCase {
    func testFilterQueueByCourse() {
        let items = [
            makeItem(questionId: "q1", courseCode: "BIO101"),
            makeItem(questionId: "q2", courseCode: "CS101"),
        ]
        XCTAssertEqual(ReviewLogic.filterQueue(items, courseCode: "BIO101").map(\.questionId), ["q1"])
        XCTAssertEqual(ReviewLogic.filterQueue(items, courseCode: nil).count, 2)
    }

    func testFormatAnswerPreviewStringAndArray() {
        XCTAssertEqual(ReviewLogic.formatAnswerPreview(.string("Paris")), "Paris")
        XCTAssertEqual(
            ReviewLogic.formatAnswerPreview(.array([.string("A"), .string("B")])),
            "A, B"
        )
    }

    func testToQuizQuestionMultipleChoice() {
        let item = makeItem(
            questionId: "q1",
            courseCode: "BIO101",
            questionType: "multiple_choice",
            options: .object([
                "choices": .array([.string("One"), .string("Two")]),
            ])
        )
        let question = ReviewLogic.toQuizQuestion(item)
        XCTAssertEqual(question?.choices, ["One", "Two"])
    }

    func testIdempotencyKeyIsStableForSameInstant() {
        let date = Date(timeIntervalSince1970: 1_700_000_000)
        let key = ReviewLogic.idempotencyKey(questionId: "abc", ratedAt: date)
        XCTAssertEqual(key, "srs-review:abc:1700000000000")
    }

    private func makeItem(
        questionId: String,
        courseCode: String,
        questionType: String = "short_answer",
        options: JSONValue? = nil
    ) -> ReviewQueueItem {
        ReviewQueueItem(
            stateId: "state-\(questionId)",
            questionId: questionId,
            courseId: "course-\(courseCode)",
            courseCode: courseCode,
            courseTitle: courseCode,
            nextReviewAt: "2026-07-01T12:00:00Z",
            stem: "Prompt",
            questionType: questionType,
            options: options,
            correctAnswer: .string("Answer"),
            explanation: nil
        )
    }
}
