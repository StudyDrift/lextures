package com.lextures.android.core.lms

import androidx.annotation.StringRes
import com.lextures.android.R
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonArray
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.doubleOrNull
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import java.util.UUID
import kotlin.math.abs

/** Course grading scale, weighted groups, and item mapping helpers (M13.4). */
object CourseGradingLogic {
    enum class GradingScaleOptionId {
        letter_standard,
        letter_plus_minus,
        percent,
        pass_fail,
    }

    enum class SchemeDisplayTypeId {
        points,
        percentage,
        letter,
        gpa,
        pass_fail,
        complete_incomplete,
    }

    data class GradingScaleOption(val id: GradingScaleOptionId)
    data class SchemeDisplayType(val id: SchemeDisplayTypeId)

    data class EditableAssignmentGroup(
        val clientKey: String,
        val id: String? = null,
        val name: String,
        val sortOrder: Int,
        val weightPercent: String,
    )

    data class GradingSchemeBand(
        val clientKey: String,
        val label: String,
        val minPct: String,
        val gpa: String = "",
    )

    data class GradableRow(val item: CourseStructureItem, val moduleTitle: String)

    data class FormBaseline(
        val gradingScale: String,
        val groups: List<EditableAssignmentGroup>,
        val schemeType: String,
        val bands: List<GradingSchemeBand>,
        val passMinPct: String,
        val completeMinPct: String,
    )

    sealed class ValidationError {
        data object GroupsNeedNames : ValidationError()
        data class BandsInvalid(val message: String) : ValidationError()
        data class SchemeInvalid(val message: String) : ValidationError()
    }

    val gradingScaleOptions = listOf(
        GradingScaleOption(GradingScaleOptionId.letter_standard),
        GradingScaleOption(GradingScaleOptionId.letter_plus_minus),
        GradingScaleOption(GradingScaleOptionId.percent),
        GradingScaleOption(GradingScaleOptionId.pass_fail),
    )

    val schemeDisplayTypes = listOf(
        SchemeDisplayType(SchemeDisplayTypeId.points),
        SchemeDisplayType(SchemeDisplayTypeId.percentage),
        SchemeDisplayType(SchemeDisplayTypeId.letter),
        SchemeDisplayType(SchemeDisplayTypeId.gpa),
        SchemeDisplayType(SchemeDisplayTypeId.pass_fail),
        SchemeDisplayType(SchemeDisplayTypeId.complete_incomplete),
    )

    @StringRes
    fun gradingScaleLabelRes(id: GradingScaleOptionId): Int = when (id) {
        GradingScaleOptionId.letter_standard -> R.string.mobile_courseSettings_grading_scale_letterStandard_label
        GradingScaleOptionId.letter_plus_minus -> R.string.mobile_courseSettings_grading_scale_letterPlusMinus_label
        GradingScaleOptionId.percent -> R.string.mobile_courseSettings_grading_scale_percent_label
        GradingScaleOptionId.pass_fail -> R.string.mobile_courseSettings_grading_scale_passFail_label
    }

    @StringRes
    fun gradingScaleDescriptionRes(id: GradingScaleOptionId): Int = when (id) {
        GradingScaleOptionId.letter_standard -> R.string.mobile_courseSettings_grading_scale_letterStandard_description
        GradingScaleOptionId.letter_plus_minus -> R.string.mobile_courseSettings_grading_scale_letterPlusMinus_description
        GradingScaleOptionId.percent -> R.string.mobile_courseSettings_grading_scale_percent_description
        GradingScaleOptionId.pass_fail -> R.string.mobile_courseSettings_grading_scale_passFail_description
    }

    @StringRes
    fun schemeTypeLabelRes(id: SchemeDisplayTypeId): Int = when (id) {
        SchemeDisplayTypeId.points -> R.string.mobile_courseSettings_grading_scheme_type_points
        SchemeDisplayTypeId.percentage -> R.string.mobile_courseSettings_grading_scheme_type_percentage
        SchemeDisplayTypeId.letter -> R.string.mobile_courseSettings_grading_scheme_type_letter
        SchemeDisplayTypeId.gpa -> R.string.mobile_courseSettings_grading_scheme_type_gpa
        SchemeDisplayTypeId.pass_fail -> R.string.mobile_courseSettings_grading_scheme_type_passFail
        SchemeDisplayTypeId.complete_incomplete -> R.string.mobile_courseSettings_grading_scheme_type_completeIncomplete
    }

    @StringRes
    fun kindLabelRes(kind: String): Int = when (kind) {
        "quiz" -> R.string.mobile_courseSettings_grading_mapping_type_quiz
        "content_page" -> R.string.mobile_courseSettings_grading_mapping_type_content
        else -> R.string.mobile_courseSettings_grading_mapping_type_assignment
    }

