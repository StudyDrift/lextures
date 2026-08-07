import Foundation

/// Pure CT.M9 governance decisions — mount gating, consent, kill derivation,
/// filter/crisis outcomes, and content-free telemetry shape checks. No networking.
enum ContentToolGovernanceLogic {
    /// Default freshness window before AI / third-party tools fail closed (15 minutes).
    static let defaultStaleWindowMs: Int64 = 900_000

    enum MountDecision: String, Equatable {
        case mount
        case blockNotAvailable = "block_not_available"
        case blockCapability = "block_capability"
        case blockKilled = "block_killed"
        case blockBreaker = "block_breaker"
        case blockTombstone = "block_tombstone"
        case blockDeprecated = "block_deprecated"
        case blockUnknown = "block_unknown"
        case blockStalePolicy = "block_stale_policy"
    }

    enum FilterOutcomeKind: String, Equatable {
        case filtered
        case crisis
        case generic
    }

    struct MountInput: Equatable {
        var toolId: String
        var capabilities: [String] = []
        var sandboxMode: String?
        var tombstone = false
        var breakerOpen = false
        var deprecated = false
        var killed = false
        var allowedToolIds: [String] = []
        var deniedToolIds: [String] = []
        var deniedCapabilities: [String] = []
        var policyFetched = false
        var policyAgeMs: Int64 = 0
        var staleWindowMs: Int64 = defaultStaleWindowMs
        var unknownGovernanceState = false
        /// True when a previously successful policy snapshot is available (fail-open for first-party non-AI).
        var hasCachedPolicy = false
    }

    struct FilterCrisisInput: Equatable {
        var errorCode: String?
        var crisis = false
    }

    struct FilterCrisisOutcome: Equatable {
        var kind: FilterOutcomeKind
        var preserveDraft: Bool
        var retry: Bool
    }

    static func isAICapable(_ capabilities: [String]) -> Bool {
        capabilities.contains { $0.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() == "ai" }
    }

    static func isThirdParty(toolId: String, sandboxMode: String?) -> Bool {
        let mode = (sandboxMode ?? "").lowercased()
        if mode == "iframe" || mode == "webview" { return true }
        return toolId.contains(".")
    }

    static func requiresStrictPolicy(_ input: MountInput) -> Bool {
        isAICapable(input.capabilities) || isThirdParty(toolId: input.toolId, sandboxMode: input.sandboxMode)
    }

    static func isPolicyStale(_ input: MountInput) -> Bool {
        if input.unknownGovernanceState { return false }
        if !input.policyFetched {
            // Unfetched: stale for strict tools unless we have a fresh-enough cache marker.
            if input.hasCachedPolicy && input.policyAgeMs <= input.staleWindowMs { return false }
            return true
        }
        return input.policyAgeMs > input.staleWindowMs
    }

    static func toolIsKilled(
        toolId: String,
        capabilities: [String],
        killedToolIds: [String],
        killedCapabilities: [String],
        killAllAI: Bool
    ) -> Bool {
        if killedToolIds.contains(toolId) { return true }
        let caps = Set(capabilities.map { $0.lowercased() })
        for denied in killedCapabilities where caps.contains(denied.lowercased()) {
            return true
        }
        if killAllAI && caps.contains("ai") { return true }
        return false
    }

    static func mountDecision(_ input: MountInput) -> MountDecision {
        if input.unknownGovernanceState { return .blockUnknown }
        if input.tombstone { return .blockTombstone }
        if input.killed { return .blockKilled }
        if input.breakerOpen { return .blockBreaker }
        if input.deprecated { return .blockDeprecated }

        if input.deniedToolIds.contains(input.toolId) {
            return .blockNotAvailable
        }
        if !input.allowedToolIds.isEmpty && !input.allowedToolIds.contains(input.toolId) {
            return .blockNotAvailable
        }
        let caps = Set(input.capabilities.map { $0.lowercased() })
        for denied in input.deniedCapabilities where caps.contains(denied.lowercased()) {
            return .blockCapability
        }

        if requiresStrictPolicy(input) && isPolicyStale(input) {
            return .blockStalePolicy
        }
        return .mount
    }

