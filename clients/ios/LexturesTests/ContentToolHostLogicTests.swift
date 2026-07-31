import XCTest
@testable import Lextures

final class ContentToolHostLogicTests: XCTestCase {
    private func fixtureRoot() throws -> [String: Any] {
        let url = fixtureURL()
        let data = try Data(contentsOf: url)
        guard let json = try JSONSerialization.jsonObject(with: data) as? [String: Any] else {
            XCTFail("invalid fixture JSON")
            return [:]
        }
        return json
    }

    private func fixtureURL() -> URL {
        let thisFile = URL(fileURLWithPath: #filePath)
        let direct = thisFile
            .deletingLastPathComponent() // LexturesTests
            .deletingLastPathComponent() // ios
            .deletingLastPathComponent() // clients
            .appendingPathComponent("mobile/fixtures/content-tools/host-logic.json")
        if FileManager.default.fileExists(atPath: direct.path) { return direct }
        var searchRoots = [
            URL(fileURLWithPath: FileManager.default.currentDirectoryPath),
            thisFile.deletingLastPathComponent(),
        ]
        for root in searchRoots {
            var dir = root
            for _ in 0 ..< 8 {
                for relative in [
                    "clients/mobile/fixtures/content-tools/host-logic.json",
                    "mobile/fixtures/content-tools/host-logic.json",
                ] {
                    let candidate = dir.appendingPathComponent(relative)
                    if FileManager.default.fileExists(atPath: candidate.path) { return candidate }
                }
                dir = dir.deletingLastPathComponent()
            }
        }
        return direct
    }

    func testClampDebounceMatchesFixture() throws {
        let root = try fixtureRoot()
        let debounce = root["debounce"] as! [String: Any]
        let cases = debounce["cases"] as! [[String: Any]]
        for item in cases {
            let expected = item["expected"] as! Int
            let actual: Int
            if item["input"] is NSNull || item["input"] == nil {
                actual = ContentToolHostLogic.clampDebounceMs(nil as Int?)
            } else if let intVal = item["input"] as? Int {
                actual = ContentToolHostLogic.clampDebounceMs(intVal)
            } else if let doubleVal = item["input"] as? Double {
                actual = ContentToolHostLogic.clampDebounceMs(doubleVal)
            } else {
                return XCTFail("unexpected input")
            }
            XCTAssertEqual(actual, expected)
        }
    }

    func testConflictPolicyMatchesFixture() throws {
        let root = try fixtureRoot()
        let cases = (root["conflictPolicy"] as! [String: Any])["cases"] as! [[String: Any]]
        for item in cases {
            let policy = ContentToolHostLogic.ConflictPolicy.from(item["policy"] as? String)
            let clientRaw = item["client"] as! [String: Any]
            let serverRaw = item["server"] as! [String: Any]
            let expectedRaw = item["expected"] as! [String: Any]
            let client = clientRaw.mapValues { JSONValue.number(($0 as! NSNumber).doubleValue) }
            let server = serverRaw.mapValues { JSONValue.number(($0 as! NSNumber).doubleValue) }
            let expected = expectedRaw.mapValues { JSONValue.number(($0 as! NSNumber).doubleValue) }
            let resolved = ContentToolHostLogic.resolveConflictState(
                policy: policy,
                client: client,
                server: server
            )
            XCTAssertEqual(resolved, expected)
        }
    }

    func testReadOnlyPrecedenceMatchesFixture() throws {
        let root = try fixtureRoot()
        let cases = root["readOnlyPrecedence"] as! [[String: Any]]
        for item in cases {
            let input = item["input"] as! [String: Any]
            let reason = ContentToolHostLogic.readOnlyReason(
                ContentToolHostLogic.ReadOnlyInput(
                    tombstone: input["tombstone"] as! Bool,
                    breakerOpen: input["breakerOpen"] as! Bool,
                    status: input["status"] as! String,
                    pastDue: input["pastDue"] as! Bool,
                    respectsDueDate: input["respectsDueDate"] as! Bool,
                    observer: input["observer"] as! Bool
                )
            )
            if item["expected"] is NSNull || item["expected"] == nil {
                XCTAssertNil(reason, item["name"] as? String ?? "")
            } else {
                let expected = ContentToolHostLogic.ReadOnlyReason(rawValue: item["expected"] as! String)
                XCTAssertEqual(reason, expected, item["name"] as? String ?? "")
            }
        }
    }

    func testContractGatingMatchesFixture() throws {
        let root = try fixtureRoot()
        let contract = root["contract"] as! [String: Any]
        let supported = contract["supportedVersion"] as! Int
        for item in contract["cases"] as! [[String: Any]] {
            let value = item["contract"] as! Int
            let ok = item["supported"] as! Bool
            XCTAssertEqual(ContentToolHostLogic.contractSupported(value, supported: supported), ok)
        }
    }

    func testFenceMappingMatchesFixture() throws {
        let root = try fixtureRoot()
        let mapping = root["fenceMapping"] as! [String: Any]
        let instances = (mapping["instances"] as! [[String: Any]]).map {
            ToolInstance(id: $0["id"] as! String, toolId: $0["toolId"] as! String)
        }
        let map = ContentToolHostLogic.instanceMap(instances)
        for item in mapping["cases"] as! [[String: Any]] {
            let id = item["instanceId"] as! String
            let found = item["found"] as! Bool
            XCTAssertEqual(ContentToolHostLogic.resolveInstance(map, instanceId: id) != nil, found)
            XCTAssertEqual(ContentToolHostLogic.shouldMountFence(map[id]), found)
        }
    }

    func testRenderGateMatchesFixture() throws {
        let root = try fixtureRoot()
        let cases = (root["renderGate"] as! [String: Any])["cases"] as! [[String: Any]]
        for item in cases {
            let mode = ContentToolHostLogic.fenceRenderMode(
                mobileContentToolsEnabled: item["mobileContentToolsEnabled"] as! Bool,
                contentToolsEnabled: item["contentToolsEnabled"] as! Bool
            )
            let expected = ContentToolHostLogic.FenceRenderMode(rawValue: item["expected"] as! String)
            XCTAssertEqual(mode, expected, item["name"] as? String ?? "")
        }
    }

    func testHighlightAnnotateUsesMergePolicy() {
        XCTAssertEqual(
            ContentToolHostLogic.conflictPolicyForTool("highlight_annotate"),
            .merge
        )
        XCTAssertEqual(
            ContentToolHostLogic.conflictPolicyForTool("noop_probe"),
            .serverWins
        )
    }

    func testActionsAreNotQueuedOffline() {
        XCTAssertFalse(ContentToolHostLogic.canQueueActionOffline())
        XCTAssertTrue(ContentToolHostLogic.canQueueStateWriteOffline())
    }

    func testOutboxOrdersPerInstance() {
        let items: [(String, Int64)] = [("b", 2), ("a", 10), ("b", 1), ("a", 1)]
        let ordered = ContentToolHostLogic.orderOutboxByInstance(items)
        XCTAssertEqual(ordered.map(\.0), ["a", "a", "b", "b"])
        XCTAssertEqual(ordered.map(\.1), [1, 10, 1, 2])
    }

    func testUnsupportedPlaceholderForUnknownToolOrContract() {
        XCTAssertTrue(ContentToolHostLogic.shouldShowUnsupportedPlaceholder(toolId: "inline_questions", contract: 1))
        XCTAssertTrue(ContentToolHostLogic.shouldShowUnsupportedPlaceholder(toolId: "noop_probe", contract: 2))
        XCTAssertFalse(ContentToolHostLogic.shouldShowUnsupportedPlaceholder(toolId: "noop_probe", contract: 1))
    }

    func testShouldFetchRequiresFlagsAndContext() {
        XCTAssertTrue(
            ContentToolHostLogic.shouldFetchInstances(
                mobileContentToolsEnabled: true,
                contentToolsEnabled: true,
                courseCode: "CS101",
                itemId: "item-1"
            )
        )
        XCTAssertFalse(
            ContentToolHostLogic.shouldFetchInstances(
                mobileContentToolsEnabled: true,
                contentToolsEnabled: true,
                courseCode: "CS101",
                itemId: nil
            )
        )
    }

    func testWebActivityPathAnchorsInstance() {
        XCTAssertEqual(
            ContentToolHostLogic.webActivityPath(courseCode: "CS101", itemId: "item-1", instanceId: "abc"),
            "/courses/CS101/modules/items/item-1#lex-tool-abc"
        )
    }
}
