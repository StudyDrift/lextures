// swiftlint:disable identifier_name large_tuple force_try type_body_length file_length
import Foundation

/// Pure CT.M8 pack-4 decisions — checkpoint scheduling/seek clamp, resume position,
/// worked-example step machine, slider throttle/settle, and client allowlist. No networking.
enum ContentToolPack4Logic {
    /// Native pack-4 tools (registered in CT.M3 registry).
    static let pack4ToolIds: Set<String> = [
        "media_checkpoints",
        "worked_example",
        "parameter_explorer",
    ]

    /// Delivered via CT.M4 sandbox only — never native-registered.
    static let sandboxToolIds: Set<String> = [
        "code_sandbox",
    ]

    /// Per-tool client allowlist (rollout). Empty entry removes a renderer without a release.
    static var clientAllowlist: Set<String> = pack4ToolIds

    static let checkpointToleranceSec = 0.25
    static let progressThrottleMs: Int64 = 5_000
    static let recomputeThrottleMs: Int64 = 33 // ≤ 30 Hz
    static let defaultCheckpointAttempts = 2
    static let defaultAttemptsPerStep = 3
    static let segmentGranularitySec = 5.0

    // MARK: - Types

    struct Checkpoint: Equatable {
        var id: String
        var atSec: Double
        var required: Bool
        var attempts: Int
    }

    struct CheckpointAnswer: Equatable {
        var done: Bool
        var attemptCount: Int
        var lastCorrect: Bool
    }

    struct SeekClamp: Equatable {
        var time: Double
        var clamped: Bool
    }

    enum StepStatus: String, Equatable {
        case current
        case solved
        case revealed
        case locked
        case scaffolded
        case allComplete = "all_complete"
    }

    struct AnswerResultView: Equatable {
        var correct: Bool?
        var feedback: String?
        var attemptsRemaining: Int?
        var done: Bool?
        var error: String?
        var message: String?
        var checkpointId: String?
    }

    struct CheckStepResultView: Equatable {
        var result: String?
        var feedback: String?
        var attemptsRemaining: Int?
        var canReveal: Bool?
        var nextStep: String?
        var stepId: String?
        var error: String?
        var message: String?
    }

    struct HintResultView: Equatable {
        var hint: String?
        var hintsRemaining: Int?
        var level: Int?
        var noMoreHints: Bool?
        var stepId: String?
        var error: String?
    }

    // MARK: - Allowlist / registry

    static func isClientAllowlisted(
        _ toolId: String,
        allowlist: Set<String> = clientAllowlist
    ) -> Bool {
        allowlist.contains(toolId)
    }

    static func allowlistedToolIds(allowlist: Set<String> = clientAllowlist) -> Set<String> {
        pack4ToolIds.intersection(allowlist)
    }

    static func conflictPolicy(for toolId: String) -> ContentToolHostLogic.ConflictPolicy {
        // All pack-4 tools (native + sandbox) use server_wins.
        switch toolId {
        case "media_checkpoints", "worked_example", "parameter_explorer", "code_sandbox":
            return .serverWins
        default:
            return .serverWins
        }
    }

    /// Pack-4 actions are never queued offline (CT.M3 FR-11).
    static func canQueueActionOffline(toolId: String, action: String) -> Bool {
        _ = toolId
        _ = action
        return false
    }

    // MARK: - Media checkpoints

    static func parseCheckpoints(_ config: JSONValue?) -> [Checkpoint] {
        arrayField(config, key: "checkpoints").compactMap { raw in
            let obj = objectMap(raw)
            guard case .string(let id) = obj["id"] else { return nil }
            let atSec = numberField(raw, key: "atSec") ?? 0
            let required = boolField(raw, key: "required") ?? true
            let attemptsRaw = numberField(raw, key: "attempts").map { Int($0.rounded()) }
                ?? defaultCheckpointAttempts
            let attempts = min(10, max(1, attemptsRaw))
            return Checkpoint(id: id, atSec: atSec, required: required, attempts: attempts)
        }
        .sorted { $0.atSec < $1.atSec }
    }

