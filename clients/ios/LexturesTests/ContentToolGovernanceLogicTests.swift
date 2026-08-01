import XCTest
@testable import Lextures

final class ContentToolGovernanceLogicTests: XCTestCase {
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
            .appendingPathComponent("mobile/fixtures/content-tools/governance-logic.json")
        if FileManager.default.fileExists(atPath: direct.path) { return direct }
        let searchRoots = [
            URL(fileURLWithPath: FileManager.default.currentDirectoryPath),
            thisFile.deletingLastPathComponent(),
        ]
        for root in searchRoots {
            var dir = root
            for _ in 0 ..< 8 {
                for relative in [
                    "clients/mobile/fixtures/content-tools/governance-logic.json",
                    "mobile/fixtures/content-tools/governance-logic.json",
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

    private func stringArray(_ value: Any?) -> [String] {
        (value as? [String]) ?? []
    }

    private func mountInput(from raw: [String: Any]) -> ContentToolGovernanceLogic.MountInput {
        ContentToolGovernanceLogic.MountInput(
            toolId: raw["toolId"] as? String ?? "",
            capabilities: stringArray(raw["capabilities"]),
            sandboxMode: raw["sandboxMode"] as? String,
            tombstone: raw["tombstone"] as? Bool ?? false,
            breakerOpen: raw["breakerOpen"] as? Bool ?? false,
            deprecated: raw["deprecated"] as? Bool ?? false,
            killed: raw["killed"] as? Bool ?? false,
            allowedToolIds: stringArray(raw["allowedToolIds"]),
            deniedToolIds: stringArray(raw["deniedToolIds"]),
            deniedCapabilities: stringArray(raw["deniedCapabilities"]),
            policyFetched: raw["policyFetched"] as? Bool ?? false,
            policyAgeMs: (raw["policyAgeMs"] as? NSNumber)?.int64Value ?? 0,
            staleWindowMs: (raw["staleWindowMs"] as? NSNumber)?.int64Value
                ?? ContentToolGovernanceLogic.defaultStaleWindowMs,
            unknownGovernanceState: raw["unknownGovernanceState"] as? Bool ?? false,
            hasCachedPolicy: raw["hasCachedPolicy"] as? Bool ?? false
        )
    }

    func testMountDecisionMatchesFixture() throws {
        let root = try fixtureRoot()
        let cases = try objects(try object(root["mountDecision"])["cases"])
        for item in cases {
            let name = item["name"] as? String ?? "?"
            let input = mountInput(from: try object(item["input"]))
            let expected = try XCTUnwrap(item["expected"] as? String)
            let actual = ContentToolGovernanceLogic.mountDecision(input).rawValue
            XCTAssertEqual(actual, expected, name)
        }
    }

    func testConsentGatingMatchesFixture() throws {
        let root = try fixtureRoot()
        let cases = try objects(try object(root["consentGating"])["cases"])
        for item in cases {
            let name = item["name"] as? String ?? "?"
            let mode = item["disclosureMode"] as? String
            let decision = item["decision"] as? String
            let fetched = item["consentFetched"] as? Bool ?? false
            let allowed = ContentToolGovernanceLogic.aiActionsAllowed(
                disclosureMode: mode,
                decision: decision,
                consentFetched: fetched
            )
            let show = ContentToolGovernanceLogic.shouldShowAIDisclosure(
                disclosureMode: mode,
                decision: decision,
                consentFetched: fetched
            )
            XCTAssertEqual(allowed, item["aiAllowed"] as? Bool ?? false, "\(name) aiAllowed")
            XCTAssertEqual(show, item["showDisclosure"] as? Bool ?? false, "\(name) showDisclosure")
        }
    }

    func testKillDerivationMatchesFixture() throws {
        let root = try fixtureRoot()
        let cases = try objects(try object(root["killDerivation"])["cases"])
        for item in cases {
            let name = item["name"] as? String ?? "?"
            let actual = ContentToolGovernanceLogic.toolIsKilled(
                toolId: item["toolId"] as? String ?? "",
                capabilities: stringArray(item["capabilities"]),
                killedToolIds: stringArray(item["killedToolIds"]),
                killedCapabilities: stringArray(item["killedCapabilities"]),
                killAllAI: item["killAllAI"] as? Bool ?? false
            )
            XCTAssertEqual(actual, item["expected"] as? Bool ?? false, name)
        }
    }

    func testReportReachableInTwoTaps() throws {
        let root = try fixtureRoot()
        let reach = try object(root["reportReachability"])
        let maxTaps = reach["maxTaps"] as? Int ?? 2
        XCTAssertLessThanOrEqual(ContentToolGovernanceLogic.reportTapCount(), maxTaps)
        XCTAssertTrue(ContentToolGovernanceLogic.reportReachableInTwoTaps())
    }

    func testTelemetryPayloadShape() throws {
        let root = try fixtureRoot()
        let shape = try object(root["telemetryPayloadShape"])
        for example in try objects(shape["validExamples"]) {
            let attrs = example.reduce(into: [String: String]()) { acc, pair in
                if let s = pair.value as? String { acc[pair.key] = s }
            }
            XCTAssertTrue(ContentToolGovernanceLogic.telemetryAttributesAreContentFree(attrs))
        }
        for example in try objects(shape["invalidExamples"]) {
            let attrs = example.reduce(into: [String: String]()) { acc, pair in
                if let s = pair.value as? String { acc[pair.key] = s }
            }
            XCTAssertFalse(ContentToolGovernanceLogic.telemetryAttributesAreContentFree(attrs))
        }
    }

    func testFilterCrisisMatchesFixture() throws {
        let root = try fixtureRoot()
        let cases = try objects(try object(root["filterCrisis"])["cases"])
        for item in cases {
            let name = item["name"] as? String ?? "?"
            let outcome = ContentToolGovernanceLogic.filterCrisisOutcome(
                ContentToolGovernanceLogic.FilterCrisisInput(
                    errorCode: item["errorCode"] as? String,
                    crisis: item["crisis"] as? Bool ?? false
                )
            )
            XCTAssertEqual(outcome.kind.rawValue, item["expectedKind"] as? String, name)
            XCTAssertEqual(outcome.preserveDraft, item["preserveDraft"] as? Bool ?? true, name)
            XCTAssertEqual(outcome.retry, item["retry"] as? Bool ?? false, name)
        }
    }

    func testMessageKeysMatchFixture() throws {
        let root = try fixtureRoot()
        let keys = try object(root["messageKeys"])
        for (decision, key) in keys {
            guard let expected = key as? String,
                  let mount = ContentToolGovernanceLogic.MountDecision(rawValue: decision)
            else { continue }
            XCTAssertEqual(ContentToolGovernanceLogic.reasonMessageKey(mount), expected, decision)
        }
    }

    func testObservabilityDropsLearnerContent() {
        ContentToolsObservability.resetForTests()
        ContentToolsObservability.record(
            "bad_event",
            toolId: "ask_questions",
            attributes: ["prompt": "what is photosynthesis?"]
        )
        XCTAssertEqual(ContentToolsObservability.count(for: "bad_event"), 0)
        ContentToolsObservability.record(
            "tool_mount",
            toolId: "flashcards",
            attributes: ["outcome": "ok"]
        )
        XCTAssertEqual(ContentToolsObservability.count(for: "tool_mount"), 1)
    }
}
