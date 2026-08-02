package com.lextures.android.core.lms

import com.lextures.android.core.offline.OfflineCacheKey
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.booleanOrNull
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.doubleOrNull
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive

/**
 * Pure CT.M5 pack-1 decisions — attempt/reveal gating, commit immutability,
 * Class Pulse poll backoff, offline action queue ordering, client allowlist,
 * and Review/SRS cache invalidation keys. No networking.
 */
object ContentToolPack1Logic {
    val pack1ToolIds: Set<String> = setOf(
        "inline_questions",
        "predict_reveal",
        "class_pulse",
        "flashcards",
    )

    /** Per-tool client allowlist (rollout). Empty entry removes a renderer without a release. */
    var clientAllowlist: Set<String> = pack1ToolIds

    const val CLASS_PULSE_POLL_INTERVAL_MS = 30_000
    const val CLASS_PULSE_MAX_BACKOFF_MS = 120_000
    val flashcardRatings: List<String> = listOf("again", "hard", "good", "easy")

    data class PendingAction(
        val instanceId: String,
        val toolId: String,
        val action: String,
        val sequence: Long,
        val payloadJSON: String,
    )

    // MARK: - Allowlist / registry

    fun isClientAllowlisted(
        toolId: String,
        allowlist: Set<String> = clientAllowlist,
    ): Boolean = toolId in allowlist

    fun allowlistedToolIds(allowlist: Set<String> = clientAllowlist): Set<String> =
        pack1ToolIds.intersect(allowlist)

    fun conflictPolicy(toolId: String): ContentToolHostLogic.ConflictPolicy =
        if (toolId == "flashcards") {
            ContentToolHostLogic.ConflictPolicy.MERGE
        } else {
            ContentToolHostLogic.ConflictPolicy.SERVER_WINS
        }

    // MARK: - Offline action queue

    fun canQueueActionOffline(toolId: String, action: String): Boolean =
        (toolId == "inline_questions" && action == "submit") ||
            (toolId == "flashcards" && action == "rate")

    fun orderPendingActions(items: List<PendingAction>): List<PendingAction> =
        items.sortedWith(compareBy({ it.instanceId }, { it.sequence }))

    // MARK: - inline_questions

    /** null = unlimited */
    fun parseAttemptsConfig(raw: JsonElement?): Int? {
        if (raw == null) return 2
        val prim = raw as? JsonPrimitive ?: return 2
        val content = prim.contentOrNull ?: return 2
        if (content.equals("unlimited", ignoreCase = true)) return null
        return content.toIntOrNull()?.coerceAtLeast(1)
            ?: prim.doubleOrNull?.toInt()?.coerceAtLeast(1)
            ?: 2
    }

    fun attemptsUsed(answers: Map<String, JsonElement>, questionId: String): Int {
        val q = answers[questionId]?.asObjectMap() ?: return 0
        val attempts = q["attempts"] ?: return 0
        return runCatching { attempts.jsonArray.size }.getOrDefault(0)
    }

    fun canSubmit(
        answers: Map<String, JsonElement>,
        questionId: String,
        maxAttempts: Int?,
        readOnly: Boolean,
    ): Boolean {
        if (readOnly) return false
        if (maxAttempts == null) return true
        return attemptsUsed(answers, questionId) < maxAttempts
    }

    fun isSequentiallyUnlocked(
        questions: List<String>,
        answers: Map<String, JsonElement>,
        questionId: String,
        sequential: Boolean,
    ): Boolean {
        if (!sequential) return true
        for (qid in questions) {
            if (qid == questionId) return true
            if (attemptsUsed(answers, qid) == 0) return false
        }
        return false
    }

    /** How many questions to show at once. null means all. */
    fun parseQuestionsAtATime(raw: JsonElement?): Int? {
        if (raw == null) return null
        val prim = raw as? JsonPrimitive ?: return null
        val content = prim.contentOrNull ?: return null
        if (content.equals("all", ignoreCase = true)) return null
        val n = content.toIntOrNull() ?: prim.doubleOrNull?.toInt() ?: return null
        return if (n in 1..3) n else null
    }

    /** Inclusive-exclusive window [start, end) of the visible page. */
    fun pageWindow(total: Int, pageSize: Int?, pageIndex: Int): IntRange {
        if (pageSize == null || pageSize <= 0 || pageSize >= total || total <= 0) {
            return 0 until maxOf(0, total)
        }
        val pageCount = (total + pageSize - 1) / pageSize
        val page = pageIndex.coerceIn(0, pageCount - 1)
        val start = page * pageSize
        val end = minOf(total, start + pageSize)
        return start until end
    }

    fun initialPageIndex(total: Int, pageSize: Int?, firstIncompleteIndex: Int): Int {
        if (pageSize == null || pageSize <= 0 || total <= 0) return 0
        val idx = firstIncompleteIndex.coerceIn(0, total - 1)
        return idx / pageSize
    }

    fun <T> shuffleStable(items: List<T>, seed: String): List<T> {
        if (items.size <= 1) return items
        val out = items.toMutableList()
        var h = 0u
        for (u in seed.toByteArray()) {
            h = h * 31u + u.toUByte().toUInt()
        }
        for (i in out.lastIndex downTo 1) {
            h = h * 1_103_515_245u + 12_345u
            val j = (h % (i + 1).toUInt()).toInt()
            val tmp = out[i]
            out[i] = out[j]
            out[j] = tmp
        }
        return out
    }

    // MARK: - predict_reveal

    fun isCommitted(state: JsonElement?): Boolean {
        val committed = ContentToolHostLogic.stringField(state, "committedAt")?.trim().orEmpty()
        return committed.isNotEmpty()
    }

    fun canShowReveal(committed: Boolean, hasRevealPayload: Boolean): Boolean =
        committed && hasRevealPayload

    fun canEditPrediction(committed: Boolean, readOnly: Boolean): Boolean =
        !committed && !readOnly

    // MARK: - class_pulse

    fun hasVoted(votes: List<JsonElement>, round: Int = 1): Boolean =
        votes.any { vote ->
            val obj = vote.asObjectMap()
            val r = when (val raw = obj["round"]) {
                is JsonPrimitive -> raw.contentOrNull?.toIntOrNull() ?: raw.doubleOrNull?.toInt()
                else -> null
            }
            r == round
        }

    fun shouldPollAggregate(visible: Boolean, hasVoted: Boolean): Boolean =
        visible && hasVoted

    fun nextPollDelayMs(
        consecutiveFailures: Int,
        baseMs: Int = CLASS_PULSE_POLL_INTERVAL_MS,
    ): Int {
        if (consecutiveFailures <= 0) return baseMs
        val factor = 1 shl minOf(consecutiveFailures, 3)
        return minOf(CLASS_PULSE_MAX_BACKOFF_MS, baseMs * factor)
    }

    // MARK: - flashcards / Review reconciliation

    fun isValidRating(rating: String): Boolean =
        rating.lowercase() in flashcardRatings

    fun reviewCacheKeysToInvalidate(): List<String> =
        listOf(OfflineCacheKey.reviewQueue(), OfflineCacheKey.reviewStats())

    fun shouldDoubleCountReviewSubmit(toolId: String): Boolean =
        toolId != "flashcards"

    // MARK: - Unknown-field preservation

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

    private fun JsonElement.asObjectMap(): Map<String, JsonElement> =
        runCatching { jsonObject.toMap() }.getOrDefault(emptyMap())
}