    fun cacheKeyGrading(courseCode: String): String = "course:$courseCode:grading-settings"
    fun settingsIdempotencyKey(courseCode: String): String = "course-grading:$courseCode:settings"
    fun schemeIdempotencyKey(courseCode: String): String = "course-grading:$courseCode:scheme"
    fun itemMappingIdempotencyKey(courseCode: String, itemId: String): String = "course-grading:$courseCode:item-group:$itemId"

    fun newClientKey(): String = "new-${UUID.randomUUID()}"

    fun defaultGroups(): List<EditableAssignmentGroup> = listOf(
        EditableAssignmentGroup(newClientKey(), null, "Assignments", 0, "100"),
    )

    fun defaultBands(): List<GradingSchemeBand> = listOf(
        GradingSchemeBand(newClientKey(), "A", "90", "4"),
        GradingSchemeBand(newClientKey(), "B", "80", "3"),
        GradingSchemeBand(newClientKey(), "C", "70", "2"),
        GradingSchemeBand(newClientKey(), "D", "60", "1"),
        GradingSchemeBand(newClientKey(), "F", "0", "0"),
    )

    fun groupsFromSettings(settings: CourseGradingSettings): List<EditableAssignmentGroup> {
        if (settings.assignmentGroups.isEmpty()) return defaultGroups()
        return settings.assignmentGroups.map { group ->
            EditableAssignmentGroup(group.id, group.id, group.name, group.sortOrder, group.weightPercent.toString())
        }
    }

    fun baseline(settings: CourseGradingSettings, scheme: CourseGradingSchemeRecord?): FormBaseline {
        val schemeType = scheme?.type?.trim()?.takeIf { it.isNotEmpty() } ?: "points"
        val parsed = parseBands(scheme?.scaleJson)
        return FormBaseline(
            gradingScale = settings.gradingScale.trim().ifEmpty { "letter_standard" },
            groups = groupsFromSettings(settings),
            schemeType = schemeType,
            bands = parsed.ifEmpty { defaultBands() },
            passMinPct = parsePassMinPct(scheme?.scaleJson) ?: "60",
            completeMinPct = parseCompleteMinPct(scheme?.scaleJson) ?: "50",
        )
    }

    fun weightTotal(groups: List<EditableAssignmentGroup>): Double {
        val total = groups.sumOf { group ->
            group.weightPercent.trim().toDoubleOrNull()?.takeIf { it.isFinite() } ?: 0.0
        }
        return kotlin.math.round(total * 1000.0) / 1000.0
    }

    fun hasWeightWarning(total: Double): Boolean = abs(total - 100.0) >= 0.01

    fun gradableRows(structure: List<CourseStructureItem>): List<GradableRow> {
        val rows = mutableListOf<GradableRow>()
        var moduleTitle = ""
        structure.forEach { item ->
            if (item.kind == "module") moduleTitle = item.title
            else if (item.isGradable) rows += GradableRow(item, moduleTitle)
        }
        return rows
    }

    fun namedGroupsWithIds(groups: List<EditableAssignmentGroup>): List<EditableAssignmentGroup> =
        groups.filter { it.name.trim().isNotEmpty() && it.id != null }

    fun isSettingsDirty(current: FormBaseline, baseline: FormBaseline): Boolean {
        if (current.gradingScale != baseline.gradingScale) return true
        return normalizedGroups(current.groups) != normalizedGroups(baseline.groups)
    }

    fun isSchemeDirty(current: FormBaseline, baseline: FormBaseline): Boolean {
        if (current.schemeType != baseline.schemeType) return true
        return when (current.schemeType) {
            "letter", "gpa" -> normalizedBands(current.bands) != normalizedBands(baseline.bands)
            "pass_fail" -> current.passMinPct.trim() != baseline.passMinPct.trim()
            "complete_incomplete" -> current.completeMinPct.trim() != baseline.completeMinPct.trim()
            else -> false
        }
    }

    fun validateGroups(groups: List<EditableAssignmentGroup>): ValidationError? {
        val named = groups.filter { it.name.trim().isNotEmpty() }
        return if (named.isEmpty() || named.size != groups.size) ValidationError.GroupsNeedNames else null
    }

    fun validateBands(bands: List<GradingSchemeBand>): ValidationError? {
        if (bands.isEmpty()) return ValidationError.BandsInvalid("bands-required")
        val parsed = mutableListOf<Pair<String, Double>>()
        bands.forEachIndexed { index, band ->
            val label = band.label.trim()
            if (label.isEmpty()) return ValidationError.BandsInvalid("band-label-$index")
            val minPct = band.minPct.trim().toDoubleOrNull()
            if (minPct == null || !minPct.isFinite() || minPct < 0 || minPct > 100) {
                return ValidationError.BandsInvalid("band-min-$index")
            }
            parsed += label to minPct
        }
        val ascending = parsed.sortedBy { it.second }
        if (abs(ascending.first().second) > 0.001) return ValidationError.BandsInvalid("lowest-zero")
        for (index in 1 until ascending.size) {
            if (ascending[index].second <= ascending[index - 1].second + 0.001) {
                return ValidationError.BandsInvalid("bands-increase")
            }
        }
        return null
    }

