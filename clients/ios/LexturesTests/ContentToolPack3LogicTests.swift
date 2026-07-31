import XCTest
@testable import Lextures

final class ContentToolPack3LogicTests: XCTestCase {
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
            .appendingPathComponent("mobile/fixtures/content-tools/pack3-logic.json")
        if FileManager.default.fileExists(atPath: direct.path) { return direct }
        var dir = URL(fileURLWithPath: FileManager.default.currentDirectoryPath)
        for _ in 0 ..< 8 {
            let candidate = dir.appendingPathComponent("clients/mobile/fixtures/content-tools/pack3-logic.json")
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

    private func asDouble(_ value: Any?, file: StaticString = #filePath, line: UInt = #line) throws -> Double {
        if let d = value as? Double { return d }
        if let i = value as? Int { return Double(i) }
        if let n = value as? NSNumber { return n.doubleValue }
        return try XCTUnwrap(value as? Double, file: file, line: line)
    }

    private func asInt(_ value: Any?, file: StaticString = #filePath, line: UInt = #line) throws -> Int {
        if let i = value as? Int { return i }
        if let n = value as? NSNumber { return n.intValue }
        return try XCTUnwrap(value as? Int, file: file, line: line)
    }

    private func stringArray(_ value: Any?, file: StaticString = #filePath, line: UInt = #line) throws -> [String] {
        try XCTUnwrap(value as? [String], file: file, line: line)
    }

    private func jsonValue(_ any: Any?) -> JSONValue? {
        guard let any else { return nil }
        if any is NSNull { return nil }
        if let stringValue = any as? String { return .string(stringValue) }
        if let boolValue = any as? Bool { return .bool(boolValue) }
        if let numberValue = any as? NSNumber {
            if CFGetTypeID(numberValue) == CFBooleanGetTypeID() {
                return .bool(numberValue.boolValue)
            }
            return .number(numberValue.doubleValue)
        }
        if let arr = any as? [Any] {
            return .array(arr.map { jsonValue($0) ?? .null })
        }
        if let dict = any as? [String: Any] {
            return .object(dict.mapValues { jsonValue($0) ?? .null })
        }
        return nil
    }

    private func categorizePlacement(_ raw: [String: Any]) -> [String: String?] {
        var out: [String: String?] = [:]
        for (key, value) in raw {
            if value is NSNull {
                out[key] = nil as String?
            } else if let stringValue = value as? String {
                out[key] = stringValue
            }
        }
        return out
    }

    private func optionalString(_ value: Any?) -> String? {
        if value == nil || value is NSNull { return nil }
        return value as? String
    }

    private func optionalInt(_ value: Any?) -> Int? {
        if value == nil || value is NSNull { return nil }
        return (value as? NSNumber)?.intValue ?? (value as? Int)
    }

    private func placementHit(_ raw: [String: Any]) throws -> ContentToolPack3Logic.PlacementHit {
        let type = try XCTUnwrap(raw["type"] as? String)
        switch type {
        case "item":
            return .item(try XCTUnwrap(raw["id"] as? String))
        case "bucket":
            return .bucket(try XCTUnwrap(raw["id"] as? String))
        case "tray":
            return .tray
        case "position":
            return .position(try asInt(raw["index"]))
        default:
            XCTFail("unknown hit type \(type)")
            return .tray
        }
    }

    private func regionShape(_ raw: [String: Any]) throws -> ContentToolPack3Logic.RegionShape {
        let kind = try XCTUnwrap(raw["kind"] as? String)
        switch kind {
        case "rect":
            return .rect(
                x: try asDouble(raw["x"]),
                y: try asDouble(raw["y"]),
                w: try asDouble(raw["w"]),
                h: try asDouble(raw["h"])
            )
        case "circle":
            return .circle(
                cx: try asDouble(raw["cx"]),
                cy: try asDouble(raw["cy"]),
                r: try asDouble(raw["r"])
            )
        default:
            XCTFail("unknown shape kind \(kind)")
            return .rect(x: 0, y: 0, w: 0, h: 0)
        }
    }

    private func assertPlacementMapsEqual(
        _ actual: [String: String?],
        _ expected: [String: String?],
        _ message: String,
        file: StaticString = #filePath,
        line: UInt = #line
    ) {
        XCTAssertEqual(Set(actual.keys), Set(expected.keys), message, file: file, line: line)
        for (key, exp) in expected {
            XCTAssertTrue(actual.keys.contains(key), "\(message) missing \(key)", file: file, line: line)
            let act: String? = actual[key]!
            XCTAssertEqual(act, exp, "\(message) key=\(key)", file: file, line: line)
        }
    }

    func testAllowlistMatchesFixture() throws {
        let root = try object(try fixtureRoot()["allowlist"])
        let pack3ToolIds = Set(try stringArray(root["pack3ToolIds"]))
        XCTAssertEqual(ContentToolPack3Logic.pack3ToolIds, pack3ToolIds)
        for item in try objects(root["cases"]) {
            let allowlist = Set(try stringArray(item["allowlist"]))
            let toolId = try XCTUnwrap(item["toolId"] as? String)
            let expected = try XCTUnwrap(item["expected"] as? Bool)
            XCTAssertEqual(
                ContentToolPack3Logic.isClientAllowlisted(toolId, allowlist: allowlist),
                expected,
                item["name"] as? String ?? toolId
            )
        }
    }

    func testAllowlistedToolIdsEqualsPack3ToolIds() {
        XCTAssertEqual(ContentToolPack3Logic.allowlistedToolIds(), ContentToolPack3Logic.pack3ToolIds)
    }

    func testConflictPolicyMatchesFixture() throws {
        for item in try objects(try object(try fixtureRoot()["conflictPolicy"])["cases"]) {
            let toolId = try XCTUnwrap(item["toolId"] as? String)
            let expected = try XCTUnwrap(item["expected"] as? String)
            XCTAssertEqual(ContentToolPack3Logic.conflictPolicy(for: toolId).rawValue, expected, toolId)
            XCTAssertEqual(ContentToolHostLogic.conflictPolicyForTool(toolId).rawValue, expected, toolId)
        }
    }

    func testOfflineQueueNeverQueues() throws {
        for item in try objects(try object(try fixtureRoot()["offlineQueue"])["cases"]) {
            let toolId = try XCTUnwrap(item["toolId"] as? String)
            let action = try XCTUnwrap(item["action"] as? String)
            XCTAssertEqual(
                ContentToolPack3Logic.canQueueActionOffline(toolId: toolId, action: action),
                try XCTUnwrap(item["expected"] as? Bool),
                "\(toolId)/\(action)"
            )
        }
    }

    func testAttemptsMatchesFixture() throws {
        let root = try object(try fixtureRoot()["attempts"])
        for item in try objects(root["cases"]) {
            let rawAny = item["raw"]
            let parsed = ContentToolPack3Logic.parseAttemptsConfig(jsonValue(rawAny))
            let expectedMax = optionalInt(item["expectedMax"])
            XCTAssertEqual(parsed, expectedMax, item["name"] as? String ?? "")
        }
        for item in try objects(root["canCheck"]) {
            let maxAttempts = optionalInt(item["maxAttempts"])
            XCTAssertEqual(
                ContentToolPack3Logic.canCheck(
                    attemptsUsed: try asInt(item["attemptsUsed"]),
                    maxAttempts: maxAttempts,
                    readOnly: try XCTUnwrap(item["readOnly"] as? Bool)
                ),
                try XCTUnwrap(item["expected"] as? Bool),
                item["name"] as? String ?? ""
            )
        }
    }

    func testSortReorderMatchesFixture() throws {
        for item in try objects(try object(try fixtureRoot()["sortReorder"])["cases"]) {
            let result = ContentToolPack3Logic.moveInOrder(
                order: try stringArray(item["order"]),
                itemId: try XCTUnwrap(item["itemId"] as? String),
                direction: try asInt(item["direction"]),
                lockedItemIds: try stringArray(item["locked"])
            )
            XCTAssertEqual(result, try stringArray(item["expected"]), item["name"] as? String ?? "")
        }
    }

    func testSortTapAssignCategorizeMatchesFixture() throws {
        for item in try objects(try object(try fixtureRoot()["sortTapAssign"])["categorize"]) {
            let placementMap = categorizePlacement(try object(item["placement"]))
            let state = ContentToolPack3Logic.EngineState(
                grabbedId: optionalString(item["grabbedId"]),
                target: nil,
                placement: .categorize(placementMap)
            )
            let next = ContentToolPack3Logic.tapItemOrTarget(
                state,
                mode: .categorize,
                lockedItemIds: [],
                hit: try placementHit(try object(item["hit"]))
            )
            XCTAssertEqual(next.grabbedId, optionalString(item["expectedGrabbed"]), item["name"] as? String ?? "")
            let expected = categorizePlacement(try object(item["expectedPlacement"]))
            let actual = try XCTUnwrap(next.placement.asCategorize)
            assertPlacementMapsEqual(actual, expected, item["name"] as? String ?? "")
        }
    }

    func testSortTapAssignOrderMatchesFixture() throws {
        for item in try objects(try object(try fixtureRoot()["sortTapAssign"])["order"]) {
            let state = ContentToolPack3Logic.EngineState(
                grabbedId: optionalString(item["grabbedId"]),
                target: nil,
                placement: .order(try stringArray(item["placement"]))
            )
            let next = ContentToolPack3Logic.tapItemOrTarget(
                state,
                mode: .order,
                lockedItemIds: [],
                hit: try placementHit(try object(item["hit"]))
            )
            XCTAssertEqual(next.grabbedId, optionalString(item["expectedGrabbed"]), item["name"] as? String ?? "")
            XCTAssertEqual(
                try XCTUnwrap(next.placement.asOrder),
                try stringArray(item["expectedPlacement"]),
                item["name"] as? String ?? ""
            )
        }
    }

    func testDragInterruptMatchesFixture() throws {
        for item in try objects(try object(try fixtureRoot()["dragInterrupt"])["cases"]) {
            let settledPlacement = categorizePlacement(try object(item["settledPlacement"]))
            let settled = ContentToolPack3Logic.EngineState(
                grabbedId: optionalString(item["inFlightGrabbed"]),
                target: .tray,
                placement: .categorize(settledPlacement)
            )
            let restored = ContentToolPack3Logic.restoreAfterDragInterrupt(settled: settled)
            XCTAssertEqual(restored.grabbedId, optionalString(item["expectedGrabbed"]), item["name"] as? String ?? "")
            let expected = categorizePlacement(try object(item["expectedPlacement"]))
            assertPlacementMapsEqual(try XCTUnwrap(restored.placement.asCategorize), expected, item["name"] as? String ?? "")
        }
    }

    func testAllPlacedMatchesFixture() throws {
        for item in try objects(try object(try fixtureRoot()["allPlaced"])["cases"]) {
            let modeRaw = try XCTUnwrap(item["mode"] as? String)
            let mode = ContentToolPack3Logic.PlacementMode(rawValue: modeRaw)!
            let itemIds = try stringArray(item["itemIds"])
            let placement: ContentToolPack3Logic.Placement
            switch mode {
            case .categorize:
                placement = .categorize(categorizePlacement(try object(item["placement"])))
            case .order:
                placement = .order(try stringArray(item["placement"]))
            }
            XCTAssertEqual(
                ContentToolPack3Logic.allPlaced(mode: mode, itemIds: itemIds, placement: placement),
                try XCTUnwrap(item["expected"] as? Bool),
                item["name"] as? String ?? ""
            )
        }
    }

    func testAnchorsBuildResolveSegmentMatchFixture() throws {
        let root = try object(try fixtureRoot()["anchors"])
        XCTAssertEqual(ContentToolPack3Logic.contextLen, try asInt(root["contextLen"]))

        for item in try objects(root["build"]) {
            let built = try XCTUnwrap(
                ContentToolPack3Logic.buildQuoteAnchor(
                    passage: try XCTUnwrap(item["passage"] as? String),
                    start: try asInt(item["start"]),
                    end: try asInt(item["end"])
                ),
                item["name"] as? String ?? ""
            )
            XCTAssertEqual(built.quote, try XCTUnwrap(item["expectedQuote"] as? String), item["name"] as? String ?? "")
            XCTAssertEqual(built.anchor.prefix, try XCTUnwrap(item["expectedPrefix"] as? String))
            XCTAssertEqual(built.anchor.suffix, try XCTUnwrap(item["expectedSuffix"] as? String))
            XCTAssertEqual(built.anchor.approxOffset, try asInt(item["expectedOffset"]))
        }

        for item in try objects(root["resolve"]) {
            let anchorObj = try object(item["anchor"])
            let anchor = ContentToolPack3Logic.QuoteAnchor(
                prefix: try XCTUnwrap(anchorObj["prefix"] as? String),
                suffix: try XCTUnwrap(anchorObj["suffix"] as? String),
                approxOffset: try asInt(anchorObj["approxOffset"])
            )
            let resolved = ContentToolPack3Logic.resolveQuoteAnchor(
                passage: try XCTUnwrap(item["passage"] as? String),
                quote: try XCTUnwrap(item["quote"] as? String),
                anchor: anchor
            )
            let expectedStart = optionalInt(item["expectedStart"])
            let expectedEnd = optionalInt(item["expectedEnd"])
            if expectedStart == nil && expectedEnd == nil {
                XCTAssertNil(resolved, item["name"] as? String ?? "")
            } else {
                let range = try XCTUnwrap(resolved, item["name"] as? String ?? "")
                XCTAssertEqual(range.start, expectedStart)
                XCTAssertEqual(range.end, expectedEnd)
            }
        }

        for item in try objects(root["segment"]) {
            let units = ContentToolPack3Logic.segmentPassage(
                try XCTUnwrap(item["passage"] as? String),
                granularity: try XCTUnwrap(item["granularity"] as? String)
            )
            let expected = try objects(item["expected"])
            XCTAssertEqual(units.count, expected.count, item["name"] as? String ?? "")
            for (unit, exp) in zip(units, expected) {
                XCTAssertEqual(unit.index, try asInt(exp["index"]))
                XCTAssertEqual(unit.text, try XCTUnwrap(exp["text"] as? String))
                XCTAssertEqual(unit.start, try asInt(exp["start"]))
                XCTAssertEqual(unit.end, try asInt(exp["end"]))
            }
        }
    }

    func testGeometryMatchesFixture() throws {
        let root = try object(try fixtureRoot()["geometry"])

        for item in try objects(root["clamp01"]) {
            XCTAssertEqual(
                ContentToolPack3Logic.clamp01(try asDouble(item["input"])),
                try asDouble(item["expected"]),
                accuracy: 1e-9
            )
        }

        for item in try objects(root["pointInShape"]) {
            let shape = try regionShape(try object(item["shape"]))
            XCTAssertEqual(
                ContentToolPack3Logic.pointInShape(
                    x: try asDouble(item["x"]),
                    y: try asDouble(item["y"]),
                    shape: shape
                ),
                try XCTUnwrap(item["expected"] as? Bool),
                item["name"] as? String ?? ""
            )
        }

        for item in try objects(root["hitTest"]) {
            let regions: [ContentToolPack3Logic.DiagramRegion] = try objects(item["regions"]).map { region in
                ContentToolPack3Logic.DiagramRegion(
                    id: try XCTUnwrap(region["id"] as? String),
                    label: try XCTUnwrap(region["label"] as? String),
                    description: try XCTUnwrap(region["description"] as? String),
                    shape: try regionShape(try object(region["shape"]))
                )
            }
            let hit = ContentToolPack3Logic.hitTestRegions(
                regions: regions,
                x: try asDouble(item["x"]),
                y: try asDouble(item["y"])
            )
            XCTAssertEqual(hit?.id, try XCTUnwrap(item["expectedId"] as? String), item["name"] as? String ?? "")
        }

        for item in try objects(root["hitTargetExpansion"]) {
            let shape = try regionShape(try object(item["shape"]))
            let original = shape
            let point = try XCTUnwrap(item["pointInsideExpanded"] as? [Any])
            let contains = ContentToolPack3Logic.pointInExpandedHitTarget(
                x: try asDouble(point[0]),
                y: try asDouble(point[1]),
                shape: shape,
                imageDisplayWidthPt: try asDouble(item["imageDisplayWidthPt"]),
                imageDisplayHeightPt: try asDouble(item["imageDisplayHeightPt"]),
                minTargetPt: try asDouble(item["minTargetPt"])
            )
            XCTAssertEqual(contains, try XCTUnwrap(item["expectedContains"] as? Bool), item["name"] as? String ?? "")
            if try XCTUnwrap(item["storedShapeUnchanged"] as? Bool) {
                XCTAssertEqual(shape, original, item["name"] as? String ?? "")
            }
        }

        for item in try objects(root["pointerToNormalized"]) {
            let point = try XCTUnwrap(
                ContentToolPack3Logic.pointerToNormalized(
                    clientX: try asDouble(item["clientX"]),
                    clientY: try asDouble(item["clientY"]),
                    viewWidth: try asDouble(item["viewWidth"]),
                    viewHeight: try asDouble(item["viewHeight"]),
                    naturalWidth: try asDouble(item["naturalWidth"]),
                    naturalHeight: try asDouble(item["naturalHeight"]),
                    zoom: try asDouble(item["zoom"]),
                    panX: try asDouble(item["panX"]),
                    panY: try asDouble(item["panY"])
                ),
                item["name"] as? String ?? ""
            )
            XCTAssertEqual(point.0, try asDouble(item["expectedX"]), accuracy: 1e-9, item["name"] as? String ?? "")
            XCTAssertEqual(point.1, try asDouble(item["expectedY"]), accuracy: 1e-9, item["name"] as? String ?? "")
        }
    }

    func testCheckResultMatchesFixture() throws {
        for item in try objects(try object(try fixtureRoot()["checkResult"])["cases"]) {
            let parsed = ContentToolPack3Logic.parseCheckResult(jsonValue(item["result"]))
            let expectedError = optionalString(item["expectedError"])
            XCTAssertEqual(parsed.error?.rawValue, expectedError, item["name"] as? String ?? "")
            let expectedScore = item["expectedScore"]
            if expectedScore == nil || expectedScore is NSNull {
                XCTAssertNil(parsed.scorePct, item["name"] as? String ?? "")
            } else {
                XCTAssertEqual(try XCTUnwrap(parsed.scorePct), try asDouble(expectedScore), accuracy: 1e-9)
            }
            XCTAssertEqual(parsed.attemptsRemaining, optionalInt(item["expectedAttemptsRemaining"]))
        }
    }

    func testRegisteredNativeIncludesPack3() {
        let ids = ContentToolHostLogic.registeredNativeToolIds()
        XCTAssertTrue(ids.contains("noop_probe"))
        for toolId in ContentToolPack3Logic.pack3ToolIds {
            XCTAssertTrue(ids.contains(toolId), toolId)
            XCTAssertFalse(ContentToolHostLogic.shouldShowUnsupportedPlaceholder(toolId: toolId, contract: 1))
        }
        XCTAssertTrue(ToolRendererRegistry.registeredIds().isSuperset(of: ContentToolPack3Logic.pack3ToolIds))
    }
}
