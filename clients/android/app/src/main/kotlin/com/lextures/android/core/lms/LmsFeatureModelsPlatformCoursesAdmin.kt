package com.lextures.android.core.lms

import kotlinx.serialization.Serializable

@Serializable
data class PlatformCourseRow(
    val id: String,
    val courseCode: String,
    val title: String,
    val status: String,
    val orgId: String,
    val orgName: String,
    val instructorName: String? = null,
    val termId: String? = null,
    val termName: String? = null,
    val enrollmentCount: Long = 0,
    val createdAt: String = "",
    val updatedAt: String = "",
)

@Serializable
data class PaginatedPlatformCourses(
    val items: List<PlatformCourseRow> = emptyList(),
    val total: Long = 0,
    val page: Int = 1,
    val perPage: Int = 25,
    val totalPages: Int = 0,
)

@Serializable
data class CoursesDashboardStats(
    val createdLast7Days: Long = 0,
    val activeCourses: Long = 0,
    val draftCourses: Long = 0,
    val totalCourses: Long = 0,
    val archivedCourses: Long = 0,
)

enum class CoursesListFilter(val apiValue: String) {
    Created7d("created_7d"),
    Active("active"),
    Draft("draft"),
    Total("total"),
    Archived("archived"),
}

@Serializable
data class PlatformCourseReport(
    val id: String,
    val courseCode: String,
    val title: String,
    val description: String? = null,
    val status: String = "",
    val orgId: String = "",
    val orgName: String = "",
    val instructorName: String? = null,
    val termId: String? = null,
    val termName: String? = null,
    val enrollmentCount: Long = 0,
    val published: Boolean = false,
    val archived: Boolean = false,
    val createdAt: String = "",
    val updatedAt: String = "",
)
