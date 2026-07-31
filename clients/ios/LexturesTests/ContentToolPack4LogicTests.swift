// swiftlint:disable identifier_name type_body_length file_length
import XCTest
@testable import Lextures

final class ContentToolPack4LogicTests: XCTestCase {
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
            .appendingPathComponent("mobile/fixtures/content-tools/pack4-logic.json")
        if FileManager.default.fileExists(atPath: direct.path) { return direct }
        var dir = URL(fileURLWithPath: FileManager.default.currentDirectoryPath)
        for _ in 0 ..< 8 {
            let candidate = dir.appendingPathComponent("clients/mobile/fixtures/content-tools/pack4-logic.json")
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

    private func optionalString(_ value: Any?) -> String? {
        if value == nil || value is NSNull { return nil }
        return value as? String
    }

    private func optionalInt64(_ value: Any?) -> Int64? {
        if value == nil || value is NSNull { return nil }
        if let i = value as? Int { return Int64(i) }
        if let n = value as? NSNumber { return n.int64Value }
        return nil
    }

    private func optionalDouble(_ value: Any?) -> Double? {
        if value == nil || value is NSNull { return nil }
        return (value as? NSNumber)?.doubleValue ?? (value as? Double)
    }

    private func jsonValue(_ any: Any?) -> JSONValue? {
        guard let any else { return nil }
        if any is NSNull { return .null }
        if let numberValue = any as? NSNumber {
            if CFGetTypeID(numberValue) == CFBooleanGetTypeID() {
                return .bool(numberValue.boolValue)
            }
            return .number(numberValue.doubleValue)
        }
        if let stringValue = any as? String { return .string(stringValue) }
        if let boolValue = any as? Bool { return .bool(boolValue) }
        if let arr = any as? [Any] {
            return .array(arr.map { jsonValue($0) ?? .null })
        }
        if let dict = any as? [String: Any] {
            return .object(dict.mapValues { jsonValue($0) ?? .null })
        }
        return nil
    }

    private func checkpoints(from root: [String: Any]) throws -> [ContentToolPack4Logic.Checkpoint] {
        try objects(root["checkpoints"]).map { item in
            ContentToolPack4Logic.Checkpoint(
                id: try XCTUnwrap(item["id"] as? String),
                atSec: try asDouble(item["atSec"]),
                required: try XCTUnwrap(item["required"] as? Bool),
                attempts: try asInt(item["attempts"])
            )
        }
    }

    private func answers(from raw: [String: Any]) -> [String: ContentToolPack4Logic.CheckpointAnswer] {
        var out: [String: ContentToolPack4Logic.CheckpointAnswer] = [:]
        for (id, value) in raw {
            guard let obj = value as? [String: Any] else { continue }
            let done = obj["done"] as? Bool ?? false
            let attempts = obj["attempts"] as? [[String: Any]] ?? []
            let lastCorrect = (attempts.last?["correct"] as? Bool) ?? false
            out[id] = ContentToolPack4Logic.CheckpointAnswer(
                done: done,
                attemptCount: attempts.count,
                lastCorrect: lastCorrect
            )
        }
        return out
    }

    func testAllowlistMatchesFixture() throws {
        let root = try object(try fixtureRoot()["allowlist"])
        let pack4ToolIds = Set(try stringArray(root["pack4ToolIds"]))
        XCTAssertEqual(ContentToolPack4Logic.pack4ToolIds, pack4ToolIds)
        let sandboxIds = Set(try stringArray(root["sandboxToolIds"]))
        XCTAssertEqual(ContentToolPack4Logic.sandboxToolIds, sandboxIds)
        for item in try objects(root["cases"]) {
            let allowlist = Set(try stringArray(item["allowlist"]))
            let toolId = try XCTUnwrap(item["toolId"] as? String)
            let expected = try XCTUnwrap(item["expected"] as? Bool)
            XCTAssertEqual(
                ContentToolPack4Logic.isClientAllowlisted(toolId, allowlist: allowlist),
                expected,
                item["name"] as? String ?? toolId
            )
        }
    }

    func testAllowlistedToolIdsEqualsPack4ToolIds() {
        XCTAssertEqual(ContentToolPack4Logic.allowlistedToolIds(), ContentToolPack4Logic.pack4ToolIds)
        XCTAssertFalse(ContentToolPack4Logic.allowlistedToolIds().contains("code_sandbox"))
    }

    func testConflictPolicyMatchesFixture() throws {
        for item in try objects(try object(try fixtureRoot()["conflictPolicy"])["cases"]) {
            let toolId = try XCTUnwrap(item["toolId"] as? String)
            let expected = try XCTUnwrap(item["expected"] as? String)
            XCTAssertEqual(ContentToolPack4Logic.conflictPolicy(for: toolId).rawValue, expected, toolId)
            if ContentToolPack4Logic.pack4ToolIds.contains(toolId) {
                XCTAssertEqual(ContentToolHostLogic.conflictPolicyForTool(toolId).rawValue, expected, toolId)
            } else {
                // code_sandbox uses manifest default (server_wins)
                XCTAssertEqual(ContentToolHostLogic.conflictPolicyForTool(toolId).rawValue, expected, toolId)
            }
        }
    }

    func testOfflineQueueNeverQueues() throws {
        for item in try objects(try object(try fixtureRoot()["offlineQueue"])["cases"]) {
            let toolId = try XCTUnwrap(item["toolId"] as? String)
            let action = try XCTUnwrap(item["action"] as? String)
            XCTAssertEqual(
                ContentToolPack4Logic.canQueueActionOffline(toolId: toolId, action: action),
                try XCTUnwrap(item["expected"] as? Bool),
                "\(toolId)/\(action)"
            )
        }
    }

    func testCheckpointEngineMatchesFixture() throws {
        let root = try object(try fixtureRoot()["checkpointEngine"])
        XCTAssertEqual(ContentToolPack4Logic.checkpointToleranceSec, try asDouble(root["toleranceSec"]))
        let cps = try checkpoints(from: root)

        for item in try objects(root["findDue"]) {
            let ans = answers(from: try object(item["answers"]))
            let prompted = Set(try stringArray(item["prompted"]))
            let due = ContentToolPack4Logic.findDueCheckpoint(
                checkpoints: cps,
                answers: ans,
                currentTime: try asDouble(item["currentTime"]),
                alreadyPromptedIds: prompted
            )
            XCTAssertEqual(due?.id, optionalString(item["expectedId"]), item["name"] as? String ?? "")
        }

        for item in try objects(root["clampSeek"]) {
            let ans = answers(from: try object(item["answers"]))
            let result = ContentToolPack4Logic.clampSeekTime(
                preventSkip: try XCTUnwrap(item["preventSkip"] as? Bool),
                checkpoints: cps,
                answers: ans,
                targetSec: try asDouble(item["targetSec"])
            )
            XCTAssertEqual(result.time, try asDouble(item["expectedTime"]), accuracy: 1e-9, item["name"] as? String ?? "")
            XCTAssertEqual(result.clamped, try XCTUnwrap(item["expectedClamped"] as? Bool), item["name"] as? String ?? "")
        }

        for item in try objects(root["isDone"]) {
            let ans = answers(from: try object(item["answers"]))
            let id = try XCTUnwrap(item["checkpointId"] as? String)
            let cp = try XCTUnwrap(cps.first { $0.id == id })
            XCTAssertEqual(
                ContentToolPack4Logic.isCheckpointDone(answers: ans, checkpoint: cp),
                try XCTUnwrap(item["expected"] as? Bool),
                item["name"] as? String ?? ""
            )
        }

        for item in try objects(root["mergeSegments"]) {
            let existingRaw = try XCTUnwrap(item["existing"] as? [[Any]])
            let existingSegs: [[Double]] = try existingRaw.map { row in
                [try asDouble(row[0]), try asDouble(row[1])]
            }
            let merged = ContentToolPack4Logic.mergeLocalSegments(
                existing: existingSegs,
                start: try asDouble(item["start"]),
                end: try asDouble(item["end"])
            )
            let expectedRaw = try XCTUnwrap(item["expected"] as? [[Any]])
            let expected: [[Double]] = try expectedRaw.map { [try asDouble($0[0]), try asDouble($0[1])] }
            XCTAssertEqual(merged.count, expected.count, item["name"] as? String ?? "")
            for i in 0 ..< expected.count {
                XCTAssertEqual(merged[i][0], expected[i][0], accuracy: 1e-9)
                XCTAssertEqual(merged[i][1], expected[i][1], accuracy: 1e-9)
            }
        }

        for item in try objects(root["resume"]) {
            let segsRaw = try XCTUnwrap(item["watchedSegments"] as? [[Any]])
            let segs: [[Double]] = try segsRaw.map { [try asDouble($0[0]), try asDouble($0[1])] }
            XCTAssertEqual(
                ContentToolPack4Logic.resumePosition(
                    furthestSec: optionalDouble(item["furthestSec"]),
                    watchedSegments: segs
                ),
                try asDouble(item["expected"]),
                accuracy: 1e-9,
                item["name"] as? String ?? ""
            )
        }

        let throttle = try object(root["progressThrottle"])
        XCTAssertEqual(ContentToolPack4Logic.progressThrottleMs, Int64(try asInt(throttle["intervalMs"])))
        for item in try objects(throttle["cases"]) {
            XCTAssertEqual(
                ContentToolPack4Logic.shouldFireProgressThrottle(
                    lastFiredAtMs: optionalInt64(item["lastFiredAtMs"]),
                    nowMs: Int64(try asInt(item["nowMs"]))
                ),
                try XCTUnwrap(item["expected"] as? Bool),
                item["name"] as? String ?? ""
            )
        }

        for item in try objects(root["playbackBlocked"]) {
            let id = try XCTUnwrap(item["checkpointId"] as? String)
            let cp = try XCTUnwrap(cps.first { $0.id == id })
            let ans = answers(from: try object(item["answers"]))
            XCTAssertEqual(
                ContentToolPack4Logic.shouldBlockPlayback(checkpoint: cp, answers: ans),
                try XCTUnwrap(item["expected"] as? Bool),
                item["name"] as? String ?? ""
            )
        }
    }

    func testWorkedExampleMatchesFixture() throws {
        let root = try object(try fixtureRoot()["workedExample"])
        let steps = try objects(root["steps"])
        let allIds = try steps.map { try XCTUnwrap($0["id"] as? String) }

        for item in try objects(root["status"]) {
            let blanked = try stringArray(item["blankedStepIds"])
            let current = try XCTUnwrap(item["currentStepId"] as? String)
            let progressRaw = try object(item["progress"])
            var progress: [String: JSONValue] = [:]
            for (k, v) in progressRaw {
                progress[k] = jsonValue(v) ?? .null
            }
            let expected = try object(item["expected"])
            for stepId in allIds {
                let status = ContentToolPack4Logic.stepStatus(
                    stepId: stepId,
                    blankedStepIds: blanked,
                    currentStepId: current,
                    progress: progress,
                    allStepIds: allIds
                )
                XCTAssertEqual(
                    status.rawValue,
                    try XCTUnwrap(expected[stepId] as? String),
                    "\(item["name"] as? String ?? "")/\(stepId)"
                )
            }
        }

        for item in try objects(root["stepDone"]) {
            XCTAssertEqual(
                ContentToolPack4Logic.isStepDone(jsonValue(item["progress"])),
                try XCTUnwrap(item["expected"] as? Bool),
                item["name"] as? String ?? ""
            )
        }

        for item in try objects(root["canCheck"]) {
            XCTAssertEqual(
                ContentToolPack4Logic.canCheckStep(
                    draft: try XCTUnwrap(item["draft"] as? String),
                    readOnly: try XCTUnwrap(item["readOnly"] as? Bool),
                    busy: try XCTUnwrap(item["busy"] as? Bool),
                    stepDone: try XCTUnwrap(item["stepDone"] as? Bool)
                ),
                try XCTUnwrap(item["expected"] as? Bool),
                item["name"] as? String ?? ""
            )
        }

        for item in try objects(root["resolveCurrent"]) {
            let blanked = try stringArray(item["blankedStepIds"])
            let progressRaw = try object(item["progress"])
            var progress: [String: JSONValue] = [:]
            for (k, v) in progressRaw {
                progress[k] = jsonValue(v) ?? .null
            }
            XCTAssertEqual(
                ContentToolPack4Logic.resolveCurrentStepId(
                    blankedStepIds: blanked,
                    currentStepId: optionalString(item["currentStepId"]),
                    progress: progress
                ),
                try XCTUnwrap(item["expected"] as? String),
                item["name"] as? String ?? ""
            )
        }
    }

    func testParameterExplorerMatchesFixture() throws {
        let root = try object(try fixtureRoot()["parameterExplorer"])
        XCTAssertEqual(ContentToolPack4Logic.recomputeThrottleMs, Int64(try asInt(root["recomputeThrottleMs"])))

        for item in try objects(root["throttle"]) {
            XCTAssertEqual(
                ContentToolPack4Logic.shouldRecompute(
                    lastAtMs: optionalInt64(item["lastAtMs"]),
                    nowMs: Int64(try asInt(item["nowMs"]))
                ),
                try XCTUnwrap(item["expected"] as? Bool),
                item["name"] as? String ?? ""
            )
        }

        for item in try objects(root["settle"]) {
            XCTAssertEqual(
                ContentToolPack4Logic.shouldAutosaveOnSettle(
                    dragging: try XCTUnwrap(item["dragging"] as? Bool),
                    dirty: try XCTUnwrap(item["dirty"] as? Bool)
                ),
                try XCTUnwrap(item["expected"] as? Bool),
                item["name"] as? String ?? ""
            )
        }

        let defaultsRoot = try object(root["defaults"])
        let config = jsonValue(["parameters": defaultsRoot["parameters"] as Any])
        let defaults = ContentToolPack4Logic.defaultParams(from: config)
        let expected = try object(defaultsRoot["expected"])
        XCTAssertEqual(defaults.count, expected.count)
        for (k, v) in expected {
            if let n = v as? NSNumber, CFGetTypeID(n) == CFBooleanGetTypeID() {
                if case .bool(let b) = defaults[k] {
                    XCTAssertEqual(b, n.boolValue, k)
                } else {
                    XCTFail("expected bool for \(k)")
                }
            } else if let n = v as? NSNumber {
                if case .number(let d) = defaults[k] {
                    XCTAssertEqual(d, n.doubleValue, accuracy: 1e-9, k)
                } else {
                    XCTFail("expected number for \(k)")
                }
            } else if let s = v as? String {
                if case .string(let actual) = defaults[k] {
                    XCTAssertEqual(actual, s, k)
                } else {
                    XCTFail("expected string for \(k)")
                }
            }
        }

        for item in try objects(root["clampNumber"]) {
            XCTAssertEqual(
                ContentToolPack4Logic.clampNumber(
                    value: try asDouble(item["value"]),
                    min: try asDouble(item["min"]),
                    max: try asDouble(item["max"]),
                    step: try asDouble(item["step"])
                ),
                try asDouble(item["expected"]),
                accuracy: 1e-9,
                item["name"] as? String ?? ""
            )
        }
    }

    func testUnknownFieldPreservation() throws {
        let root = try object(try fixtureRoot()["unknownFieldPreservation"])
        let baseAny = try object(root["base"])
        var base: [String: JSONValue] = [:]
        for (k, v) in baseAny { base[k] = jsonValue(v) ?? .null }
        let patchAny = try object(root["patch"])
        var patch: [String: JSONValue] = [:]
        for (k, v) in patchAny { patch[k] = jsonValue(v) ?? .null }
        let merged = ContentToolPack4Logic.mergePreservingUnknown(base: base, patch: patch)
        let expectedKeys = Set(try stringArray(root["expectedKeys"]))
        XCTAssertEqual(Set(merged.keys), expectedKeys)
        if case .string(let keep) = merged["customClientKey"] {
            XCTAssertEqual(keep, "keep-me")
        } else {
            XCTFail("customClientKey lost")
        }
    }

    func testMediaProviderReliability() throws {
        for item in try objects(try object(try fixtureRoot()["mediaProvider"])["cases"]) {
            XCTAssertEqual(
                ContentToolPack4Logic.hasReliableCheckpointTiming(
                    source: optionalString(item["source"]),
                    url: optionalString(item["url"]),
                    provider: optionalString(item["provider"])
                ),
                try XCTUnwrap(item["expectedReliable"] as? Bool),
                item["name"] as? String ?? ""
            )
        }
    }

    func testRegistryIncludesPack4NativeTools() {
        for toolId in ContentToolPack4Logic.pack4ToolIds {
            XCTAssertTrue(ContentToolHostLogic.hasNativeRenderer(toolId), toolId)
        }
        XCTAssertFalse(ContentToolHostLogic.hasNativeRenderer("code_sandbox"))
        XCTAssertTrue(ToolRendererRegistry.registeredIds().isSuperset(of: ContentToolPack4Logic.pack4ToolIds))
        XCTAssertFalse(ToolRendererRegistry.registeredIds().contains("code_sandbox"))
    }
}
