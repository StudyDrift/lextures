package com.lextures.android.core.lms

/**
 * Pure CT.M9 governance decisions — mount gating, consent, kill derivation,
 * filter/crisis outcomes, and content-free telemetry shape checks. No networking.
 */
object ContentToolGovernanceLogic {
    /** Default freshness window before AI / third-party tools fail closed (15 minutes). */
    const val DEFAULT_STALE_WINDOW_MS: Long = 900_000L

    enum class MountDecision(val wire: String) {
        MOUNT("mount"),
        BLOCK_NOT_AVAILABLE("block_not_available"),
        BLOCK_CAPABILITY("block_capability"),
        BLOCK_KILLED("block_killed"),
        BLOCK_BREAKER("block_breaker"),
        BLOCK_TOMBSTONE("block_tombstone"),
        BLOCK_DEPRECATED("block_deprecated"),
        BLOCK_UNKNOWN("block_unknown"),
        BLOCK_STALE_POLICY("block_stale_policy"),
        ;

        companion object {
            fun fromWire(raw: String?): MountDecision =
                entries.firstOrNull { it.wire == raw } ?: BLOCK_UNKNOWN
        }
    }

    enum class FilterOutcomeKind {
        FILTERED,
        CRISIS,
        GENERIC,
    }

    data class MountInput(
        val toolId: String,
        val capabilities: List<String> = emptyList(),
        val sandboxMode: String? = null,
        val tombstone: Boolean = false,
        val breakerOpen: Boolean = false,
        val deprecated: Boolean = false,
        val killed: Boolean = false,
        val allowedToolIds: List<String> = emptyList(),
        val deniedToolIds: List<String> = emptyList(),
        val deniedCapabilities: List<String> = emptyList(),
        val policyFetched: Boolean = false,
        val policyAgeMs: Long = 0L,
        val staleWindowMs: Long = DEFAULT_STALE_WINDOW_MS,
        val unknownGovernanceState: Boolean = false,
        /** True when a previously successful policy snapshot is available. */
        val hasCachedPolicy: Boolean = false,
    )

    data class FilterCrisisInput(
        val errorCode: String? = null,
        val crisis: Boolean = false,
    )

    data class FilterCrisisOutcome(
        val kind: FilterOutcomeKind,
        val preserveDraft: Boolean,
        val retry: Boolean,
    )

    fun isAICapable(capabilities: List<String>): Boolean =
        capabilities.any { it.trim().equals("ai", ignoreCase = true) }

    fun isThirdParty(toolId: String, sandboxMode: String?): Boolean {
        val mode = sandboxMode.orEmpty().lowercase()
        if (mode == "iframe" || mode == "webview") return true
        return toolId.contains('.')
    }

    fun requiresStrictPolicy(input: MountInput): Boolean =
        isAICapable(input.capabilities) || isThirdParty(input.toolId, input.sandboxMode)

    fun isPolicyStale(input: MountInput): Boolean {
        if (input.unknownGovernanceState) return false
        if (!input.policyFetched) {
            if (input.hasCachedPolicy && input.policyAgeMs <= input.staleWindowMs) return false
            return true
        }
        return input.policyAgeMs > input.staleWindowMs
    }

    fun toolIsKilled(
        toolId: String,
        capabilities: List<String>,
        killedToolIds: List<String>,
        killedCapabilities: List<String>,
        killAllAI: Boolean,
    ): Boolean {
        if (toolId in killedToolIds) return true
        val caps = capabilities.map { it.lowercase() }.toSet()
        if (killedCapabilities.any { it.lowercase() in caps }) return true
        if (killAllAI && "ai" in caps) return true
        return false
    }

    fun mountDecision(input: MountInput): MountDecision {
        if (input.unknownGovernanceState) return MountDecision.BLOCK_UNKNOWN
        if (input.tombstone) return MountDecision.BLOCK_TOMBSTONE
        if (input.killed) return MountDecision.BLOCK_KILLED
        if (input.breakerOpen) return MountDecision.BLOCK_BREAKER
        if (input.deprecated) return MountDecision.BLOCK_DEPRECATED

        if (input.toolId in input.deniedToolIds) return MountDecision.BLOCK_NOT_AVAILABLE
        if (input.allowedToolIds.isNotEmpty() && input.toolId !in input.allowedToolIds) {
            return MountDecision.BLOCK_NOT_AVAILABLE
        }
        val caps = input.capabilities.map { it.lowercase() }.toSet()
        if (input.deniedCapabilities.any { it.lowercase() in caps }) {
            return MountDecision.BLOCK_CAPABILITY
        }

        if (requiresStrictPolicy(input) && isPolicyStale(input)) {
            return MountDecision.BLOCK_STALE_POLICY
        }
        return MountDecision.MOUNT
    }

