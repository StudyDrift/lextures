import XCTest
@testable import Lextures

final class ContentToolSandboxLogicTests: XCTestCase {
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
            .appendingPathComponent("mobile/fixtures/content-tools/bridge/messages.json")
        if FileManager.default.fileExists(atPath: direct.path) { return direct }
        let searchRoots = [
            URL(fileURLWithPath: FileManager.default.currentDirectoryPath),
            thisFile.deletingLastPathComponent(),
        ]
        for root in searchRoots {
            var dir = root
            for _ in 0 ..< 8 {
                for relative in [
                    "clients/mobile/fixtures/content-tools/bridge/messages.json",
                    "mobile/fixtures/content-tools/bridge/messages.json",
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

    private func double(_ value: Any?, file: StaticString = #filePath, line: UInt = #line) throws -> Double {
        if let d = value as? Double { return d }
        if let number = value as? NSNumber { return number.doubleValue }
        return try XCTUnwrap(value as? Double, file: file, line: line)
    }

    private func string(_ value: Any?, file: StaticString = #filePath, line: UInt = #line) throws -> String {
        try XCTUnwrap(value as? String, file: file, line: line)
    }

    func testConstantsMatchFixture() throws {
        let c = try object(try fixtureRoot()["constants"])
        XCTAssertEqual(try int(c["bridgeVersion"]), ContentToolSandboxLogic.bridgeVersion)
        XCTAssertEqual(try int(c["maxMessageBytes"]), ContentToolSandboxLogic.bridgeMaxMessageBytes)
        XCTAssertEqual(try int(c["maxMessagesPerSec"]), ContentToolSandboxLogic.bridgeMaxMessagesPerSec)
        XCTAssertEqual(try int(c["readyTimeoutMs"]), ContentToolSandboxLogic.readyTimeoutMs)
        XCTAssertEqual(try double(c["minHeight"]), ContentToolSandboxLogic.minHeightPt)
        XCTAssertEqual(try double(c["maxHeight"]), ContentToolSandboxLogic.maxHeightPt)
        XCTAssertEqual(try int(c["maxLiveWebViews"]), ContentToolSandboxLogic.maxLiveWebViews)
    }

    func testValidationMatchesFixture() throws {
        let cases = try objects(try object(try fixtureRoot()["validation"])["cases"])
        for item in cases {
            let name = try string(item["name"])
            let direction = try string(item["direction"])
            let accept = try bool(item["accept"])
            let msg = item["msg"]
            let actual: Bool
            switch direction {
            case "fromTool": actual = ContentToolSandboxLogic.isBridgeFromTool(msg)
            case "toTool": actual = ContentToolSandboxLogic.isBridgeToTool(msg)
            default: return XCTFail("bad direction")
            }
            XCTAssertEqual(actual, accept, name)
        }
    }

    func testRateLimitMatchesFixture() throws {
        let rate = try object(try fixtureRoot()["rateLimit"])
        let max = try int(rate["maxPerSec"])
        for item in try objects(rate["cases"]) {
            let limiter = ContentToolSandboxLogic.BridgeRateLimiter(maxPerSec: max)
            let stamps = try XCTUnwrap(item["timestampsMs"] as? [Any]).map { try int($0) }
            let expected = try XCTUnwrap(item["expectedAllow"] as? [Any]).map { try bool($0) }
            let actual = stamps.map { limiter.allow(nowMs: Int64($0)) }
            XCTAssertEqual(actual, expected, try string(item["name"]))
        }
    }

    func testSizeGuardRejectsOversize() throws {
        for item in try objects(try object(try fixtureRoot()["sizeGuard"])["cases"]) {
            let approx = try int(item["approxBytes"])
            let reject = try bool(item["reject"])
            let payload = String(repeating: "x", count: approx)
            let raw = "{\"t\":\"announce\",\"v\":1,\"message\":\"\(payload)\"}"
            let limiter = ContentToolSandboxLogic.BridgeRateLimiter(maxPerSec: 1000)
            let reason = ContentToolSandboxLogic.rejectIngress(rawJSON: raw, limiter: limiter, nowMs: 1)
            if reject {
                XCTAssertEqual(reason, .oversized, try string(item["name"]))
            } else {
                XCTAssertNil(reason, try string(item["name"]))
            }
        }
    }

    func testHeightClampMatchesFixture() throws {
        for item in try objects(try object(try fixtureRoot()["heightClamp"])["cases"]) {
            XCTAssertEqual(
                ContentToolSandboxLogic.clampHeight(try double(item["input"])),
                try double(item["expected"])
            )
        }
    }

    func testOpaqueParticipantIdMatchesFixture() throws {
        for item in try objects(try object(try fixtureRoot()["opaqueParticipantId"])["cases"]) {
            let hint: String?
            if item["enrollmentHint"] is NSNull || item["enrollmentHint"] == nil {
                hint = nil
            } else {
                hint = try string(item["enrollmentHint"])
            }
            XCTAssertEqual(
                ContentToolSandboxLogic.opaqueParticipantId(try string(item["instanceId"]), enrollmentHint: hint),
                try string(item["expected"])
            )
        }
    }

    func testResolutionMatchesFixture() throws {
        for item in try objects(try object(try fixtureRoot()["resolution"])["cases"]) {
            let registeredList = try XCTUnwrap(item["registered"] as? [Any]).map { try string($0) }
            let sandboxMode: String?
            if item["sandboxMode"] is NSNull || item["sandboxMode"] == nil {
                sandboxMode = nil
            } else {
                sandboxMode = try string(item["sandboxMode"])
            }
            let path = ContentToolSandboxLogic.resolveRenderPath(
                toolId: try string(item["toolId"]),
                contract: try int(item["contract"]),
                sandboxMode: sandboxMode,
                sandboxEnabled: try bool(item["sandboxEnabled"]),
                registered: Set(registeredList),
                tombstone: try bool(item["tombstone"]),
                breakerOpen: try bool(item["breakerOpen"]),
                deprecated: try bool(item["deprecated"]),
                killed: try bool(item["killed"])
            )
            XCTAssertEqual(path.rawValue, try string(item["expected"]), try string(item["name"]))
        }
    }

    func testContractRangeMatchesFixture() throws {
        let root = try object(try fixtureRoot()["contractRange"])
        for item in try objects(root["cases"]) {
            XCTAssertEqual(
                ContentToolSandboxLogic.contractInSupportedRange(try string(item["contract"])),
                try bool(item["ok"])
            )
        }
    }

    func testPoolEvictsBeyondMax() {
        XCTAssertFalse(ContentToolSandboxLogic.poolShouldEvict(aliveCount: 3, maxAlive: 3))
        XCTAssertTrue(ContentToolSandboxLogic.poolShouldEvict(aliveCount: 4, maxAlive: 3))
    }
}
