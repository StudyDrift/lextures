package com.lextures.android.core.lms

import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.booleanOrNull
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.doubleOrNull
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject

/**
 * Pure CT.M6 pack-2 decisions — draft lifecycle, AI error classification,
 * consent gating, discussion control visibility, pagination cursoring,
 * offline action rules, and client allowlist. No networking.
 */
object ContentToolPack2Logic {
    val pack2ToolIds: Set<String> = setOf(
        "ask_questions",
        "explain_it_back",
        "inline_discussion",
    )

    /** Per-tool client allowlist (rollout). Empty entry removes a renderer without a release. */
    var clientAllowlist: Set<String> = pack2ToolIds

    const val DEFAULT_PAGE_SIZE = 20

    enum class AIErrorClass(val code: String, val messageKey: String) {
        RATE_LIMITED("rate_limited", "mobile.contentTools.ai.error.rateLimited"),
        BUDGET("budget", "mobile.contentTools.ai.error.budget"),
        PROVIDER_UNAVAILABLE("provider_unavailable", "mobile.contentTools.ai.error.providerUnavailable"),
        FILTERED("filtered", "mobile.contentTools.ai.error.filtered"),
        OPT_OUT("opt_out", "mobile.contentTools.ai.error.optOut"),
        COPPA("coppa", "mobile.contentTools.ai.error.coppa"),
        TOO_SHORT("too_short", "mobile.contentTools.tools.explain_it_back.error.tooShort"),
        TOO_LONG("too_long", "mobile.contentTools.tools.explain_it_back.error.tooLong"),
        MAX_ATTEMPTS("max_attempts", "mobile.contentTools.tools.explain_it_back.error.maxAttempts"),
        FORBIDDEN("forbidden", "mobile.contentTools.tools.inline_discussion.error.forbidden"),
        OFFLINE("offline", "mobile.contentTools.runtime.offlineComposer"),
        UNKNOWN("unknown", "mobile.contentTools.runtime.retry"),
    }

    enum class DraftEvent {
        SAVE,
        RESTORE,
        CLEAR_ON_SUCCESS,
        RETAIN_ON_FAILURE,
    }

    data class DiscussionControls(
        val canEdit: Boolean,
        val canDelete: Boolean,
        val canEndorse: Boolean,
        val canModerate: Boolean,
        val canUpvote: Boolean,
        val canReport: Boolean,
        val canReply: Boolean,
    )

    // MARK: - Allowlist / registry

    fun isClientAllowlisted(
        toolId: String,
        allowlist: Set<String> = clientAllowlist,
    ): Boolean = toolId in allowlist

    fun allowlistedToolIds(allowlist: Set<String> = clientAllowlist): Set<String> =
        pack2ToolIds.intersect(allowlist)

    fun conflictPolicy(toolId: String): ContentToolHostLogic.ConflictPolicy =
        when (toolId) {
            "ask_questions", "explain_it_back" -> ContentToolHostLogic.ConflictPolicy.MERGE
            else -> ContentToolHostLogic.ConflictPolicy.SERVER_WINS
        }

    /** Pack-2 actions are never queued offline (CT.M3 FR-11 / CT.M6 FR-5). */
    @Suppress("UNUSED_PARAMETER")
    fun canQueueActionOffline(toolId: String, action: String): Boolean = false

    // MARK: - Draft lifecycle

    fun draftStorageKey(instanceId: String, slot: String = "composer"): String =
        "content-tool-draft:$instanceId:$slot"

    fun draftEventAfterAction(success: Boolean, preserveInput: Boolean): DraftEvent {
        if (success) return DraftEvent.CLEAR_ON_SUCCESS
        @Suppress("UNUSED_EXPRESSION")
        preserveInput
        return DraftEvent.RETAIN_ON_FAILURE
    }

    fun shouldClearDraft(success: Boolean): Boolean = success

    @Suppress("UNUSED_PARAMETER")
    fun shouldRetainDraftOnFailure(preserveInput: Boolean): Boolean = true

    // MARK: - AI consent gating

    /** Fail-closed: unknown/unfetched consent blocks AI composers (CT.M6 / CT.M9). */
    fun composerAIAllowed(
        disclosureMode: String?,
        decision: String?,
        consentFetched: Boolean,
    ): Boolean {
        if (!consentFetched) return false
        val mode = (disclosureMode ?: "acknowledge").lowercase()
        if (mode == "none") return true
        val d = (decision ?: "").lowercase()
        if (d == "opted_out") return false
        if (mode == "banner") return d != "opted_out"
        return d == "acknowledged"
    }

    fun shouldShowAIDisclosure(
        disclosureMode: String?,
        decision: String?,
        consentFetched: Boolean,
    ): Boolean {
        if (!consentFetched) return true
        val mode = (disclosureMode ?: "acknowledge").lowercase()
        if (mode == "none") return false
        val d = (decision ?: "").lowercase()
        return d != "acknowledged" && d != "opted_out"
    }

    // MARK: - Error classification

