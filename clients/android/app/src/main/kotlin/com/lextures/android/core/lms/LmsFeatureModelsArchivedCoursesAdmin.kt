package com.lextures.android.core.lms

import kotlinx.serialization.Serializable

@Serializable
data class ArchivedCourseRow(
    val id: String,
    val courseCode: String,
    val title: String,
    val archivedAt: String? = null,
    val archivedByUserId: String? = null,
    val archivedByName: String? = null,
    val archivedByEmail: String? = null,
)

@Serializable
data class ArchivedCoursesListResponse(
    val courses: List<ArchivedCourseRow> = emptyList(),
)