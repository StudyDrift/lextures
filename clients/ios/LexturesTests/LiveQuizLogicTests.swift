import XCTest
@testable import Lextures

final class LiveQuizLogicTests: XCTestCase {
    func testNormalizeAndValidateJoinCode() {
        XCTAssertEqual(LiveQuizLogic.normalizeJoinCode("  ab12  "), "AB12")
        XCTAssertTrue(LiveQuizLogic.isValidJoinCode("AB12"))
        XCTAssertFalse(LiveQuizLogic.isValidJoinCode(""))
        XCTAssertFalse(LiveQuizLogic.isValidJoinCode("bad code"))
    }

    func testValidateNickname() {
        switch LiveQuizLogic.validateNickname("  Ada  ") {
        case .ok(let nick):
            XCTAssertEqual(nick, "Ada")
        case .invalid:
            XCTFail("expected ok")
        }
        if case .invalid(let reason) = LiveQuizLogic.validateNickname("") {
            XCTAssertEqual(reason, .empty)
        } else {
            XCTFail("expected empty")
        }
        if case .invalid(let reason) = LiveQuizLogic.validateNickname(String(repeating: "x", count: 25)) {
            XCTAssertEqual(reason, .tooLong)
        } else {
            XCTFail("expected tooLong")
        }
        if case .invalid(let reason) = LiveQuizLogic.validateNickname("bad@name") {
            XCTAssertEqual(reason, .charset)
        } else {
            XCTFail("expected charset")
        }
    }

    func testJoinErrorMapping() {
        XCTAssertEqual(LiveQuizLogic.joinErrorReason(status: 404, body: ""), .notFound)
        XCTAssertEqual(LiveQuizLogic.joinErrorReason(status: 429, body: ""), .rateLimited)
        XCTAssertEqual(LiveQuizLogic.joinErrorReason(status: 409, body: "nickname taken"), .nicknameTaken)
        XCTAssertEqual(LiveQuizLogic.joinErrorReason(status: 409, body: "already connected"), .oneSession)
        XCTAssertEqual(LiveQuizLogic.joinErrorReason(status: 403, body: "lobby locked"), .lobbyLocked)
        XCTAssertEqual(LiveQuizLogic.joinErrorReason(status: 400, body: "game ended"), .gameEnded)
        XCTAssertEqual(LiveQuizLogic.joinErrorReason(status: 400, body: "isn't allowed"), .nicknameDenied)
    }

    func testWorkspaceSectionGate() {
        var course = CourseSummary(
            id: "1",
            courseCode: "C1",
            title: "Course",
            description: "",
            interactiveQuizzesEnabled: true
        )
        var features = MobilePlatformFeatures()
        features.ffMobileLiveQuiz = false
        XCTAssertFalse(LiveQuizLogic.shouldShowWorkspaceSection(course: course, features: features))
        features.ffMobileLiveQuiz = true
        XCTAssertTrue(LiveQuizLogic.shouldShowWorkspaceSection(course: course, features: features))
        course.interactiveQuizzesEnabled = false
        XCTAssertFalse(LiveQuizLogic.shouldShowWorkspaceSection(course: course, features: features))
    }

    func testPlayerSessionStoreRoundTrip() {
        let session = LiveQuizPlayerSession(
            gameId: "g1",
            courseCode: "C1",
            playerId: "p1",
            playerToken: "tok",
            nickname: "Ada",
            joinCode: "AB12"
        )
        LiveQuizPlayerSessionStore.save(session)
        XCTAssertEqual(LiveQuizPlayerSessionStore.load(gameId: "g1"), session)
        LiveQuizPlayerSessionStore.clear(gameId: "g1")
        XCTAssertNil(LiveQuizPlayerSessionStore.load(gameId: "g1"))
    }
}

final class LiveGameLogicTests: XCTestCase {
    func testAnswerPayloadSerialization() throws {
        let mc = try LiveGameLogic.AnswerPayload.optionId("b").jsonData()
        let obj = try XCTUnwrap(JSONSerialization.jsonObject(with: mc) as? [String: Any])
        XCTAssertEqual(obj["optionId"] as? String, "b")

        let multi = LiveGameLogic.AnswerPayload.optionIds(["a", "c"]).jsonObject()
        XCTAssertEqual(multi["optionIds"] as? [String], ["a", "c"])

        let typed = LiveGameLogic.AnswerPayload.text("paris").jsonObject()
        XCTAssertEqual(typed["text"] as? String, "paris")

        let numeric = LiveGameLogic.AnswerPayload.value(42).jsonObject()
        XCTAssertEqual(numeric["value"] as? Double, 42)

        let order = LiveGameLogic.AnswerPayload.order(["1", "2"]).jsonObject()
        XCTAssertEqual(order["order"] as? [String], ["1", "2"])
    }

