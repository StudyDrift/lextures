package com.lextures.android.features.grades

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import com.lextures.android.core.lms.GradeCalculator
import com.lextures.android.core.lms.GradeColumn
import com.lextures.android.core.lms.MyGradesResponse

/** Client-only what-if projection state layered over real grades (M6.1 / plan 3.16). */
class WhatIfState {
    var mode by mutableStateOf(false)
    var overrides by mutableStateOf<Map<String, String>>(emptyMap())

    val hasOverrides: Boolean
        get() = overrides.values.any { it.trim().isNotEmpty() }

    fun toggleMode() {
        mode = !mode
    }

    fun reset() {
        overrides = emptyMap()
    }

    fun setOverride(itemId: String, value: String) {
        val trimmed = value.trim()
        overrides = if (trimmed.isEmpty()) {
            overrides - itemId
        } else {
            overrides + (itemId to trimmed)
        }
    }

    fun projectedPercent(grades: MyGradesResponse): Double? {
        if (!mode) return null
        return GradeCalculator.computeWhatIfFinalPercent(
            GradeCalculator.calcColumnsFrom(grades),
            grades.grades,
            GradeCalculator.groupsFrom(grades),
            GradeCalculator.excusedByItemIdFrom(grades),
            overrides,
            GradeCalculator.heldSetFrom(grades),
        )
    }

    fun actualPercent(grades: MyGradesResponse): Double? = GradeCalculator.overallPercent(grades)

    fun activeDropped(grades: MyGradesResponse): Map<String, Boolean> =
        GradeCalculator.activeDroppedGrades(grades, mode, overrides)
}

data class GradesSection(
    val id: String,
    val title: String,
    val weightPercent: Double?,
    val columns: List<GradeColumn>,
)

object GradesDisplayLogic {
    fun buildSections(response: MyGradesResponse): List<GradesSection> {
        val groupIds = response.assignmentGroups.map { it.id }.toSet()
        val grouped = mutableMapOf<String, MutableList<GradeColumn>>()
        val ungrouped = mutableListOf<GradeColumn>()

        for (column in response.columns) {
            val gid = column.assignmentGroupId?.trim()?.takeIf { it.isNotEmpty() }
            if (gid != null && gid in groupIds) {
                grouped.getOrPut(gid) { mutableListOf() }.add(column)
            } else {
                ungrouped.add(column)
            }
        }

        val sections = response.assignmentGroups.mapNotNull { group ->
            val cols = grouped[group.id] ?: return@mapNotNull null
            if (cols.isEmpty()) return@mapNotNull null
            GradesSection(
                id = group.id,
                title = group.name.ifBlank { "Assignments" },
                weightPercent = group.weightPercent.takeIf { it > 0 },
                columns = cols,
            )
        }.toMutableList()

        if (ungrouped.isNotEmpty()) {
            sections += GradesSection("__ungrouped__", "Other", null, ungrouped)
        }
        return sections
    }

    fun statusBadges(
        column: GradeColumn,
        response: MyGradesResponse,
        dropped: Map<String, Boolean>,
    ): List<String> = buildList {
        when {
            response.gradeStatuses[column.id] == "excused" -> add("Excused")
            response.heldGradeItemIds.contains(column.id) -> add("Pending")
            dropped[column.id] == true -> add("Dropped")
            response.gradeStatuses[column.id] == "late" -> add("Late")
        }
    }
}