    fun validateScheme(form: FormBaseline): ValidationError? = when (form.schemeType) {
        "letter", "gpa" -> validateBands(form.bands)
        "pass_fail" -> {
            val value = form.passMinPct.trim().toDoubleOrNull()
            if (value == null || !value.isFinite() || value < 0 || value > 100) ValidationError.SchemeInvalid("pass-min")
            else null
        }
        "complete_incomplete" -> {
            val value = form.completeMinPct.trim().toDoubleOrNull()
            if (value == null || !value.isFinite() || value < 0 || value > 100) ValidationError.SchemeInvalid("complete-min")
            else null
        }
        else -> null
    }

    fun buildPutSettingsBody(form: FormBaseline): PutCourseGradingSettingsBody =
        PutCourseGradingSettingsBody(
            gradingScale = form.gradingScale,
            assignmentGroups = form.groups.mapIndexed { index, group ->
                val weight = group.weightPercent.trim().toDoubleOrNull()?.takeIf { it.isFinite() } ?: 0.0
                CourseAssignmentGroupInput(
                    id = group.id,
                    name = group.name.trim(),
                    sortOrder = index,
                    weightPercent = weight,
                )
            },
        )

    fun buildPutSchemeBody(form: FormBaseline): PutCourseGradingSchemeBody =
        PutCourseGradingSchemeBody(
            type = form.schemeType,
            scaleJson = when (form.schemeType) {
                "letter", "gpa" -> encodeBands(form.bands)
                "pass_fail" -> buildJsonObject { put("pass_min_pct", JsonPrimitive(form.passMinPct.trim().toDoubleOrNull() ?: 60.0)) }
                "complete_incomplete" -> buildJsonObject { put("complete_min_pct", JsonPrimitive(form.completeMinPct.trim().toDoubleOrNull() ?: 50.0)) }
                else -> buildJsonObject { }
            },
        )

    fun sortBandsDescending(bands: List<GradingSchemeBand>): List<GradingSchemeBand> =
        bands.sortedByDescending { it.minPct.trim().toDoubleOrNull() ?: 0.0 }

    private fun normalizedGroups(groups: List<EditableAssignmentGroup>): List<Pair<String, String>> =
        groups.map { it.name.trim() to (it.weightPercent.trim().toDoubleOrNull() ?: 0.0).toString() }

    private fun normalizedBands(bands: List<GradingSchemeBand>): List<Triple<String, String, String>> =
        sortBandsDescending(bands).map { Triple(it.label.trim(), it.minPct.trim(), it.gpa.trim()) }

    private fun parseBands(scaleJson: JsonElement?): List<GradingSchemeBand> {
        val array = scaleJson as? JsonArray ?: return emptyList()
        return array.mapNotNull { element ->
            val obj = element.jsonObject
            val label = obj["label"]?.jsonPrimitive?.content?.trim().orEmpty()
            val minPct = obj["min_pct"]?.jsonPrimitive?.doubleOrNull?.toString()
                ?: obj["min_pct"]?.jsonPrimitive?.content
            if (label.isEmpty() || minPct.isNullOrBlank()) return@mapNotNull null
            val gpa = obj["gpa"]?.jsonPrimitive?.doubleOrNull?.toString()
                ?: obj["gpa"]?.jsonPrimitive?.content.orEmpty()
            GradingSchemeBand(newClientKey(), label, minPct, gpa)
        }
    }

    private fun parsePassMinPct(scaleJson: JsonElement?): String? =
        (scaleJson as? JsonObject)?.get("pass_min_pct")?.jsonPrimitive?.doubleOrNull?.toString()
            ?: (scaleJson as? JsonObject)?.get("pass_min_pct")?.jsonPrimitive?.content

    private fun parseCompleteMinPct(scaleJson: JsonElement?): String? =
        (scaleJson as? JsonObject)?.get("complete_min_pct")?.jsonPrimitive?.doubleOrNull?.toString()
            ?: (scaleJson as? JsonObject)?.get("complete_min_pct")?.jsonPrimitive?.content

    private fun encodeBands(bands: List<GradingSchemeBand>): JsonArray = buildJsonArray {
        sortBandsDescending(bands).forEach { band ->
            add(
                buildJsonObject {
                    put("label", JsonPrimitive(band.label.trim()))
                    put("min_pct", JsonPrimitive(band.minPct.trim().toDoubleOrNull() ?: 0.0))
                    band.gpa.trim().toDoubleOrNull()?.let { put("gpa", JsonPrimitive(it)) }
                },
            )
        }
    }
}