    func testBuildAnswerPerType() {
        XCTAssertEqual(
            LiveGameLogic.buildAnswer(
                questionType: .mcSingle,
                selectedOptionId: "a",
                selectedOptionIds: [],
                text: nil,
                numeric: nil,
                order: nil
            ),
            .optionId("a")
        )
        XCTAssertNil(
            LiveGameLogic.buildAnswer(
                questionType: .mcMultiple,
                selectedOptionId: nil,
                selectedOptionIds: [],
                text: nil,
                numeric: nil,
                order: nil
            )
        )
        XCTAssertEqual(
            LiveGameLogic.buildAnswer(
                questionType: .typeAnswer,
                selectedOptionId: nil,
                selectedOptionIds: [],
                text: " hi ",
                numeric: nil,
                order: nil
            ),
            .text("hi")
        )
        XCTAssertEqual(
            LiveGameLogic.buildAnswer(
                questionType: .numeric,
                selectedOptionId: nil,
                selectedOptionIds: [],
                text: nil,
                numeric: 3.5,
                order: nil
            ),
            .value(3.5)
        )
    }

    func testPlaySurfaceAndSubmitGate() {
        XCTAssertEqual(
            LiveGameLogic.playSurface(for: .lobby, conn: .connected),
            .lobby
        )
        XCTAssertEqual(
            LiveGameLogic.playSurface(for: .questionOpen, conn: .connected),
            .question
        )
        XCTAssertEqual(
            LiveGameLogic.playSurface(for: .questionOpen, conn: .kicked),
            .kicked
        )
        XCTAssertTrue(
            LiveGameLogic.canSubmitAnswer(phase: .questionOpen, hasAnswered: false, conn: .connected)
        )
        XCTAssertFalse(
            LiveGameLogic.canSubmitAnswer(phase: .questionLocked, hasAnswered: false, conn: .connected)
        )
        XCTAssertFalse(
            LiveGameLogic.canSubmitAnswer(phase: .questionOpen, hasAnswered: true, conn: .connected)
        )
    }

    func testAuthHandshakeAndReconnectDelay() {
        let guest = LiveGameLogic.authHandshake(authToken: nil, role: .player, playerToken: "pt")
        XCTAssertEqual(guest["authToken"] as? String, "")
        XCTAssertEqual(guest["role"] as? String, "player")
        XCTAssertEqual(guest["playerToken"] as? String, "pt")
        XCTAssertEqual(LiveGameLogic.reconnectDelayMs(retry: 0), 500)
        XCTAssertEqual(LiveGameLogic.reconnectDelayMs(retry: 4), 8000)
        XCTAssertEqual(LiveGameLogic.reconnectDelayMs(retry: 10), 8000)
    }

    func testParseInboundMessages() throws {
        let stateJSON = """
        {"type":"state","seq":2,"gameId":"g1","phase":"question_open","status":"running","questionIndex":0,"joinCode":"AB12","kitTitle":"Demo","pacing":"manual","players":[],"questionCount":1}
        """.data(using: .utf8)!
        if case .state(let frame) = LiveGameLogic.parseInbound(stateJSON) {
            XCTAssertEqual(frame.seq, 2)
            XCTAssertEqual(frame.phase, "question_open")
        } else {
            XCTFail("expected state")
        }

        let ackJSON = """
        {"type":"answer_ack","ok":true,"questionIndex":0,"isCorrect":true,"points":100}
        """.data(using: .utf8)!
        if case .answerAck(let ack) = LiveGameLogic.parseInbound(ackJSON) {
            XCTAssertTrue(ack.ok)
            XCTAssertEqual(ack.points, 100)
        } else {
            XCTFail("expected ack")
        }

        let kickedJSON = #"{"type":"kicked"}"#.data(using: .utf8)!
        if case .kicked = LiveGameLogic.parseInbound(kickedJSON) {
            // ok
        } else {
            XCTFail("expected kicked")
        }
    }

    func testAnswerShapesAreStable() {
        XCTAssertEqual(LiveGameLogic.answerShapeName(index: 0), "triangle")
        XCTAssertEqual(LiveGameLogic.answerShapeName(index: 8), "triangle")
        XCTAssertFalse(LiveGameLogic.answerShapeLabel(index: 1).isEmpty)
    }

    func testShouldClearAnsweredIndex() {
        XCTAssertTrue(
            LiveGameLogic.shouldClearAnsweredIndex(
                previousQuestionIndex: 0,
                nextQuestionIndex: 1,
                nextPhase: .questionOpen
            )
        )
        XCTAssertFalse(
            LiveGameLogic.shouldClearAnsweredIndex(
                previousQuestionIndex: 1,
                nextQuestionIndex: 1,
                nextPhase: .questionOpen
            )
        )
    }
}
