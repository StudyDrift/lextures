import Foundation

/// Pure CT.M5 pack-1 decisions — attempt/reveal gating, commit immutability,
/// Class Pulse poll backoff, offline action queue ordering, client allowlist,
/// and Review/SRS cache invalidation keys. No networking.
enum ContentToolPack1Logic {
    static let pack1ToolIds: Set<String> = [
        "inline_questions",
        "predict_reveal",
        "class_pulse",
        "flashcards",
    ]

    /// Per-tool client allowlist (rollout). Empty entry removes a renderer without a release.
    static var clientAllowlist: Set<String> = pack1ToolIds

    static let classPulsePollIntervalMs = 30_000
    static let classPulseMaxBackoffMs = 120_000
    static let flashcardRatings = ["again", "hard", "good", "easy"]

    struct PendingAction: Equatable {
        var instanceId: String
        var toolId: String
        var action: String
        var sequence: Int64
        var payloadJSON: String
    }

    // MARK: - Allowlist / registry

    static func isClientAllowlisted(
        _ toolId: String,
        allowlist: Set<String> = clientAllowlist
    ) -> Bool {
        allowlist.contains(toolId)
    }

    static func allowlistedToolIds(allowlist: Set<String> = clientAllowlist) -> Set<String> {
        pack1ToolIds.intersection(allowlist)
    }

    static func conflictPolicy(for toolId: String) -> ContentToolHostLogic.ConflictPolicy {
        if toolId == "flashcards" { return .merge }
        return .serverWins
    }

    // MARK: - Offline action queue (tool-local; CT.M3 host never queues actions)

    /// Pack-1 exceptions: `submit` and `rate` may queue; vote/commit/aggregate never do.
    static func canQueueActionOffline(toolId: String, action: String) -> Bool {
        switch (toolId, action) {
        case ("inline_questions", "submit"), ("flashcards", "rate"):
            return true
        default:
            return false
        }
    }

    static func orderPendingActions(_ items: [PendingAction]) -> [PendingAction] {
        items.sorted {
            if $0.instanceId != $1.instanceId { return $0.instanceId < $1.instanceId }
            return $0.sequence < $1.sequence
        }
    }

    // MARK: - inline_questions

    static func parseAttemptsConfig(_ raw: JSONValue?) -> Int? {
        // nil = unlimited
        guard let raw else { return 2 }
        switch raw {
        case .string(let text) where text.lowercased() == "unlimited":
            return nil
        case .string(let text):
            return Int(text).map { max(1, $0) } ?? 2
        case .number(let number):
            return max(1, Int(number.rounded()))
        default:
            return 2
        }
    }

    static func attemptsUsed(answers: [String: JSONValue], questionId: String) -> Int {
        guard case .object(let question)? = answers[questionId],
              case .array(let attempts)? = question["attempts"]
        else { return 0 }
        return attempts.count
    }

    static func canSubmit(
        answers: [String: JSONValue],
        questionId: String,
        maxAttempts: Int?,
        readOnly: Bool
    ) -> Bool {
        if readOnly { return false }
        guard let maxAttempts else { return true }
        return attemptsUsed(answers: answers, questionId: questionId) < maxAttempts
    }

    static func isSequentiallyUnlocked(
        questions: [String],
        answers: [String: JSONValue],
        questionId: String,
        sequential: Bool
    ) -> Bool {
        guard sequential else { return true }
        for qid in questions {
            if qid == questionId { return true }
            if attemptsUsed(answers: answers, questionId: qid) == 0 { return false }
        }
        return false
    }

    static func shuffleStable<T>(_ items: [T], seed: String, id: (T) -> String) -> [T] {
        guard items.count > 1 else { return items }
        var out = items
        var h: UInt32 = 0
        for byte in seed.utf8 {
            h = h &* 31 &+ UInt32(byte)
        }
        if out.count > 1 {
            for i in stride(from: out.count - 1, through: 1, by: -1) {
                h = h &* 1_103_515_245 &+ 12_345
                let j = Int(h % UInt32(i + 1))
                out.swapAt(i, j)
            }
        }
        _ = id
        return out
    }

    // MARK: - predict_reveal

    static func isCommitted(_ state: JSONValue?) -> Bool {
        guard let committed = ContentToolHostLogic.stringField(state, key: "committedAt"),
              !committed.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
        else { return false }
        return true
    }

    /// Reveal is gated on server-confirmed commit — never on local draft alone.
    static func canShowReveal(committed: Bool, hasRevealPayload: Bool) -> Bool {
        committed && hasRevealPayload
    }

    static func canEditPrediction(committed: Bool, readOnly: Bool) -> Bool {
        !committed && !readOnly
    }

    // MARK: - class_pulse

    static func hasVoted(votes: [JSONValue], round: Int = 1) -> Bool {
        votes.contains { vote in
            guard case .object(let obj) = vote else { return false }
            if case .number(let roundNumber) = obj["round"] { return Int(roundNumber) == round }
            if case .string(let roundText) = obj["round"], let parsed = Int(roundText) {
                return parsed == round
            }
            return false
        }
    }

    static func shouldPollAggregate(visible: Bool, hasVoted: Bool) -> Bool {
        visible && hasVoted
    }

    /// Visibility-gated poll with linear backoff on failures (caps at 120s).
    static func nextPollDelayMs(consecutiveFailures: Int, baseMs: Int = classPulsePollIntervalMs) -> Int {
        if consecutiveFailures <= 0 { return baseMs }
        let factor = 1 << min(consecutiveFailures, 3)
        return min(classPulseMaxBackoffMs, baseMs * factor)
    }

    // MARK: - flashcards / Review reconciliation

    static func isValidRating(_ rating: String) -> Bool {
        flashcardRatings.contains(rating.lowercased())
    }

    /// Same SRS queue as Review — invalidate these keys after a session so due/streak refresh.
    static func reviewCacheKeysToInvalidate() -> [String] {
        [OfflineCacheKey.reviewQueue(), OfflineCacheKey.reviewStats()]
    }

    static func shouldDoubleCountReviewSubmit(toolId: String) -> Bool {
        // Never call POST /learners/.../review from the flashcards tool.
        toolId != "flashcards"
    }

    // MARK: - Unknown-field preservation

    /// Apply known keys without dropping unknown server fields (NFR backward compat).
    static func mergePreservingUnknown(
        base: [String: JSONValue],
        patch: [String: JSONValue]
    ) -> [String: JSONValue] {
        var out = base
        for (k, v) in patch { out[k] = v }
        return out
    }

    // MARK: - JSON helpers used by renderers

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
}