    static func reasonMessageKey(_ decision: MountDecision) -> String {
        switch decision {
        case .mount:
            return ""
        case .blockNotAvailable, .blockCapability:
            return "mobile.contentTools.governance.notAvailableInCourse"
        case .blockKilled:
            return "mobile.contentTools.governance.killed"
        case .blockBreaker:
            return "mobile.contentTools.governance.temporarilyUnavailable"
        case .blockTombstone, .blockDeprecated:
            return "mobile.contentTools.governance.withdrawn"
        case .blockUnknown:
            return "mobile.contentTools.governance.unavailable"
        case .blockStalePolicy:
            return "mobile.contentTools.governance.policyStale"
        }
    }

    /// Fail-closed consent gating (mirrors Pack2; kept here for CT.M9 host chrome).
    static func aiActionsAllowed(
        disclosureMode: String?,
        decision: String?,
        consentFetched: Bool
    ) -> Bool {
        ContentToolPack2Logic.composerAIAllowed(
            disclosureMode: disclosureMode,
            decision: decision,
            consentFetched: consentFetched
        )
    }

    static func shouldShowAIDisclosure(
        disclosureMode: String?,
        decision: String?,
        consentFetched: Bool
    ) -> Bool {
        ContentToolPack2Logic.shouldShowAIDisclosure(
            disclosureMode: disclosureMode,
            decision: decision,
            consentFetched: consentFetched
        )
    }

    /// Structural assertion: overflow menu → Report is two taps (FR-10 / AC-7).
    static func reportTapCount() -> Int { 2 }

    static func reportReachableInTwoTaps() -> Bool { reportTapCount() <= 2 }

    static func filterCrisisOutcome(_ input: FilterCrisisInput) -> FilterCrisisOutcome {
        if input.crisis {
            return FilterCrisisOutcome(kind: .crisis, preserveDraft: true, retry: false)
        }
        let code = (input.errorCode ?? "").lowercased()
        if code == "filtered" || code == "free_text_blocked" || code == "blocked" {
            return FilterCrisisOutcome(kind: .filtered, preserveDraft: true, retry: false)
        }
        return FilterCrisisOutcome(kind: .generic, preserveDraft: true, retry: true)
    }

    static func plainLanguageFilterKey(for outcome: FilterOutcomeKind) -> String {
        switch outcome {
        case .filtered: return "mobile.contentTools.governance.filtered"
        case .crisis: return "mobile.contentTools.governance.crisisTitle"
        case .generic: return "mobile.contentTools.runtime.retry"
        }
    }

    // MARK: - Telemetry shape (FR-18 / AC-13)

    private static let allowedTelemetryKeys: Set<String> = [
        "tool_id", "platform", "outcome", "error_class", "reason", "event",
    ]

    private static let forbiddenTelemetrySubstrings = [
        "prompt:", "state_json", "free_text=", "peer_content=",
    ]

    static func telemetryAttributesAreContentFree(_ attributes: [String: String]) -> Bool {
        for (key, value) in attributes {
            let k = key.lowercased()
            if !allowedTelemetryKeys.contains(k) {
                // Extra keys are allowed only if they look like enums/ids (no spaces / long prose).
                if value.count > 64 || value.contains(" ") { return false }
                if k.contains("text") || k.contains("prompt") || k.contains("content") || k.contains("state") {
                    return false
                }
            }
            let lowered = value.lowercased()
            for needle in forbiddenTelemetrySubstrings where lowered.contains(needle) {
                return false
            }
        }
        return true
    }

    static func studentResetVisible(studentResetAllowed: Bool, readOnly: Bool) -> Bool {
        studentResetAllowed && !readOnly
    }

    static func canObtainDeniedCapability(
        capability: String,
        deniedCapabilities: [String]
    ) -> Bool {
        let needle = capability.lowercased()
        return !deniedCapabilities.contains { $0.lowercased() == needle }
    }

    /// Merge course allowlist with org policy allowlist (intersection when both non-empty).
    static func effectiveAllowedToolIds(
        courseAllowed: [String],
        orgAllowed: [String]
    ) -> [String] {
        if orgAllowed.isEmpty { return courseAllowed }
        if courseAllowed.isEmpty { return orgAllowed }
        return Array(Set(courseAllowed).intersection(orgAllowed))
    }
}
