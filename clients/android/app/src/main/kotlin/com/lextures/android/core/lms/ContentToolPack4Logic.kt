package com.lextures.android.core.lms

import kotlin.math.floor
import kotlin.math.max
import kotlin.math.min
import kotlin.math.round
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.booleanOrNull
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.doubleOrNull
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive

/**
 * Pure CT.M8 pack-4 decisions — checkpoint scheduling/seek clamp, resume position,
 * worked-example step machine, slider throttle/settle, and client allowlist. No networking.
 */
object ContentToolPack4Logic {
    /** Native pack-4 tools (registered in CT.M3 registry). */
    val pack4ToolIds: Set<String> = setOf(
        "media_checkpoints",
        "worked_example",
        "parameter_explorer",
    )

    /** Delivered via CT.M4 sandbox only — never native-registered. */
    val sandboxToolIds: Set<String> = setOf(
        "code_sandbox",
    )

    /** Per-tool client allowlist (rollout). Empty entry removes a renderer without a release. */
    var clientAllowlist: Set<String> = pack4ToolIds

    const val CHECKPOINT_TOLERANCE_SEC = 0.25
    const val PROGRESS_THROTTLE_MS: Long = 5_000
    const val RECOMPUTE_THROTTLE_MS: Long = 33 // ≤ 30 Hz
    const val DEFAULT_CHECKPOINT_ATTEMPTS = 2
    const val DEFAULT_ATTEMPTS_PER_STEP = 3
    const val SEGMENT_GRANULARITY_SEC = 5.0

    data class Checkpoint(
        val id: String,
        val atSec: Double,
        val required: Boolean,
        val attempts: Int,
    )

    data class CheckpointAnswer(
        val done: Boolean,
        val attemptCount: Int,
        val lastCorrect: Boolean,
    )

    data class SeekClamp(
        val time: Double,
        val clamped: Boolean,
    )

    enum class StepStatus(val wire: String) {
        CURRENT("current"),
        SOLVED("solved"),
        REVEALED("revealed"),
        LOCKED("locked"),
        SCAFFOLDED("scaffolded"),
        ALL_COMPLETE("all_complete"),
    }

    data class AnswerResultView(
        val correct: Boolean?,
        val feedback: String?,
        val attemptsRemaining: Int?,
        val done: Boolean?,
        val error: String?,
        val message: String?,
        val checkpointId: String?,
    )

    data class CheckStepResultView(
        val result: String?,
        val feedback: String?,
        val attemptsRemaining: Int?,
        val canReveal: Boolean?,
        val nextStep: String?,
        val stepId: String?,
        val error: String?,
        val message: String?,
    )

    data class HintResultView(
        val hint: String?,
        val hintsRemaining: Int?,
        val level: Int?,
        val noMoreHints: Boolean?,
        val stepId: String?,
        val error: String?,
    )

    // MARK: - Allowlist / registry

    fun isClientAllowlisted(
        toolId: String,
        allowlist: Set<String> = clientAllowlist,
    ): Boolean = toolId in allowlist

    fun allowlistedToolIds(allowlist: Set<String> = clientAllowlist): Set<String> =
        pack4ToolIds.intersect(allowlist)

    fun conflictPolicy(toolId: String): ContentToolHostLogic.ConflictPolicy =
        when (toolId) {
            "media_checkpoints", "worked_example", "parameter_explorer", "code_sandbox" ->
                ContentToolHostLogic.ConflictPolicy.SERVER_WINS
            else -> ContentToolHostLogic.ConflictPolicy.SERVER_WINS
        }

    /** Pack-4 actions are never queued offline (CT.M3 FR-11). */
    @Suppress("UNUSED_PARAMETER")
    fun canQueueActionOffline(toolId: String, action: String): Boolean = false

    // MARK: - Media checkpoints

    fun parseCheckpoints(config: JsonElement?): List<Checkpoint> =
        arrayField(config, "checkpoints").mapNotNull { raw ->
            val obj = objectMap(raw)
            val id = (obj["id"] as? JsonPrimitive)?.contentOrNull ?: return@mapNotNull null
            val atSec = numberField(raw, "atSec") ?: 0.0
            val required = boolField(raw, "required") ?: true
            val attemptsRaw = numberField(raw, "attempts")?.toInt() ?: DEFAULT_CHECKPOINT_ATTEMPTS
            val attempts = min(10, max(1, attemptsRaw))
            Checkpoint(id, atSec, required, attempts)
        }.sortedBy { it.atSec }

