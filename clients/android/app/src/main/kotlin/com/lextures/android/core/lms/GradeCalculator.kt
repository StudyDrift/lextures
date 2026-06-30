package com.lextures.android.core.lms

import java.util.Date

/** Full port of `clients/web/src/pages/lms/gradebook/compute-course-final-percent.ts`. */
object GradeCalculator {
    private const val UNGROUPED = "__ungrouped__"

    data class ColumnForFinal(
        val id: String,
        val maxPoints: Double?,
        val assignmentGroupId: String? = null,
        val neverDrop: Boolean = false,
        val replaceWithFinal: Boolean = false,
        val dueAt: String? = null,
    )

    data class GroupWeight(
        val id: String,
        val weightPercent: Double,
        val dropLowest: Int = 0,
        val dropHighest: Int = 0,
        val replaceLowestWithFinal: Boolean = false,
    )

    enum class Mode { Actual, WhatIf }

    data class ComputeOptions(
        val mode: Mode = Mode.Actual,
        val whatIfOverrides: Map<String, String> = emptyMap(),
        val heldItemIds: Set<String> = emptySet(),
        val now: Date = Date(),
    )

    fun mergeGradesForWhatIf(
        actualGrades: Map<String, String>,
        overrides: Map<String, String>,
        heldItemIds: Set<String>,
    ): Map<String, String> {
        val merged = actualGrades.toMutableMap()
        for (id in heldItemIds) merged.remove(id)
        for ((id, value) in overrides) {
            val trimmed = value.trim()
            if (trimmed.isEmpty()) merged.remove(id) else merged[id] = trimmed
        }
        return merged
    }

    fun groupEffectiveEarnedAndMax(
        policy: GroupWeight,
        lines: List<LineInput>,
    ): GroupResult {
        if (lines.isEmpty()) return GroupResult(0.0, 0.0, emptySet())

        data class Scored(
            val id: String,
            val max: Double,
            val earned: Double,
            val pct: Double,
            val canDrop: Boolean,
            val isFinal: Boolean,
        )

        val rows = lines.map { line ->
            val max = if (line.max > 0 && line.max.isFinite()) line.max else 0.0
            val earned = maxOf(0.0, line.earned)
            val pct = if (max > 0) earned / max else 0.0
            val canDrop = !line.neverDrop && !line.isFinal
            Scored(line.itemId, max, earned, if (pct.isFinite()) pct else 0.0, canDrop, line.isFinal)
        }.filter { it.max > 0 }

        val sorted = rows.sortedWith(compareBy({ it.pct }, { it.id }))
        val dropped = mutableSetOf<String>()
        val work = sorted.filter { it.canDrop }.toMutableList()

        repeat(maxOf(0, policy.dropLowest)) {
            if (work.isEmpty()) return@repeat
            dropped += work.removeAt(0).id
        }
        repeat(maxOf(0, policy.dropHighest)) {
            if (work.isEmpty()) return@repeat
            dropped += work.removeAt(work.lastIndex).id
        }

        var effectiveMax = 0.0
        var effectiveEarned = 0.0
        for (row in sorted) {
            if (row.id in dropped) continue
            effectiveMax += row.max
            effectiveEarned += row.earned
        }

        if (policy.replaceLowestWithFinal) {
            val finalRow = sorted.firstOrNull { it.isFinal && it.id !in dropped && it.pct > 0 }
            val others = sorted.filter { !it.isFinal && it.id !in dropped }
            if (finalRow != null && others.isNotEmpty()) {
                val lowest = others.minWith(compareBy({ it.pct }, { it.id }))
                if (finalRow.pct > lowest.pct + 1e-12) {
                    effectiveEarned -= lowest.earned
                    effectiveEarned += lowest.max * finalRow.pct
                }
            }
        }

        return GroupResult(effectiveEarned, effectiveMax, dropped)
    }