    static func parseAnswers(_ state: JSONValue?) -> [String: CheckpointAnswer] {
        let answersObj = objectMap(objectMap(state)["answers"])
        var out: [String: CheckpointAnswer] = [:]
        for (id, raw) in answersObj {
            let obj = objectMap(raw)
            let done = boolField(raw, key: "done") ?? false
            let attempts = arrayField(raw, key: "attempts")
            let lastCorrect: Bool
            if let last = attempts.last {
                lastCorrect = boolField(last, key: "correct") ?? false
            } else {
                lastCorrect = false
            }
            out[id] = CheckpointAnswer(
                done: done,
                attemptCount: attempts.count,
                lastCorrect: lastCorrect
            )
        }
        return out
    }

    static func isCheckpointDone(answers: [String: CheckpointAnswer], checkpoint: Checkpoint) -> Bool {
        guard let ans = answers[checkpoint.id] else { return false }
        if ans.done { return true }
        if ans.lastCorrect { return true }
        return ans.attemptCount >= checkpoint.attempts
    }

    static func findDueCheckpoint(
        checkpoints: [Checkpoint],
        answers: [String: CheckpointAnswer],
        currentTime: Double,
        alreadyPromptedIds: Set<String>
    ) -> Checkpoint? {
        for cp in checkpoints {
            if alreadyPromptedIds.contains(cp.id) { continue }
            if isCheckpointDone(answers: answers, checkpoint: cp) { continue }
            if currentTime + checkpointToleranceSec >= cp.atSec && currentTime + 2 >= cp.atSec {
                if currentTime >= cp.atSec - checkpointToleranceSec {
                    return cp
                }
            }
        }
        return nil
    }

    static func earliestUnansweredRequiredSec(
        checkpoints: [Checkpoint],
        answers: [String: CheckpointAnswer]
    ) -> Double? {
        var earliest: Double?
        for cp in checkpoints {
            if !cp.required { continue }
            if isCheckpointDone(answers: answers, checkpoint: cp) { continue }
            if earliest == nil || cp.atSec < earliest! {
                earliest = cp.atSec
            }
        }
        return earliest
    }

    static func clampSeekTime(
        preventSkip: Bool,
        checkpoints: [Checkpoint],
        answers: [String: CheckpointAnswer],
        targetSec: Double
    ) -> SeekClamp {
        guard preventSkip else { return SeekClamp(time: targetSec, clamped: false) }
        guard let limit = earliestUnansweredRequiredSec(checkpoints: checkpoints, answers: answers) else {
            return SeekClamp(time: targetSec, clamped: false)
        }
        if targetSec > limit + 0.05 {
            return SeekClamp(time: limit, clamped: true)
        }
        return SeekClamp(time: targetSec, clamped: false)
    }

    static func mergeLocalSegments(
        existing: [[Double]],
        start: Double,
        end: Double,
        granularity: Double = segmentGranularitySec
    ) -> [[Double]] {
        guard end > start else { return existing }
        func floorG(_ v: Double) -> Double {
            floor(max(0, v) / granularity) * granularity
        }
        func ceilG(_ v: Double) -> Double {
            let f = floorG(v)
            return v > f ? f + granularity : f
        }
        var s = floorG(start)
        var e = ceilG(end)
        if e <= s { e = s + granularity }
        var all = existing + [[s, e]]
        all.sort { a, b in
            if a[0] != b[0] { return a[0] < b[0] }
            return a[1] < b[1]
        }
        var merged: [[Double]] = []
        for seg in all {
            if let last = merged.last, seg[0] <= last[1] {
                if seg[1] > last[1] {
                    merged[merged.count - 1] = [last[0], seg[1]]
                }
            } else {
                merged.append(seg)
            }
        }
        return merged
    }

    static func resumePosition(furthestSec: Double?, watchedSegments: [[Double]]) -> Double {
        var best = furthestSec ?? 0
        for seg in watchedSegments {
            if seg.count >= 2 {
                best = max(best, seg[1])
            }
        }
        return max(0, best)
    }

    static func shouldFireProgressThrottle(lastFiredAtMs: Int64?, nowMs: Int64, intervalMs: Int64 = progressThrottleMs) -> Bool {
        guard let last = lastFiredAtMs else { return true }
        return nowMs - last >= intervalMs
    }