    fun parseAnswers(state: JsonElement?): Map<String, CheckpointAnswer> {
        val answersObj = objectMap(objectMap(state)["answers"])
        return answersObj.mapValues { (_, raw) ->
            val attempts = arrayField(raw, "attempts")
            val lastCorrect = attempts.lastOrNull()?.let { boolField(it, "correct") } ?: false
            CheckpointAnswer(
                done = boolField(raw, "done") ?: false,
                attemptCount = attempts.size,
                lastCorrect = lastCorrect,
            )
        }
    }

    fun isCheckpointDone(answers: Map<String, CheckpointAnswer>, checkpoint: Checkpoint): Boolean {
        val ans = answers[checkpoint.id] ?: return false
        if (ans.done) return true
        if (ans.lastCorrect) return true
        return ans.attemptCount >= checkpoint.attempts
    }

    fun findDueCheckpoint(
        checkpoints: List<Checkpoint>,
        answers: Map<String, CheckpointAnswer>,
        currentTime: Double,
        alreadyPromptedIds: Set<String>,
    ): Checkpoint? {
        for (cp in checkpoints) {
            if (cp.id in alreadyPromptedIds) continue
            if (isCheckpointDone(answers, cp)) continue
            if (currentTime + CHECKPOINT_TOLERANCE_SEC >= cp.atSec && currentTime + 2 >= cp.atSec) {
                if (currentTime >= cp.atSec - CHECKPOINT_TOLERANCE_SEC) {
                    return cp
                }
            }
        }
        return null
    }

    fun earliestUnansweredRequiredSec(
        checkpoints: List<Checkpoint>,
        answers: Map<String, CheckpointAnswer>,
    ): Double? {
        var earliest: Double? = null
        for (cp in checkpoints) {
            if (!cp.required) continue
            if (isCheckpointDone(answers, cp)) continue
            if (earliest == null || cp.atSec < earliest) earliest = cp.atSec
        }
        return earliest
    }

    fun clampSeekTime(
        preventSkip: Boolean,
        checkpoints: List<Checkpoint>,
        answers: Map<String, CheckpointAnswer>,
        targetSec: Double,
    ): SeekClamp {
        if (!preventSkip) return SeekClamp(targetSec, false)
        val limit = earliestUnansweredRequiredSec(checkpoints, answers) ?: return SeekClamp(targetSec, false)
        if (targetSec > limit + 0.05) return SeekClamp(limit, true)
        return SeekClamp(targetSec, false)
    }

    fun mergeLocalSegments(
        existing: List<List<Double>>,
        start: Double,
        end: Double,
        granularity: Double = SEGMENT_GRANULARITY_SEC,
    ): List<List<Double>> {
        if (end <= start) return existing
        fun floorG(v: Double): Double = floor(max(0.0, v) / granularity) * granularity
        fun ceilG(v: Double): Double {
            val f = floorG(v)
            return if (v > f) f + granularity else f
        }
        var s = floorG(start)
        var e = ceilG(end)
        if (e <= s) e = s + granularity
        val all = (existing + listOf(listOf(s, e))).sortedWith(compareBy({ it[0] }, { it[1] }))
        val merged = mutableListOf<List<Double>>()
        for (seg in all) {
            val last = merged.lastOrNull()
            if (last == null || seg[0] > last[1]) {
                merged += seg
            } else if (seg[1] > last[1]) {
                merged[merged.lastIndex] = listOf(last[0], seg[1])
            }
        }
        return merged
    }

    fun resumePosition(furthestSec: Double?, watchedSegments: List<List<Double>>): Double {
        var best = furthestSec ?: 0.0
        for (seg in watchedSegments) {
            if (seg.size >= 2) best = max(best, seg[1])
        }
        return max(0.0, best)
    }

    fun shouldFireProgressThrottle(
        lastFiredAtMs: Long?,
        nowMs: Long,
        intervalMs: Long = PROGRESS_THROTTLE_MS,
    ): Boolean {
        if (lastFiredAtMs == null) return true
        return nowMs - lastFiredAtMs >= intervalMs
    }