    fun classifyAIError(code: String?): AIErrorClass =
        when ((code ?: "").lowercase()) {
            "rate_limited" -> AIErrorClass.RATE_LIMITED
            "budget" -> AIErrorClass.BUDGET
            "provider_unavailable" -> AIErrorClass.PROVIDER_UNAVAILABLE
            "filtered" -> AIErrorClass.FILTERED
            "opt_out" -> AIErrorClass.OPT_OUT
            "coppa" -> AIErrorClass.COPPA
            "too_short", "length" -> AIErrorClass.TOO_SHORT
            "too_long" -> AIErrorClass.TOO_LONG
            "max_attempts" -> AIErrorClass.MAX_ATTEMPTS
            "forbidden" -> AIErrorClass.FORBIDDEN
            "offline" -> AIErrorClass.OFFLINE
            else -> AIErrorClass.UNKNOWN
        }

    fun plainLanguageMessageKey(code: String?): String = classifyAIError(code).messageKey

    // MARK: - explain_it_back length guidance

    fun wordCount(text: String): Int {
        val trimmed = text.trim()
        if (trimmed.isEmpty()) return 0
        return trimmed.split(Regex("[^\\p{L}\\p{N}]+")).count { it.isNotEmpty() }
    }

    fun lengthGuidanceOK(text: String, minWords: Int, maxWords: Int): Boolean {
        val count = wordCount(text)
        return count in minWords..maxWords
    }

    fun canSubmitExplanation(
        text: String,
        minWords: Int,
        maxWords: Int,
        attemptsUsed: Int,
        maxAttempts: Int,
        readOnly: Boolean,
        online: Boolean,
        consentAllowed: Boolean,
    ): Boolean {
        if (readOnly || !online || !consentAllowed) return false
        if (attemptsUsed >= maxAttempts) return false
        return lengthGuidanceOK(text, minWords, maxWords)
    }

    // MARK: - ask_questions

    fun canAsk(
        text: String,
        readOnly: Boolean,
        online: Boolean,
        consentAllowed: Boolean,
        busy: Boolean,
    ): Boolean {
        if (readOnly || !online || !consentAllowed || busy) return false
        return text.trim().isNotEmpty()
    }

    // MARK: - Discussion controls / pagination

    fun discussionControls(
        isOwn: Boolean,
        canEditFlag: Boolean,
        canDeleteFlag: Boolean,
        allowReplies: Boolean,
        viewerCanEndorse: Boolean,
        viewerCanModerate: Boolean,
        readOnly: Boolean,
        removed: Boolean,
    ): DiscussionControls {
        if (removed) {
            return DiscussionControls(
                canEdit = false,
                canDelete = false,
                canEndorse = viewerCanModerate && !readOnly,
                canModerate = viewerCanModerate && !readOnly,
                canUpvote = false,
                canReport = false,
                canReply = false,
            )
        }
        return DiscussionControls(
            canEdit = isOwn && canEditFlag && !readOnly,
            canDelete = isOwn && canDeleteFlag && !readOnly,
            canEndorse = viewerCanEndorse && !readOnly,
            canModerate = viewerCanModerate && !readOnly,
            canUpvote = !readOnly,
            canReport = !isOwn && !readOnly,
            canReply = allowReplies && !readOnly,
        )
    }

    fun nextPage(currentPage: Int, pageSize: Int, total: Int?): Int? {
        val page = maxOf(1, currentPage)
        val size = maxOf(1, pageSize)
        if (total == null || total <= page * size) return null
        return page + 1
    }

    fun shouldRenderTombstone(removed: Boolean, tombstone: Boolean, moderationState: String?): Boolean {
        if (removed || tombstone) return true
        if (moderationState == null) return false
        return when (moderationState.lowercase()) {
            "hidden", "removed", "flagged" -> true
            else -> true // unknown moderation state → generic tombstone
        }
    }

    fun authorDisplay(serverAuthorDisplay: String?, anonymity: String, isOwn: Boolean): String? {
        // Never invent identity the server withheld (AC-11).
        if (anonymity == "anonymous_to_peers" && !isOwn) {
            return serverAuthorDisplay
        }
        return serverAuthorDisplay
    }

    fun composerSendEnabled(
        text: String,
        readOnly: Boolean,
        online: Boolean,
        busy: Boolean,
        consentAllowed: Boolean = true,
    ): Boolean {
        if (readOnly || !online || busy || !consentAllowed) return false
        return text.trim().isNotEmpty()
    }

    fun mergePreservingUnknown(
        base: Map<String, JsonElement>,
        patch: Map<String, JsonElement>,
    ): Map<String, JsonElement> = base + patch

    fun objectMap(value: JsonElement?): Map<String, JsonElement> =
        value?.asObjectMap() ?: emptyMap()

    fun arrayField(value: JsonElement?, key: String): List<JsonElement> {
        val arr = value?.asObjectMap()?.get(key) ?: return emptyList()
        return runCatching { arr.jsonArray.toList() }.getOrDefault(emptyList())
    }

    fun boolField(value: JsonElement?, key: String): Boolean? {
        val prim = value?.asObjectMap()?.get(key) as? JsonPrimitive ?: return null
        return prim.booleanOrNull
    }

    fun numberField(value: JsonElement?, key: String): Double? {
        val prim = value?.asObjectMap()?.get(key) as? JsonPrimitive ?: return null
        return prim.doubleOrNull ?: prim.contentOrNull?.toDoubleOrNull()
    }

    fun stringField(value: JsonElement?, key: String): String? =
        ContentToolHostLogic.stringField(value, key)

    private fun JsonElement.asObjectMap(): Map<String, JsonElement> =
        runCatching { jsonObject.toMap() }.getOrDefault(emptyMap())
}
