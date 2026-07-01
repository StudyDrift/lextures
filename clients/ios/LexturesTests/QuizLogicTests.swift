import XCTest
@testable import Lextures

final class QuizLogicTests: XCTestCase {
    func testVisibleChoicesTrimsEmpty() {
        let question = QuizQuestion(
            id: "q1",
            prompt: "Pick one",
            questionType: "multiple_choice",
            choices: [" A ", "", "B"],
            choiceIds: nil,
            typeConfig: nil,
            correctChoiceIndex: nil,
            multipleAnswer: nil,
            answerWithImage: nil,
            required: nil,
            points: nil,
            estimatedMinutes: nil
        )
        XCTAssertEqual(QuizLogic.visibleChoices(question), ["A", "B"])
    }

    func testBuildResponseItemMultipleChoice() {
        let question = QuizQuestion(
            id: "q1",
            prompt: "Pick",
            questionType: "multiple_choice",
            choices: ["A", "B"],
            choiceIds: nil,
            typeConfig: nil,
            correctChoiceIndex: nil,
            multipleAnswer: false,
            answerWithImage: nil,
            required: nil,
            points: nil,
            estimatedMinutes: nil
        )
        let item = QuizLogic.buildResponseItem(question: question, answer: QuizAnswerState(choice: 1))
        XCTAssertEqual(item.questionId, "q1")
        XCTAssertEqual(item.selectedChoiceIndex, 1)
    }

    func testIsAnsweredShortAnswer() {
        let question = QuizQuestion(
            id: "q1",
            prompt: "Explain",
            questionType: "short_answer",
            choices: nil,
            choiceIds: nil,
            typeConfig: nil,
            correctChoiceIndex: nil,
            multipleAnswer: nil,
            answerWithImage: nil,
            required: nil,
            points: nil,
            estimatedMinutes: nil
        )
        XCTAssertFalse(QuizLogic.isAnswered(question: question, answer: nil))
        XCTAssertTrue(QuizLogic.isAnswered(question: question, answer: QuizAnswerState(text: "hello")))
        XCTAssertFalse(QuizLogic.isAnswered(question: question, answer: QuizAnswerState(text: "   ")))
    }

    func testServerLockdownModes() {
        XCTAssertTrue(QuizLogic.isServerLockdown("one_at_a_time"))
        XCTAssertTrue(QuizLogic.isServerLockdown("kiosk"))
        XCTAssertFalse(QuizLogic.isServerLockdown("standard"))
    }

    func testFormatTimer() {
        XCTAssertEqual(QuizLogic.formatTimer(125), "2:05")
        XCTAssertEqual(QuizLogic.formatTimer(59), "0:59")
    }

    func testMatchingPairsPayload() {
        let question = QuizQuestion(
            id: "q1",
            prompt: "Match",
            questionType: "matching",
            choices: nil,
            choiceIds: nil,
            typeConfig: QuizTypeConfig(
                items: nil,
                pairs: [
                    QuizMatchingPairConfig(leftId: "l1", rightId: "r1", left: "Cat", right: "Meow"),
                    QuizMatchingPairConfig(leftId: "l2", rightId: "r2", left: "Dog", right: "Bark"),
                ],
                starterCode: nil,
                language: nil
            ),
            correctChoiceIndex: nil,
            multipleAnswer: nil,
            answerWithImage: nil,
            required: nil,
            points: nil,
            estimatedMinutes: nil
        )
        let answer = QuizAnswerState(matching: ["l1": "Meow"])
        let payload = QuizLogic.buildMatchingPairsPayload(question: question, answer: answer)
        XCTAssertEqual(payload.count, 1)
        XCTAssertEqual(payload[0].leftId, "l1")
        XCTAssertEqual(payload[0].rightId, "r1")
    }
}