    fun computeCourseFinalPercent(
        columns: List<ColumnForFinal>,
        gradesByItemId: Map<String, String>,
        assignmentGroups: List<GroupWeight>,
        excusedByItemId: Map<String, Boolean> = emptyMap(),
        options: ComputeOptions = ComputeOptions(),
    ): Double? {
        val mergedGrades = if (options.mode == Mode.WhatIf) {
            mergeGradesForWhatIf(gradesByItemId, options.whatIfOverrides, options.heldItemIds)
        } else {
            gradesByItemId
        }

        val settingsIds = assignmentGroups.map { it.id }.toSet()
        val polByG = assignmentGroups.associateBy { it.id }
        val maxByBucket = mutableMapOf<String, Double>()
        val earnedByBucket = mutableMapOf<String, Double>()
        val byGroup = mutableMapOf<String, MutableList<LineInput>>()
        val nowMs = options.now.time

        for (col in columns) {
            val max = col.maxPoints ?: continue
            if (max <= 0) continue
            if (excusedByItemId[col.id] == true) continue

            val hasOverride = options.mode == Mode.WhatIf &&
                (options.whatIfOverrides[col.id]?.trim()?.isNotEmpty() == true)
            val gradeStr = mergedGrades[col.id]
            if (!shouldIncludeColumn(col, gradeStr, hasOverride, options.mode, nowMs)) continue

            val earned = parseEarned(gradeStr)
            val gid = col.assignmentGroupId?.trim()?.takeIf { it.isNotEmpty() }
            val bucket = if (gid != null && gid in settingsIds) gid else UNGROUPED

            if (bucket == UNGROUPED) {
                maxByBucket[bucket] = (maxByBucket[bucket] ?: 0.0) + max
                earnedByBucket[bucket] = (earnedByBucket[bucket] ?: 0.0) + earned
            } else {
                byGroup.getOrPut(bucket) { mutableListOf() }.add(
                    LineInput(col.id, max, earned, col.neverDrop, col.replaceWithFinal),
                )
            }
        }

        for ((gid, lines) in byGroup) {
            val policy = polByG[gid] ?: GroupWeight(gid, 0.0)
            val result = groupEffectiveEarnedAndMax(policy, lines)
            maxByBucket[gid] = (maxByBucket[gid] ?: 0.0) + result.effectiveMax
            earnedByBucket[gid] = (earnedByBucket[gid] ?: 0.0) + result.effectiveEarned
        }

        val totalMaxPoints = maxByBucket.values.sum()
        if (totalMaxPoints <= 0) return null

        val bucketsWithColumns = maxByBucket.filterValues { it > 0 }.keys
        if (bucketsWithColumns.isEmpty()) return null

        val configuredSum = assignmentGroups.sumOf { g ->
            if (g.weightPercent.isFinite() && g.weightPercent > 0) g.weightPercent else 0.0
        }
        val remainder = maxOf(0.0, 100 - configuredSum)

        var lostConfiguredWeight = 0.0
        for (group in assignmentGroups) {
            val w = if (group.weightPercent.isFinite() && group.weightPercent > 0) group.weightPercent else 0.0
            if (w <= 0) continue
            if (group.id !in bucketsWithColumns) lostConfiguredWeight += w
        }

        val maxUngrouped = maxByBucket[UNGROUPED] ?: 0.0
        val rawWeight = mutableMapOf<String, Double>()
        for (group in assignmentGroups) {
            if (group.id !in bucketsWithColumns) continue
            val w = if (group.weightPercent.isFinite() && group.weightPercent > 0) group.weightPercent else 0.0
            if (w > 0) rawWeight[group.id] = w
        }

        if (UNGROUPED in bucketsWithColumns) {
            var wU = remainder + lostConfiguredWeight
            if (wU <= 0 && maxUngrouped > 0 && totalMaxPoints > 0) {
                wU = (maxUngrouped / totalMaxPoints) * 100
            }
            rawWeight[UNGROUPED] = (rawWeight[UNGROUPED] ?: 0.0) + wU
        }

        val weightSum = rawWeight.values.sum()
        if (weightSum <= 0) {
            val earnedTotal = earnedByBucket.values.sum()
            return (earnedTotal / totalMaxPoints) * 100
        }

        var acc = 0.0
        for ((bucket, rw) in rawWeight) {
            if (rw <= 0) continue
            val maxB = maxByBucket[bucket] ?: 0.0
            val earnedB = earnedByBucket[bucket] ?: 0.0
            val ratio = if (maxB > 0) earnedB / maxB else 0.0
            acc += ratio * (rw / weightSum)
        }
        return acc * 100
    }

    fun computeWhatIfFinalPercent(
        columns: List<ColumnForFinal>,
        actualGrades: Map<String, String>,
        assignmentGroups: List<GroupWeight>,
        excusedByItemId: Map<String, Boolean>,
        whatIfOverrides: Map<String, String>,
        heldItemIds: Set<String>,
        now: Date = Date(),
    ): Double? = computeCourseFinalPercent(
        columns,
        actualGrades,
        assignmentGroups,
        excusedByItemId,
        ComputeOptions(Mode.WhatIf, whatIfOverrides, heldItemIds, now),
    )

