import XCTest
@testable import Lextures

final class ContentToolPack2LogicTests: XCTestCase {
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
            .appendingPathComponent("mobile/fixtures/content-tools/pack2-logic.json")
        if FileManager.default.fileExists(atPath: direct.path) { return direct }
        var dir = URL(fileURLWithPath: FileManager.default.currentDirectoryPath)
        for _ in 0 ..< 8 {
            let candidate = dir.appendingPathComponent("clients/mobile/fixtures/content-tools/pack2-logic.json")
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

    func testAllowlistMatchesFixture() throws {
        let cases = try objects(try object(try fixtureRoot()["allowlist"])["cases"])
        for item in cases {
            let allowlist = Set(try XCTUnwrap(item["allowlist"] as? [String]))
            let toolId = try XCTUnwrap(item["toolId"] as? String)
            let expected = try XCTUnwrap(item["expected"] as? Bool)
            XCTAssertEqual(
                ContentToolPack2Logic.isClientAllowlisted(toolId, allowlist: allowlist),
                expected,
                item["name"] as? String ?? toolId
            )
        }
    }

    func testOfflineQueueNeverQueues() throws {
        for item in try objects(try object(try fixtureRoot()["offlineQueue"])["cases"]) {
            let toolId = try XCTUnwrap(item["toolId"] as? String)
            let action = try XCTUnwrap(item["action"] as? String)
            XCTAssertEqual(
                ContentToolPack2Logic.canQueueActionOffline(toolId: toolId, action: action),
                try XCTUnwrap(item["expected"] as? Bool),
                "\(toolId)/\(action)"
            )
        }
    }

    func testDraftLifecycleMatchesFixture() throws {
        let root = try object(try fixtureRoot()["draftLifecycle"])
        for item in try objects(root["cases"]) {
            let event = ContentToolPack2Logic.draftEventAfterAction(
                success: try XCTUnwrap(item["success"] as? Bool),
                preserveInput: try XCTUnwrap(item["preserveInput"] as? Bool)
            )
            XCTAssertEqual(event.rawValue, try XCTUnwrap(item["expected"] as? String), item["name"] as? String ?? "")
        }
        for item in try objects(root["keys"]) {
            XCTAssertEqual(
                ContentToolPack2Logic.draftStorageKey(
                    instanceId: try XCTUnwrap(item["instanceId"] as? String),
                    slot: try XCTUnwrap(item["slot"] as? String)
                ),
                try XCTUnwrap(item["expected"] as? String)
            )
        }
    }

    func testConsentGatingMatchesFixture() throws {
        for item in try objects(try object(try fixtureRoot()["consentGating"])["cases"]) {
            let mode = item["disclosureMode"] as? String
            let decision = item["decision"] as? String
            let fetched = try XCTUnwrap(item["consentFetched"] as? Bool)
            XCTAssertEqual(
                ContentToolPack2Logic.composerAIAllowed(
                    disclosureMode: mode,
                    decision: decision,
                    consentFetched: fetched
                ),
                try XCTUnwrap(item["composerAllowed"] as? Bool),
                item["name"] as? String ?? ""
            )
            XCTAssertEqual(
                ContentToolPack2Logic.shouldShowAIDisclosure(
                    disclosureMode: mode,
                    decision: decision,
                    consentFetched: fetched
                ),
                try XCTUnwrap(item["showDisclosure"] as? Bool),
                item["name"] as? String ?? ""
            )
        }
    }

    func testErrorClassificationMatchesFixture() throws {
        for item in try objects(try object(try fixtureRoot()["errorClassification"])["cases"]) {
            let code = item["code"] as? String
            let classified = ContentToolPack2Logic.classifyAIError(code: code)
            XCTAssertEqual(classified.rawValue, try XCTUnwrap(item["expected"] as? String), code ?? "nil")
            XCTAssertEqual(
                ContentToolPack2Logic.plainLanguageMessageKey(for: code),
                try XCTUnwrap(item["messageKey"] as? String)
            )
        }
    }

    func testLengthGuidanceMatchesFixture() throws {
        for item in try objects(try object(try fixtureRoot()["lengthGuidance"])["cases"]) {
            XCTAssertEqual(
                ContentToolPack2Logic.lengthGuidanceOK(
                    text: try XCTUnwrap(item["text"] as? String),
                    minWords: try XCTUnwrap(item["minWords"] as? Int),
                    maxWords: try XCTUnwrap(item["maxWords"] as? Int)
                ),
                try XCTUnwrap(item["expected"] as? Bool)
            )
        }
    }

    func testPaginationMatchesFixture() throws {
        for item in try objects(try object(try fixtureRoot()["pagination"])["cases"]) {
            let next = ContentToolPack2Logic.nextPage(
                currentPage: try XCTUnwrap(item["currentPage"] as? Int),
                pageSize: try XCTUnwrap(item["pageSize"] as? Int),
                total: item["total"] as? Int
            )
            if item["expectedNext"] is NSNull || item["expectedNext"] == nil {
                XCTAssertNil(next)
            } else {
                XCTAssertEqual(next, item["expectedNext"] as? Int)
            }
        }
    }

    func testDiscussionControlsMatchFixture() throws {
        for item in try objects(try object(try fixtureRoot()["discussionControls"])["cases"]) {
            let controls = ContentToolPack2Logic.discussionControls(
                isOwn: try XCTUnwrap(item["isOwn"] as? Bool),
                canEditFlag: try XCTUnwrap(item["canEditFlag"] as? Bool),
                canDeleteFlag: try XCTUnwrap(item["canDeleteFlag"] as? Bool),
                allowReplies: try XCTUnwrap(item["allowReplies"] as? Bool),
                viewerCanEndorse: try XCTUnwrap(item["viewerCanEndorse"] as? Bool),
                viewerCanModerate: try XCTUnwrap(item["viewerCanModerate"] as? Bool),
                readOnly: try XCTUnwrap(item["readOnly"] as? Bool),
                removed: try XCTUnwrap(item["removed"] as? Bool)
            )
            let expected = try object(item["expected"])
            XCTAssertEqual(controls.canEdit, try XCTUnwrap(expected["canEdit"] as? Bool), item["name"] as? String ?? "")
            XCTAssertEqual(controls.canDelete, try XCTUnwrap(expected["canDelete"] as? Bool))
            XCTAssertEqual(controls.canEndorse, try XCTUnwrap(expected["canEndorse"] as? Bool))
            XCTAssertEqual(controls.canModerate, try XCTUnwrap(expected["canModerate"] as? Bool))
            XCTAssertEqual(controls.canUpvote, try XCTUnwrap(expected["canUpvote"] as? Bool))
            XCTAssertEqual(controls.canReport, try XCTUnwrap(expected["canReport"] as? Bool))
            XCTAssertEqual(controls.canReply, try XCTUnwrap(expected["canReply"] as? Bool))
        }
    }

    func testTombstoneMatchesFixture() throws {
        for item in try objects(try object(try fixtureRoot()["tombstone"])["cases"]) {
            XCTAssertEqual(
                ContentToolPack2Logic.shouldRenderTombstone(
                    removed: try XCTUnwrap(item["removed"] as? Bool),
                    tombstone: try XCTUnwrap(item["tombstone"] as? Bool),
                    moderationState: item["moderationState"] as? String
                ),
                try XCTUnwrap(item["expected"] as? Bool)
            )
        }
    }

    func testConflictPolicyAndComposerSend() throws {
        for item in try objects(try object(try fixtureRoot()["conflictPolicy"])["cases"]) {
            let toolId = try XCTUnwrap(item["toolId"] as? String)
            let expected = try XCTUnwrap(item["expected"] as? String)
            XCTAssertEqual(ContentToolPack2Logic.conflictPolicy(for: toolId).rawValue, expected)
            XCTAssertEqual(ContentToolHostLogic.conflictPolicyForTool(toolId).rawValue, expected)
        }
        for item in try objects(try object(try fixtureRoot()["composerSend"])["cases"]) {
            XCTAssertEqual(
                ContentToolPack2Logic.composerSendEnabled(
                    text: try XCTUnwrap(item["text"] as? String),
                    readOnly: try XCTUnwrap(item["readOnly"] as? Bool),
                    online: try XCTUnwrap(item["online"] as? Bool),
                    busy: try XCTUnwrap(item["busy"] as? Bool),
                    consentAllowed: try XCTUnwrap(item["consentAllowed"] as? Bool)
                ),
                try XCTUnwrap(item["expected"] as? Bool)
            )
        }
    }

    func testRegisteredNativeIncludesPack2() {
        let ids = ContentToolHostLogic.registeredNativeToolIds()
        XCTAssertTrue(ids.contains("noop_probe"))
        for toolId in ContentToolPack2Logic.pack2ToolIds {
            XCTAssertTrue(ids.contains(toolId), toolId)
            XCTAssertFalse(ContentToolHostLogic.shouldShowUnsupportedPlaceholder(toolId: toolId, contract: 1))
        }
    }

    func testDraftStoreRoundTrip() {
        let key = ContentToolPack2Logic.draftStorageKey(instanceId: "test-inst", slot: "composer")
        ContentToolDraftStore.clear(key: key)
        XCTAssertEqual(ContentToolDraftStore.load(key: key), "")
        ContentToolDraftStore.save(key: key, text: "half typed")
        XCTAssertEqual(ContentToolDraftStore.load(key: key), "half typed")
        ContentToolDraftStore.clear(key: key)
        XCTAssertEqual(ContentToolDraftStore.load(key: key), "")
    }
}
