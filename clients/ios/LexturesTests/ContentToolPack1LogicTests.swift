import XCTest
@testable import Lextures

final class ContentToolPack1LogicTests: XCTestCase {
    private func fixtureRoot() throws -> [String: Any] {
        let data = try Data(contentsOf: fixtureURL())
        return try XCTUnwrap(JSONSerialization.jsonObject(with: data) as? [String: Any])
    }

    private func fixtureURL() -> URL {
        let thisFile = URL(fileURLWithPath: #filePath)
        let direct = thisFile
            .deletingLastPathComponent()
            .deletingLastPathComponent()
            .deletingLastPathComponent()
            .appendingPathComponent("mobile/fixtures/content-tools/pack1-logic.json")
        if FileManager.default.fileExists(atPath: direct.path) { return direct }
        var dir = URL(fileURLWithPath: FileManager.default.currentDirectoryPath)
        for _ in 0 ..< 8 {
            let candidate = dir.appendingPathComponent("clients/mobile/fixtures/content-tools/pack1-logic.json")
            if FileManager.default.fileExists(atPath: candidate.path) { return candidate }
            dir = dir.deletingLastPathComponent()
        }
        return direct
    }

    private func object(_ value: Any?, file: StaticString = #filePath, line: UInt = #line) throws -> [String: Any] {
        try XCTUnwrap(value as? [String: Any], file: file, line: line)
    }

    private func objects(_ value: Any?, file: StaticString = #filePath, line: UInt = #line) throws -> [[String: Any]] {
        try XCTUnwrap(value as? [[String: Any]], file: file, line: line)
    }

    private func answersMap(_ raw: [String: Any]) -> [String: JSONValue] {
        raw.mapValues { value in
            guard let obj = value as? [String: Any] else { return .null }
            var out: [String: JSONValue] = [:]
            if let attempts = obj["attempts"] as? [[String: Any]] {
                out["attempts"] = .array(attempts.map { _ in .object(["correct": .bool(false)]) })
            }
            return .object(out)
        }
    }

    func testAllowlistMatchesFixture() throws {
        let cases = try objects(try object(try fixtureRoot()["allowlist"])["cases"])
        for item in cases {
            let allowlist = Set(try XCTUnwrap(item["allowlist"] as? [String]))
            let toolId = try XCTUnwrap(item["toolId"] as? String)
            let expected = try XCTUnwrap(item["expected"] as? Bool)
            XCTAssertEqual(
                ContentToolPack1Logic.isClientAllowlisted(toolId, allowlist: allowlist),
                expected,
                item["name"] as? String ?? toolId
            )
        }
    }

    func testOfflineQueueRulesMatchFixture() throws {
        let root = try object(try fixtureRoot()["offlineQueue"])
        for item in try objects(root["cases"]) {
            let toolId = try XCTUnwrap(item["toolId"] as? String)
            let action = try XCTUnwrap(item["action"] as? String)
            let expected = try XCTUnwrap(item["expected"] as? Bool)
            XCTAssertEqual(
                ContentToolPack1Logic.canQueueActionOffline(toolId: toolId, action: action),
                expected,
                "\(toolId)/\(action)"
            )
        }
        let order = try object(root["order"])
        let input = try objects(order["input"])
        let pending: [ContentToolPack1Logic.PendingAction] = try input.map { item in
            let instanceId = try XCTUnwrap(item["instanceId"] as? String)
            let sequence = try XCTUnwrap(item["sequence"] as? NSNumber).int64Value
            return ContentToolPack1Logic.PendingAction(
                instanceId: instanceId,
                toolId: "flashcards",
                action: "rate",
                sequence: sequence,
                payloadJSON: "{}"
            )
        }
        let ordered = ContentToolPack1Logic.orderPendingActions(pending)
        XCTAssertEqual(ordered.map(\.instanceId), order["expectedInstanceOrder"] as? [String])
        XCTAssertEqual(ordered.map(\.sequence), (order["expectedSequenceOrder"] as? [Int])?.map { Int64($0) })
    }

    func testAttemptGatingMatchesFixture() throws {
        for item in try objects(try object(try fixtureRoot()["attempts"])["cases"]) {
            let answers = answersMap(try object(item["answers"]))
            let questionId = try XCTUnwrap(item["questionId"] as? String)
            let maxAttempts = item["maxAttempts"] as? Int
            let readOnly = try XCTUnwrap(item["readOnly"] as? Bool)
            let expected = try XCTUnwrap(item["expected"] as? Bool)
            XCTAssertEqual(
                ContentToolPack1Logic.canSubmit(
                    answers: answers,
                    questionId: questionId,
                    maxAttempts: maxAttempts,
                    readOnly: readOnly
                ),
                expected,
                item["name"] as? String ?? questionId
            )
        }
    }

    func testSequentialUnlockMatchesFixture() throws {
        let root = try object(try fixtureRoot()["sequential"])
        let questions = try XCTUnwrap(root["questions"] as? [String])
        for item in try objects(root["cases"]) {
            let answers = answersMap(try object(item["answers"]))
            XCTAssertEqual(
                ContentToolPack1Logic.isSequentiallyUnlocked(
                    questions: questions,
                    answers: answers,
                    questionId: try XCTUnwrap(item["questionId"] as? String),
                    sequential: try XCTUnwrap(item["sequential"] as? Bool)
                ),
                try XCTUnwrap(item["expected"] as? Bool),
                item["name"] as? String ?? ""
            )
        }
    }

    func testPredictRevealGatingMatchesFixture() throws {
        for item in try objects(try object(try fixtureRoot()["predictReveal"])["cases"]) {
            let committedAt = item["committedAt"] as? String
            let state: JSONValue = {
                guard let committedAt, !committedAt.isEmpty else { return .object([:]) }
                return .object(["committedAt": .string(committedAt)])
            }()
            let committed = ContentToolPack1Logic.isCommitted(state)
            let hasReveal = try XCTUnwrap(item["hasReveal"] as? Bool)
            XCTAssertEqual(
                ContentToolPack1Logic.canShowReveal(committed: committed, hasRevealPayload: hasReveal),
                try XCTUnwrap(item["canShow"] as? Bool)
            )
            XCTAssertEqual(
                ContentToolPack1Logic.canEditPrediction(committed: committed, readOnly: false),
                try XCTUnwrap(item["canEdit"] as? Bool)
            )
        }
    }

    func testClassPulsePollMatchesFixture() throws {
        let root = try object(try fixtureRoot()["classPulsePoll"])
        XCTAssertEqual(ContentToolPack1Logic.classPulsePollIntervalMs, try XCTUnwrap(root["baseMs"] as? Int))
        for item in try objects(root["visibility"]) {
            XCTAssertEqual(
                ContentToolPack1Logic.shouldPollAggregate(
                    visible: try XCTUnwrap(item["visible"] as? Bool),
                    hasVoted: try XCTUnwrap(item["hasVoted"] as? Bool)
                ),
                try XCTUnwrap(item["expected"] as? Bool)
            )
        }
        for item in try objects(root["backoff"]) {
            XCTAssertEqual(
                ContentToolPack1Logic.nextPollDelayMs(consecutiveFailures: try XCTUnwrap(item["failures"] as? Int)),
                try XCTUnwrap(item["expected"] as? Int)
            )
        }
    }

    func testFlashcardsRatingsAndReviewKeys() throws {
        let root = try object(try fixtureRoot()["flashcards"])
        for rating in try XCTUnwrap(root["validRatings"] as? [String]) {
            XCTAssertTrue(ContentToolPack1Logic.isValidRating(rating))
        }
        for rating in try XCTUnwrap(root["invalidRatings"] as? [String]) {
            // "AGAIN " keeps trailing space — ratings must match exactly after lowercasing.
            XCTAssertFalse(ContentToolPack1Logic.isValidRating(rating), rating)
        }
        XCTAssertEqual(
            ContentToolPack1Logic.reviewCacheKeysToInvalidate(),
            try XCTUnwrap(root["reviewCacheKeys"] as? [String])
        )
        XCTAssertFalse(ContentToolPack1Logic.shouldDoubleCountReviewSubmit(toolId: "flashcards"))
    }

    func testConflictPolicyAndUnknownPreservation() throws {
        for item in try objects(try object(try fixtureRoot()["conflictPolicy"])["cases"]) {
            let toolId = try XCTUnwrap(item["toolId"] as? String)
            let expected = try XCTUnwrap(item["expected"] as? String)
            XCTAssertEqual(ContentToolPack1Logic.conflictPolicy(for: toolId).rawValue, expected)
            XCTAssertEqual(ContentToolHostLogic.conflictPolicyForTool(toolId).rawValue, expected)
        }
        let preserve = try object(try fixtureRoot()["unknownFieldPreservation"])
        let base = (try object(preserve["base"])).mapValues { _ in JSONValue.string("x") }
        // rebuild with real shapes
        var baseMap: [String: JSONValue] = [
            "v": .number(1),
            "futureField": .string("keep-me"),
            "drafts": .object(["q1": .string("a")]),
        ]
        let merged = ContentToolPack1Logic.mergePreservingUnknown(
            base: baseMap,
            patch: ["drafts": .object(["q1": .string("b")])]
        )
        let expectedKeys = Set(try XCTUnwrap(preserve["expectedKeys"] as? [String]))
        XCTAssertEqual(Set(merged.keys), expectedKeys)
        if case .object(let drafts) = merged["drafts"], case .string(let draftValue) = drafts["q1"] {
            XCTAssertEqual(draftValue, "b")
        } else {
            XCTFail("drafts not merged")
        }
        XCTAssertFalse(base.isEmpty)
    }

    func testRegisteredNativeIncludesPack1() {
        let ids = ContentToolHostLogic.registeredNativeToolIds()
        XCTAssertTrue(ids.contains("noop_probe"))
        for toolId in ContentToolPack1Logic.pack1ToolIds {
            XCTAssertTrue(ids.contains(toolId), toolId)
        }
        XCTAssertFalse(ContentToolHostLogic.shouldShowUnsupportedPlaceholder(toolId: "inline_questions", contract: 1))
        // Pack-2 registers ask_questions when allowlisted (CT.M6).
        XCTAssertFalse(ContentToolHostLogic.shouldShowUnsupportedPlaceholder(toolId: "ask_questions", contract: 1))
    }

    func testQuestionsAtATimePagingHelpers() {
        XCTAssertNil(ContentToolPack1Logic.parseQuestionsAtATime(.string("all")))
        XCTAssertNil(ContentToolPack1Logic.parseQuestionsAtATime(nil))
        XCTAssertEqual(ContentToolPack1Logic.parseQuestionsAtATime(.number(1.0)), Optional(1))
        XCTAssertEqual(ContentToolPack1Logic.parseQuestionsAtATime(.number(2.0)), Optional(2))
        XCTAssertEqual(ContentToolPack1Logic.parseQuestionsAtATime(.string("2")), Optional(2))
        XCTAssertNil(ContentToolPack1Logic.parseQuestionsAtATime(.number(9.0)))

        let all = ContentToolPack1Logic.pageWindow(total: 3, pageSize: nil, pageIndex: 0)
        XCTAssertEqual(all.start, 0)
        XCTAssertEqual(all.end, 3)
        let page0 = ContentToolPack1Logic.pageWindow(total: 3, pageSize: 1, pageIndex: 0)
        XCTAssertEqual(page0.start, 0)
        XCTAssertEqual(page0.end, 1)
        let pageMid = ContentToolPack1Logic.pageWindow(total: 3, pageSize: 1, pageIndex: 1)
        XCTAssertEqual(pageMid.start, 1)
        XCTAssertEqual(pageMid.end, 2)
        let pageWide = ContentToolPack1Logic.pageWindow(total: 3, pageSize: 2, pageIndex: 0)
        XCTAssertEqual(pageWide.start, 0)
        XCTAssertEqual(pageWide.end, 2)
        let pageLast = ContentToolPack1Logic.pageWindow(total: 3, pageSize: 2, pageIndex: 1)
        XCTAssertEqual(pageLast.start, 2)
        XCTAssertEqual(pageLast.end, 3)
        // pageSize 1 → page index equals the item index (0-based).
        XCTAssertEqual(ContentToolPack1Logic.initialPageIndex(total: 3, pageSize: 1, firstIncompleteIndex: 2), 2)
        XCTAssertEqual(ContentToolPack1Logic.initialPageIndex(total: 3, pageSize: 2, firstIncompleteIndex: 1), 0)
    }
}