    fun computeDroppedGrades(
        columns: List<ColumnForFinal>,
        gradesByItemId: Map<String, String>,
        assignmentGroups: List<GroupWeight>,
        excusedByItemId: Map<String, Boolean> = emptyMap(),
        options: ComputeOptions = ComputeOptions(),
    ): Map<String, Boolean> {
        val mergedGrades = if (options.mode == Mode.WhatIf) {
            mergeGradesForWhatIf(gradesByItemId, options.whatIfOverrides, options.heldItemIds)
        } else {
            gradesByItemId
        }

        val settingsIds = assignmentGroups.map { it.id }.toSet()
        val polByG = assignmentGroups.associateBy { it.id }
        val byGroup = mutableMapOf<String, MutableList<LineInput>>()
        val nowMs = options.now.time
        val dropped = mutableMapOf<String, Boolean>()

        for (col in columns) {
            val max = col.maxPoints ?: continue
            if (max <= 0) continue
            if (excusedByItemId[col.id] == true) continue

            val hasOverride = options.mode == Mode.WhatIf &&
                (options.whatIfOverrides[col.id]?.trim()?.isNotEmpty() == true)
            val gradeStr = mergedGrades[col.id]
            if (!shouldIncludeColumn(col, gradeStr, hasOverride, options.mode, nowMs)) continue

            val earned = parseEarned(gradeStr)
            val gid = col.assignmentGroupId?.trim()?.takeIf { it.isNotEmpty() }
            val bucket = if (gid != null && gid in settingsIds) gid else UNGROUPED
            if (bucket == UNGROUPED) continue

            byGroup.getOrPut(bucket) { mutableListOf() }.add(
                LineInput(col.id, max, earned, col.neverDrop, col.replaceWithFinal),
            )
        }

        for ((gid, lines) in byGroup) {
            val policy = polByG[gid] ?: GroupWeight(gid, 0.0)
            val result = groupEffectiveEarnedAndMax(policy, lines)
            for (id in result.droppedIds) dropped[id] = true
        }
        return dropped
    }

    fun formatFinalPercent(pct: Double?): String {
        if (pct == null || !pct.isFinite()) return "—"
        val rounded = kotlin.math.round(pct * 10) / 10
        return "$rounded%"
    }

    fun columnsFrom(response: MyGradesResponse): List<ColumnForFinal> =
        response.columns.map {
            ColumnForFinal(
                id = it.id,
                maxPoints = it.maxPoints,
                assignmentGroupId = it.assignmentGroupId,
                neverDrop = it.neverDrop,
                replaceWithFinal = it.replaceWithFinal,
                dueAt = it.dueAt,
            )
        }

    fun groupsFrom(response: MyGradesResponse): List<GroupWeight> =
        response.assignmentGroups.map {
            GroupWeight(
                id = it.id,
                weightPercent = it.weightPercent,
                dropLowest = it.dropLowest,
                dropHighest = it.dropHighest,
                replaceLowestWithFinal = it.replaceLowestWithFinal,
            )
        }

    fun excusedByItemIdFrom(response: MyGradesResponse): Map<String, Boolean> =
        response.gradeStatuses.filterValues { it == "excused" }.mapValues { true }

    fun heldSetFrom(response: MyGradesResponse): Set<String> = response.heldGradeItemIds.toSet()

    fun calcColumnsFrom(response: MyGradesResponse): List<ColumnForFinal> {
        val held = heldSetFrom(response)
        return columnsFrom(response).filter { it.id !in held }
    }

    fun overallPercent(response: MyGradesResponse, options: ComputeOptions = ComputeOptions()): Double? =
        computeCourseFinalPercent(
            calcColumnsFrom(response),
            response.grades,
            groupsFrom(response),
            excusedByItemIdFrom(response),
            options,
        )

    fun activeDroppedGrades(
        response: MyGradesResponse,
        whatIfMode: Boolean,
        whatIfOverrides: Map<String, String>,
    ): Map<String, Boolean> {
        if (whatIfMode && whatIfOverrides.isNotEmpty()) {
            return computeDroppedGrades(
                calcColumnsFrom(response),
                response.grades,
                groupsFrom(response),
                excusedByItemIdFrom(response),
                ComputeOptions(
                    mode = Mode.WhatIf,
                    whatIfOverrides = whatIfOverrides,
                    heldItemIds = heldSetFrom(response),
                ),
            )
        }
        return response.droppedGrades
    }

    data class LineInput(
        val itemId: String,
        val max: Double,
        val earned: Double,
        val neverDrop: Boolean,
        val isFinal: Boolean,
    )

    data class GroupResult(
        val effectiveEarned: Double,
        val effectiveMax: Double,
        val droppedIds: Set<String>,
    )

    private fun parseEarned(raw: String?): Double {
        val trimmed = raw?.trim().orEmpty()
        if (trimmed.isEmpty()) return 0.0
        return trimmed.replace(",", "").toDoubleOrNull() ?: 0.0
    }

    private fun shouldIncludeColumn(
        col: ColumnForFinal,
        gradeStr: String?,
        hasOverride: Boolean,
        mode: Mode,
        nowMs: Long,
    ): Boolean {
        if (mode == Mode.WhatIf && hasOverride) return true
        val hasGrade = gradeStr?.trim()?.isNotEmpty() == true
        val isPastDue = LmsDates.parse(col.dueAt)?.toEpochMilli()?.let { it < nowMs } == true
        return hasGrade || isPastDue
    }
}