    static func shouldBlockPlayback(
        checkpoint: Checkpoint,
        answers: [String: CheckpointAnswer]
    ) -> Bool {
        guard checkpoint.required else { return false }
        return !isCheckpointDone(answers: answers, checkpoint: checkpoint)
    }

    /// Direct media / course files are reliable; YouTube/Vimeo embeds are not for checkpoints.
    static func hasReliableCheckpointTiming(
        source: String?,
        url: String?,
        provider: String?
    ) -> Bool {
        let trimmed = url?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        guard !trimmed.isEmpty else { return false }
        let p = (provider ?? "").lowercased()
        if p == "youtube" || p == "vimeo" { return false }
        let lower = trimmed.lowercased()
        if lower.contains("youtube.com") || lower.contains("youtu.be") || lower.contains("vimeo.com") {
            return false
        }
        _ = source
        return true
    }

    static func parseWatchedSegments(_ state: JSONValue?) -> [[Double]] {
        arrayField(state, key: "watchedSegments").compactMap { raw in
            guard case .array(let arr) = raw, arr.count >= 2 else { return nil }
            guard case .number(let a) = arr[0], case .number(let b) = arr[1] else { return nil }
            return [a, b]
        }
    }

    static func parseAnswerResult(_ value: JSONValue?) -> AnswerResultView {
        AnswerResultView(
            correct: boolField(value, key: "correct"),
            feedback: stringField(value, key: "feedback"),
            attemptsRemaining: numberField(value, key: "attemptsRemaining").map { Int($0.rounded()) },
            done: boolField(value, key: "done"),
            error: stringField(value, key: "error"),
            message: stringField(value, key: "message"),
            checkpointId: stringField(value, key: "checkpointId")
        )
    }

    // MARK: - Worked example

    static func parseStepIds(_ config: JSONValue?) -> [String] {
        arrayField(config, key: "steps").compactMap { raw in
            stringField(raw, key: "id")
        }
    }

    static func blankedStepIds(config: JSONValue?, state: JSONValue?) -> [String] {
        let fromState = arrayField(state, key: "blankedStepIds").compactMap { raw -> String? in
            if case .string(let s) = raw { return s }
            return nil
        }
        if !fromState.isEmpty { return fromState }
        return arrayField(config, key: "steps").compactMap { raw in
            let obj = objectMap(raw)
            guard obj["blank"] != nil, case .string(let id) = obj["id"] else { return nil }
            return id
        }
    }

    static func isStepDone(_ progress: JSONValue?) -> Bool {
        if boolField(progress, key: "revealed") == true { return true }
        if stringField(progress, key: "completedAt") != nil { return true }
        let attempts = arrayField(progress, key: "attempts")
        guard let last = attempts.last else { return false }
        let result = stringField(last, key: "result")
        return result == "correct" || result == "needs_review"
    }

    static func stepProgressMap(_ state: JSONValue?) -> [String: JSONValue] {
        objectMap(objectMap(state)["steps"])
    }

    static func resolveCurrentStepId(
        blankedStepIds: [String],
        currentStepId: String?,
        progress: [String: JSONValue]
    ) -> String {
        if let currentStepId, !currentStepId.isEmpty {
            return currentStepId
        }
        for id in blankedStepIds {
            if !isStepDone(progress[id]) { return id }
        }
        return blankedStepIds.first ?? ""
    }

    static func stepStatus(
        stepId: String,
        blankedStepIds: [String],
        currentStepId: String,
        progress: [String: JSONValue],
        allStepIds: [String]
    ) -> StepStatus {
        let blanked = Set(blankedStepIds)
        if !blanked.contains(stepId) {
            return .scaffolded
        }
        let sp = progress[stepId]
        if boolField(sp, key: "revealed") == true {
            return .revealed
        }
        if isStepDone(sp) {
            return .solved
        }
        if stepId == currentStepId {
            return .current
        }
        let currentIndex = blankedStepIds.firstIndex(of: currentStepId) ?? 0
        let stepIndex = blankedStepIds.firstIndex(of: stepId) ?? Int.max
        if stepIndex > currentIndex {
            return .locked
        }
        _ = allStepIds
        return .locked
    }

    static func canCheckStep(draft: String, readOnly: Bool, busy: Bool, stepDone: Bool) -> Bool {
        if readOnly || busy || stepDone { return false }
        return !draft.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
    }