    fun shouldBlockPlayback(
        checkpoint: Checkpoint,
        answers: Map<String, CheckpointAnswer>,
    ): Boolean {
        if (!checkpoint.required) return false
        return !isCheckpointDone(answers, checkpoint)
    }

    /** Direct media / course files are reliable; YouTube/Vimeo embeds are not for checkpoints. */
    fun hasReliableCheckpointTiming(
        source: String?,
        url: String?,
        provider: String?,
    ): Boolean {
        val trimmed = url?.trim().orEmpty()
        if (trimmed.isEmpty()) return false
        val p = provider.orEmpty().lowercase()
        if (p == "youtube" || p == "vimeo") return false
        val lower = trimmed.lowercase()
        if ("youtube.com" in lower || "youtu.be" in lower || "vimeo.com" in lower) return false
        @Suppress("UNUSED_EXPRESSION")
        source
        return true
    }

    fun parseWatchedSegments(state: JsonElement?): List<List<Double>> =
        arrayField(state, "watchedSegments").mapNotNull { raw ->
            val arr = raw as? JsonArray ?: return@mapNotNull null
            if (arr.size < 2) return@mapNotNull null
            val a = (arr[0] as? JsonPrimitive)?.doubleOrNull ?: return@mapNotNull null
            val b = (arr[1] as? JsonPrimitive)?.doubleOrNull ?: return@mapNotNull null
            listOf(a, b)
        }

    fun parseAnswerResult(value: JsonElement?): AnswerResultView =
        AnswerResultView(
            correct = boolField(value, "correct"),
            feedback = stringField(value, "feedback"),
            attemptsRemaining = numberField(value, "attemptsRemaining")?.toInt(),
            done = boolField(value, "done"),
            error = stringField(value, "error"),
            message = stringField(value, "message"),
            checkpointId = stringField(value, "checkpointId"),
        )

    // MARK: - Worked example

    fun parseStepIds(config: JsonElement?): List<String> =
        arrayField(config, "steps").mapNotNull { stringField(it, "id") }

    fun blankedStepIds(config: JsonElement?, state: JsonElement?): List<String> {
        val fromState = arrayField(state, "blankedStepIds").mapNotNull {
            (it as? JsonPrimitive)?.contentOrNull
        }
        if (fromState.isNotEmpty()) return fromState
        return arrayField(config, "steps").mapNotNull { raw ->
            val obj = objectMap(raw)
            if (obj["blank"] == null) return@mapNotNull null
            (obj["id"] as? JsonPrimitive)?.contentOrNull
        }
    }

    fun isStepDone(progress: JsonElement?): Boolean {
        if (boolField(progress, "revealed") == true) return true
        if (stringField(progress, "completedAt") != null) return true
        val attempts = arrayField(progress, "attempts")
        val last = attempts.lastOrNull() ?: return false
        val result = stringField(last, "result")
        return result == "correct" || result == "needs_review"
    }

    fun stepProgressMap(state: JsonElement?): Map<String, JsonElement> =
        objectMap(objectMap(state)["steps"])

    fun resolveCurrentStepId(
        blankedStepIds: List<String>,
        currentStepId: String?,
        progress: Map<String, JsonElement>,
    ): String {
        if (!currentStepId.isNullOrEmpty()) return currentStepId
        for (id in blankedStepIds) {
            if (!isStepDone(progress[id])) return id
        }
        return blankedStepIds.firstOrNull().orEmpty()
    }

    fun stepStatus(
        stepId: String,
        blankedStepIds: List<String>,
        currentStepId: String,
        progress: Map<String, JsonElement>,
        allStepIds: List<String>,
    ): StepStatus {
        val blanked = blankedStepIds.toSet()
        if (stepId !in blanked) return StepStatus.SCAFFOLDED
        val sp = progress[stepId]
        if (boolField(sp, "revealed") == true) return StepStatus.REVEALED
        if (isStepDone(sp)) return StepStatus.SOLVED
        if (stepId == currentStepId) return StepStatus.CURRENT
        @Suppress("UNUSED_EXPRESSION")
        allStepIds
        return StepStatus.LOCKED
    }

    fun canCheckStep(draft: String, readOnly: Boolean, busy: Boolean, stepDone: Boolean): Boolean {
        if (readOnly || busy || stepDone) return false
        return draft.trim().isNotEmpty()
    }

