import Foundation

/// Pure CT.M6 pack-2 decisions — draft lifecycle, AI error classification,
/// consent gating, discussion control visibility, pagination cursoring,
/// offline action rules, and client allowlist. No networking.
enum ContentToolPack2Logic {
    static let pack2ToolIds: Set<String> = [
        "ask_questions",
        "explain_it_back",
        "inline_discussion",
    ]

    /// Per-tool client allowlist (rollout). Empty entry removes a renderer without a release.
    /// AI tools can be held back independently of `inline_discussion`.
    static var clientAllowlist: Set<String> = pack2ToolIds

    static let defaultPageSize = 20

    enum AIErrorClass: String, Equatable {
        case rateLimited = "rate_limited"
        case budget = "budget"
        case providerUnavailable = "provider_unavailable"
        case filtered = "filtered"
        case optOut = "opt_out"
        case coppa = "coppa"
        case tooShort = "too_short"
        case tooLong = "too_long"
        case maxAttempts = "max_attempts"
        case forbidden = "forbidden"
        case offline = "offline"
        case unknown = "unknown"

        var messageKey: String {
            switch self {
            case .rateLimited: return "mobile.contentTools.ai.error.rateLimited"
            case .budget: return "mobile.contentTools.ai.error.budget"
            case .providerUnavailable: return "mobile.contentTools.ai.error.providerUnavailable"
            case .filtered: return "mobile.contentTools.ai.error.filtered"
            case .optOut: return "mobile.contentTools.ai.error.optOut"
            case .coppa: return "mobile.contentTools.ai.error.coppa"
            case .tooShort: return "mobile.contentTools.tools.explain_it_back.error.tooShort"
            case .tooLong: return "mobile.contentTools.tools.explain_it_back.error.tooLong"
            case .maxAttempts: return "mobile.contentTools.tools.explain_it_back.error.maxAttempts"
            case .forbidden: return "mobile.contentTools.tools.inline_discussion.error.forbidden"
            case .offline: return "mobile.contentTools.runtime.offlineComposer"
            case .unknown: return "mobile.contentTools.runtime.retry"
            }
        }
    }

    enum DraftEvent: String, Equatable {
        case save
        case restore
        case clearOnSuccess
        case retainOnFailure
    }

    struct DiscussionControls: Equatable {
        var canEdit: Bool
        var canDelete: Bool
        var canEndorse: Bool
        var canModerate: Bool
        var canUpvote: Bool
        var canReport: Bool
        var canReply: Bool
    }

    // MARK: - Allowlist / registry

    static func isClientAllowlisted(
        _ toolId: String,
        allowlist: Set<String> = clientAllowlist
    ) -> Bool {
        allowlist.contains(toolId)
    }

    static func allowlistedToolIds(allowlist: Set<String> = clientAllowlist) -> Set<String> {
        pack2ToolIds.intersection(allowlist)
    }

    static func conflictPolicy(for toolId: String) -> ContentToolHostLogic.ConflictPolicy {
        switch toolId {
        case "ask_questions", "explain_it_back":
            return .merge
        default:
            return .serverWins
        }
    }

    /// Pack-2 actions are never queued offline (CT.M3 FR-11 / CT.M6 FR-5).
    static func canQueueActionOffline(toolId: String, action: String) -> Bool {
        _ = toolId
        _ = action
        return false
    }

    // MARK: - Draft lifecycle

    static func draftStorageKey(instanceId: String, slot: String = "composer") -> String {
        "content-tool-draft:\(instanceId):\(slot)"
    }

    /// Decide what to do with a local draft after an action outcome.
    static func draftEventAfterAction(success: Bool, preserveInput: Bool) -> DraftEvent {
        if success { return .clearOnSuccess }
        if preserveInput { return .retainOnFailure }
        return .retainOnFailure
    }

    static func shouldClearDraft(success: Bool) -> Bool { success }

    static func shouldRetainDraftOnFailure(preserveInput: Bool) -> Bool {
        _ = preserveInput
        return true
    }

    // MARK: - AI consent gating

    /// Fail-closed: unknown/unfetched consent blocks AI composers (CT.M6 / CT.M9).
    static func composerAIAllowed(
        disclosureMode: String?,
        decision: String?,
        consentFetched: Bool
    ) -> Bool {
        guard consentFetched else { return false }
        let mode = (disclosureMode ?? "acknowledge").lowercased()
        if mode == "none" { return true }
        let d = (decision ?? "").lowercased()
        if d == "opted_out" { return false }
        if mode == "banner" { return d != "opted_out" }
        // acknowledge
        return d == "acknowledged"
    }