    static func mergeStepDraft(
        state: [String: JSONValue],
        stepId: String,
        draft: String
    ) -> [String: JSONValue] {
        var steps = objectMap(state["steps"])
        var sp = objectMap(steps[stepId])
        sp["draft"] = .string(draft)
        steps[stepId] = .object(sp)
        return mergePreservingUnknown(base: state, patch: [
            "v": state["v"] ?? .number(1),
            "steps": .object(steps),
        ])
    }

    static func parseCheckStepResult(_ value: JSONValue?) -> CheckStepResultView {
        CheckStepResultView(
            result: stringField(value, key: "result"),
            feedback: stringField(value, key: "feedback"),
            attemptsRemaining: numberField(value, key: "attemptsRemaining").map { Int($0.rounded()) },
            canReveal: boolField(value, key: "canReveal"),
            nextStep: stringField(value, key: "nextStep"),
            stepId: stringField(value, key: "stepId"),
            error: stringField(value, key: "error"),
            message: stringField(value, key: "message")
        )
    }

    static func parseHintResult(_ value: JSONValue?) -> HintResultView {
        HintResultView(
            hint: stringField(value, key: "hint"),
            hintsRemaining: numberField(value, key: "hintsRemaining").map { Int($0.rounded()) },
            level: numberField(value, key: "level").map { Int($0.rounded()) },
            noMoreHints: boolField(value, key: "noMoreHints"),
            stepId: stringField(value, key: "stepId"),
            error: stringField(value, key: "error")
        )
    }

    // MARK: - Parameter explorer

    static func shouldRecompute(lastAtMs: Int64?, nowMs: Int64, intervalMs: Int64 = recomputeThrottleMs) -> Bool {
        guard let last = lastAtMs else { return true }
        return nowMs - last >= intervalMs
    }

    static func shouldAutosaveOnSettle(dragging: Bool, dirty: Bool) -> Bool {
        !dragging && dirty
    }

    static func defaultParams(from config: JSONValue?) -> [String: JSONValue] {
        var out: [String: JSONValue] = [:]
        for raw in arrayField(config, key: "parameters") {
            let obj = objectMap(raw)
            guard case .string(let id) = obj["id"], let def = obj["default"] else { continue }
            out[id] = def
        }
        return out
    }

    static func clampNumber(value: Double, min: Double, max: Double, step: Double) -> Double {
        let lo = Swift.min(min, max)
        let hi = Swift.max(min, max)
        var v = Swift.max(lo, Swift.min(hi, value))
        if step > 0 {
            let steps = ((v - lo) / step).rounded()
            v = lo + steps * step
            v = Swift.max(lo, Swift.min(hi, v))
        }
        return v
    }

    static func mergeParamsPreservingUnknown(
        state: [String: JSONValue],
        params: [String: JSONValue]
    ) -> [String: JSONValue] {
        mergePreservingUnknown(base: state, patch: [
            "v": state["v"] ?? .number(1),
            "params": .object(params),
        ])
    }

    // MARK: - JSON helpers

    static func mergePreservingUnknown(
        base: [String: JSONValue],
        patch: [String: JSONValue]
    ) -> [String: JSONValue] {
        var out = base
        for (k, v) in patch { out[k] = v }
        return out
    }

    static func objectMap(_ value: JSONValue?) -> [String: JSONValue] {
        guard case .object(let obj) = value else { return [:] }
        return obj
    }

    static func arrayField(_ value: JSONValue?, key: String) -> [JSONValue] {
        guard case .object(let obj) = value, case .array(let arr) = obj[key] else { return [] }
        return arr
    }

    static func boolField(_ value: JSONValue?, key: String) -> Bool? {
        guard case .object(let obj) = value, let field = obj[key] else { return nil }
        if case .bool(let flag) = field { return flag }
        return nil
    }

    static func numberField(_ value: JSONValue?, key: String) -> Double? {
        guard case .object(let obj) = value, let field = obj[key] else { return nil }
        if case .number(let number) = field { return number }
        if case .string(let text) = field { return Double(text) }
        return nil
    }

    static func stringField(_ value: JSONValue?, key: String) -> String? {
        ContentToolHostLogic.stringField(value, key: key)
    }
}