    fun mergeStepDraft(
        state: Map<String, JsonElement>,
        stepId: String,
        draft: String,
    ): Map<String, JsonElement> {
        val steps = objectMap(state["steps"]).toMutableMap()
        val sp = objectMap(steps[stepId]).toMutableMap()
        sp["draft"] = JsonPrimitive(draft)
        steps[stepId] = JsonObject(sp)
        return mergePreservingUnknown(
            state,
            mapOf(
                "v" to (state["v"] ?: JsonPrimitive(1)),
                "steps" to JsonObject(steps),
            ),
        )
    }

    fun parseCheckStepResult(value: JsonElement?): CheckStepResultView =
        CheckStepResultView(
            result = stringField(value, "result"),
            feedback = stringField(value, "feedback"),
            attemptsRemaining = numberField(value, "attemptsRemaining")?.toInt(),
            canReveal = boolField(value, "canReveal"),
            nextStep = stringField(value, "nextStep"),
            stepId = stringField(value, "stepId"),
            error = stringField(value, "error"),
            message = stringField(value, "message"),
        )

    fun parseHintResult(value: JsonElement?): HintResultView =
        HintResultView(
            hint = stringField(value, "hint"),
            hintsRemaining = numberField(value, "hintsRemaining")?.toInt(),
            level = numberField(value, "level")?.toInt(),
            noMoreHints = boolField(value, "noMoreHints"),
            stepId = stringField(value, "stepId"),
            error = stringField(value, "error"),
        )

    // MARK: - Parameter explorer

    fun shouldRecompute(
        lastAtMs: Long?,
        nowMs: Long,
        intervalMs: Long = RECOMPUTE_THROTTLE_MS,
    ): Boolean {
        if (lastAtMs == null) return true
        return nowMs - lastAtMs >= intervalMs
    }

    fun shouldAutosaveOnSettle(dragging: Boolean, dirty: Boolean): Boolean =
        !dragging && dirty

    fun defaultParams(config: JsonElement?): Map<String, JsonElement> {
        val out = mutableMapOf<String, JsonElement>()
        for (raw in arrayField(config, "parameters")) {
            val obj = objectMap(raw)
            val id = (obj["id"] as? JsonPrimitive)?.contentOrNull ?: continue
            val def = obj["default"] ?: continue
            out[id] = def
        }
        return out
    }

    fun clampNumber(value: Double, minV: Double, maxV: Double, step: Double): Double {
        val lo = min(minV, maxV)
        val hi = max(minV, maxV)
        var v = max(lo, min(hi, value))
        if (step > 0) {
            val steps = round((v - lo) / step)
            v = lo + steps * step
            v = max(lo, min(hi, v))
        }
        return v
    }

    fun mergeParamsPreservingUnknown(
        state: Map<String, JsonElement>,
        params: Map<String, JsonElement>,
    ): Map<String, JsonElement> =
        mergePreservingUnknown(
            state,
            mapOf(
                "v" to (state["v"] ?: JsonPrimitive(1)),
                "params" to JsonObject(params),
            ),
        )

    // MARK: - JSON helpers

    fun mergePreservingUnknown(
        base: Map<String, JsonElement>,
        patch: Map<String, JsonElement>,
    ): Map<String, JsonElement> = base + patch

    fun objectMap(value: JsonElement?): Map<String, JsonElement> =
        value?.jsonObject ?: emptyMap()

    fun arrayField(value: JsonElement?, key: String): List<JsonElement> {
        val obj = value?.jsonObject ?: return emptyList()
        return obj[key]?.jsonArray?.toList().orEmpty()
    }

    fun boolField(value: JsonElement?, key: String): Boolean? {
        val obj = value?.jsonObject ?: return null
        return (obj[key] as? JsonPrimitive)?.booleanOrNull
    }

    fun numberField(value: JsonElement?, key: String): Double? {
        val obj = value?.jsonObject ?: return null
        val field = obj[key] ?: return null
        val prim = field as? JsonPrimitive ?: return null
        prim.doubleOrNull?.let { return it }
        return prim.contentOrNull?.toDoubleOrNull()
    }

    fun stringField(value: JsonElement?, key: String): String? =
        ContentToolHostLogic.stringField(value, key)
}
