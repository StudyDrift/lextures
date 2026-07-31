import XCTest
@testable import Lextures

final class ContentToolHostLogicTests: XCTestCase {
    private func fixtureRoot() throws -> [String: Any] {
        let url = fixtureURL()
        let data = try Data(contentsOf: url)
        return try XCTUnwrap(JSONSerialization.jsonObject(with: data) as? [String: Any])
    }

    private func fixtureURL() -> URL {
        let thisFile = URL(fileURLWithPath: #filePath)
        let direct = thisFile
            .deletingLastPathComponent() // LexturesTests
            .deletingLastPathComponent() // ios
            .deletingLastPathComponent() // clients
            .appendingPathComponent("mobile/fixtures/content-tools/host-logic.json")
        if FileManager.default.fileExists(atPath: direct.path) { return direct }
        let searchRoots = [
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

    private func object(_ value: Any?, file: StaticString = #filePath, line: UInt = #line) throws -> [String: Any] {
        try XCTUnwrap(value as? [String: Any], file: file, line: line)
    }

    private func objects(_ value: Any?, file: StaticString = #filePath, line: UInt = #line) throws -> [[String: Any]] {
        try XCTUnwrap(value as? [[String: Any]], file: file, line: line)
    }

    private func bool(_ value: Any?, file: StaticString = #filePath, line: UInt = #line) throws -> Bool {
        try XCTUnwrap(value as? Bool, file: file, line: line)
    }

    private func int(_ value: Any?, file: StaticString = #filePath, line: UInt = #line) throws -> Int {
        if let intVal = value as? Int { return intVal }
        if let number = value as? NSNumber { return number.intValue }
        return try XCTUnwrap(value as? Int, file: file, line: line)
    }

    private func string(_ value: Any?, file: StaticString = #filePath, line: UInt = #line) throws -> String {
        try XCTUnwrap(value as? String, file: file, line: line)
    }

    private func numberMap(_ raw: [String: Any]) -> [String: JSONValue] {
        raw.mapValues { value in
            if let number = value as? NSNumber {
                return .number(number.doubleValue)
            }
            return .null
        }
    }

    func testClampDebounceMatchesFixture() throws {
        let root = try fixtureRoot()
        let debounce = try object(root["debounce"])
        let cases = try objects(debounce["cases"])
        for item in cases {
            let expected = try int(item["expected"])
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
        let cases = try objects(try object(root["conflictPolicy"])["cases"])
        for item in cases {
            let policy = ContentToolHostLogic.ConflictPolicy.from(item["policy"] as? String)
            let client = numberMap(try object(item["client"]))
            let server = numberMap(try object(item["server"]))
            let expected = numberMap(try object(item["expected"]))
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
        let cases = try objects(root["readOnlyPrecedence"])
        for item in cases {
            let input = try object(item["input"])
            let reason = ContentToolHostLogic.readOnlyReason(
                ContentToolHostLogic.ReadOnlyInput(
                    tombstone: try bool(input["tombstone"]),
                    breakerOpen: try bool(input["breakerOpen"]),
                    status: try string(input["status"]),
                    pastDue: try bool(input["pastDue"]),
                    respectsDueDate: try bool(input["respectsDueDate"]),
                    observer: try bool(input["observer"])
                )
            )
            if item["expected"] is NSNull || item["expected"] == nil {
                XCTAssertNil(reason, item["name"] as? String ?? "")
            } else {
                let expected = ContentToolHostLogic.ReadOnlyReason(rawValue: try string(item["expected"]))
                XCTAssertEqual(reason, expected, item["name"] as? String ?? "")
            }
        }
    }

    func testContractGatingMatchesFixture() throws {
        let root = try fixtureRoot()
        let contract = try object(root["contract"])
        let supported = try int(contract["supportedVersion"])
        for item in try objects(contract["cases"]) {
            let value = try int(item["contract"])
            let ok = try bool(item["supported"])
            XCTAssertEqual(ContentToolHostLogic.contractSupported(value, supported: supported), ok)
        }
    }

    func testFenceMappingMatchesFixture() throws {
        let root = try fixtureRoot()
        let mapping = try object(root["fenceMapping"])
        let instances = try objects(mapping["instances"]).map {
            ToolInstance(id: try string($0["id"]), toolId: try string($0["toolId"]))
        }
        let map = ContentToolHostLogic.instanceMap(instances)
        for item in try objects(mapping["cases"]) {
            let id = try string(item["instanceId"])
            let found = try bool(item["found"])
            XCTAssertEqual(ContentToolHostLogic.resolveInstance(map, instanceId: id) != nil, found)
            XCTAssertEqual(ContentToolHostLogic.shouldMountFence(map[id]), found)
        }
    }

    func testRenderGateMatchesFixture() throws {
        let root = try fixtureRoot()
        let cases = try objects(try object(root["renderGate"])["cases"])
        for item in cases {
            let mode = ContentToolHostLogic.fenceRenderMode(
                mobileContentToolsEnabled: try bool(item["mobileContentToolsEnabled"]),
                contentToolsEnabled: try bool(item["contentToolsEnabled"])
            )
            let expected = ContentToolHostLogic.FenceRenderMode(rawValue: try string(item["expected"]))
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