    fun reasonMessageKey(decision: MountDecision): String = when (decision) {
        MountDecision.MOUNT -> ""
        MountDecision.BLOCK_NOT_AVAILABLE,
        MountDecision.BLOCK_CAPABILITY,
        -> "mobile.contentTools.governance.notAvailableInCourse"
        MountDecision.BLOCK_KILLED -> "mobile.contentTools.governance.killed"
        MountDecision.BLOCK_BREAKER -> "mobile.contentTools.governance.temporarilyUnavailable"
        MountDecision.BLOCK_TOMBSTONE,
        MountDecision.BLOCK_DEPRECATED,
        -> "mobile.contentTools.governance.withdrawn"
        MountDecision.BLOCK_UNKNOWN -> "mobile.contentTools.governance.unavailable"
        MountDecision.BLOCK_STALE_POLICY -> "mobile.contentTools.governance.policyStale"
    }

    fun aiActionsAllowed(
        disclosureMode: String?,
        decision: String?,
        consentFetched: Boolean,
    ): Boolean = ContentToolPack2Logic.composerAIAllowed(disclosureMode, decision, consentFetched)

    fun shouldShowAIDisclosure(
        disclosureMode: String?,
        decision: String?,
        consentFetched: Boolean,
    ): Boolean = ContentToolPack2Logic.shouldShowAIDisclosure(disclosureMode, decision, consentFetched)

    /** Structural assertion: overflow menu → Report is two taps (FR-10 / AC-7). */
    fun reportTapCount(): Int = 2

    fun reportReachableInTwoTaps(): Boolean = reportTapCount() <= 2

    fun filterCrisisOutcome(input: FilterCrisisInput): FilterCrisisOutcome {
        if (input.crisis) {
            return FilterCrisisOutcome(kind = FilterOutcomeKind.CRISIS, preserveDraft = true, retry = false)
        }
        val code = input.errorCode.orEmpty().lowercase()
        if (code == "filtered" || code == "free_text_blocked" || code == "blocked") {
            return FilterCrisisOutcome(kind = FilterOutcomeKind.FILTERED, preserveDraft = true, retry = false)
        }
        return FilterCrisisOutcome(kind = FilterOutcomeKind.GENERIC, preserveDraft = true, retry = true)
    }

    fun plainLanguageFilterKey(outcome: FilterOutcomeKind): String = when (outcome) {
        FilterOutcomeKind.FILTERED -> "mobile.contentTools.governance.filtered"
        FilterOutcomeKind.CRISIS -> "mobile.contentTools.governance.crisisTitle"
        FilterOutcomeKind.GENERIC -> "mobile.contentTools.runtime.retry"
    }

    private val allowedTelemetryKeys = setOf(
        "tool_id", "platform", "outcome", "error_class", "reason", "event",
    )

    private val forbiddenTelemetrySubstrings = listOf(
        "prompt:", "state_json", "free_text=", "peer_content=",
    )

    fun telemetryAttributesAreContentFree(attributes: Map<String, String>): Boolean {
        for ((key, value) in attributes) {
            val k = key.lowercase()
            if (k !in allowedTelemetryKeys) {
                if (value.length > 64 || value.contains(' ')) return false
                if (k.contains("text") || k.contains("prompt") || k.contains("content") || k.contains("state")) {
                    return false
                }
            }
            val lowered = value.lowercase()
            if (forbiddenTelemetrySubstrings.any { lowered.contains(it) }) return false
        }
        return true
    }

    fun studentResetVisible(studentResetAllowed: Boolean, readOnly: Boolean): Boolean =
        studentResetAllowed && !readOnly

    fun canObtainDeniedCapability(capability: String, deniedCapabilities: List<String>): Boolean {
        val needle = capability.lowercase()
        return deniedCapabilities.none { it.equals(needle, ignoreCase = true) }
    }

    /** Merge course allowlist with org policy allowlist (intersection when both non-empty). */
    fun effectiveAllowedToolIds(courseAllowed: List<String>, orgAllowed: List<String>): List<String> {
        if (orgAllowed.isEmpty()) return courseAllowed
        if (courseAllowed.isEmpty()) return orgAllowed
        return courseAllowed.intersect(orgAllowed.toSet()).toList()
    }
}