    static func shouldShowAIDisclosure(
        disclosureMode: String?,
        decision: String?,
        consentFetched: Bool
    ) -> Bool {
        guard consentFetched else { return true }
        let mode = (disclosureMode ?? "acknowledge").lowercased()
        if mode == "none" { return false }
        let d = (decision ?? "").lowercased()
        return d != "acknowledged" && d != "opted_out"
    }

    // MARK: - Error classification

    static func classifyAIError(code: String?) -> AIErrorClass {
        switch (code ?? "").lowercased() {
        case "rate_limited": return .rateLimited
        case "budget": return .budget
        case "provider_unavailable": return .providerUnavailable
        case "filtered": return .filtered
        case "opt_out": return .optOut
        case "coppa": return .coppa
        case "too_short", "length": return .tooShort
        case "too_long": return .tooLong
        case "max_attempts": return .maxAttempts
        case "forbidden": return .forbidden
        case "offline": return .offline
        default: return .unknown
        }
    }

    static func plainLanguageMessageKey(for code: String?) -> String {
        classifyAIError(code: code).messageKey
    }

    // MARK: - explain_it_back length guidance

    static func wordCount(_ text: String) -> Int {
        let parts = text
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .split { !$0.isLetter && !$0.isNumber }
        return parts.filter { !$0.isEmpty }.count
    }

    static func lengthGuidanceOK(text: String, minWords: Int, maxWords: Int) -> Bool {
        let count = wordCount(text)
        return count >= minWords && count <= maxWords
    }

    static func canSubmitExplanation(
        text: String,
        minWords: Int,
        maxWords: Int,
        attemptsUsed: Int,
        maxAttempts: Int,
        readOnly: Bool,
        online: Bool,
        consentAllowed: Bool
    ) -> Bool {
        if readOnly || !online || !consentAllowed { return false }
        if attemptsUsed >= maxAttempts { return false }
        return lengthGuidanceOK(text: text, minWords: minWords, maxWords: maxWords)
    }

    // MARK: - ask_questions

    static func canAsk(
        text: String,
        readOnly: Bool,
        online: Bool,
        consentAllowed: Bool,
        busy: Bool
    ) -> Bool {
        if readOnly || !online || !consentAllowed || busy { return false }
        return !text.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
    }

    // MARK: - Discussion controls / pagination

    static func discussionControls(
        isOwn: Bool,
        canEditFlag: Bool,
        canDeleteFlag: Bool,
        allowReplies: Bool,
        viewerCanEndorse: Bool,
        viewerCanModerate: Bool,
        readOnly: Bool,
        removed: Bool
    ) -> DiscussionControls {
        if removed {
            return DiscussionControls(
                canEdit: false,
                canDelete: false,
                canEndorse: viewerCanModerate && !readOnly,
                canModerate: viewerCanModerate && !readOnly,
                canUpvote: false,
                canReport: false,
                canReply: false
            )
        }
        return DiscussionControls(
            canEdit: isOwn && canEditFlag && !readOnly,
            canDelete: isOwn && canDeleteFlag && !readOnly,
            canEndorse: viewerCanEndorse && !readOnly,
            canModerate: viewerCanModerate && !readOnly,
            canUpvote: !readOnly,
            canReport: !isOwn && !readOnly,
            canReply: allowReplies && !readOnly
        )
    }

    static func nextPage(currentPage: Int, pageSize: Int, total: Int?) -> Int? {
        let page = max(1, currentPage)
        let size = max(1, pageSize)
        guard let total, total > page * size else { return nil }
        return page + 1
    }

    static func shouldRenderTombstone(removed: Bool, tombstone: Bool, moderationState: String?) -> Bool {
        if removed || tombstone { return true }
        guard let moderationState else { return false }
        switch moderationState.lowercased() {
        case "hidden", "removed", "flagged":
            return true
        default:
            return true // unknown moderation state → generic tombstone (NFR)
        }
    }

    static func authorDisplay(serverAuthorDisplay: String?, anonymity: String, isOwn: Bool) -> String? {
        // Never invent identity the server withheld (AC-11).
        if anonymity == "anonymous_to_peers", !isOwn {
            return serverAuthorDisplay
        }
        return serverAuthorDisplay
    }

    // MARK: - Composer send gate

    static func composerSendEnabled(
        text: String,
        readOnly: Bool,
        online: Bool,
        busy: Bool,
        consentAllowed: Bool = true
    ) -> Bool {
        if readOnly || !online || busy || !consentAllowed { return false }
        return !text.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
    }

    // MARK: - Unknown-field preservation

    static func mergePreservingUnknown(
        base: [String: JSONValue],
        patch: [String: JSONValue]
    ) -> [String: JSONValue] {
        var out = base
        for (k, v) in patch { out[k] = v }
        return out
    }

    // MARK: - JSON helpers

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
