package com.lextures.android.core.lms

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonElement

@Serializable
data class CourseAssignmentGroup(
    val id: String,
    val sortOrder: Int = 0,
    val name: String,
    val weightPercent: Double,
    val dropLowest: Int? = null,
    val dropHighest: Int? = null,
    val replaceLowestWithFinal: Boolean? = null,
)

@Serializable
data class CourseGradingSettings(
    val gradingScale: String,
    val assignmentGroups: List<CourseAssignmentGroup> = emptyList(),
    val sbgEnabled: Boolean? = null,
    val sbgAggregationRule: String? = null,
)

@Serializable
data class CourseAssignmentGroupInput(
    val id: String? = null,
    val name: String,
    val sortOrder: Int,
    val weightPercent: Double,
    val dropLowest: Int = 0,
    val dropHighest: Int = 0,
    val replaceLowestWithFinal: Boolean = false,
)

@Serializable
data class PutCourseGradingSettingsBody(
    val gradingScale: String,
    val assignmentGroups: List<CourseAssignmentGroupInput>,
)

@Serializable
data class CourseGradingSchemeRecord(
    val id: String? = null,
    val name: String? = null,
    val type: String,
    @SerialName("scaleJson") val scaleJson: JsonElement? = null,
)

@Serializable
data class CourseGradingSchemeEnvelope(
    val scheme: CourseGradingSchemeRecord? = null,
)

@Serializable
data class PutCourseGradingSchemeBody(
    val type: String,
    @SerialName("scaleJson") val scaleJson: JsonElement? = null,
)

@Serializable
data class PatchItemAssignmentGroupBody(
    val assignmentGroupId: String? = null,
)