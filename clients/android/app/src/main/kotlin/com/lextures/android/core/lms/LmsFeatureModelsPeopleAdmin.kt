package com.lextures.android.core.lms

import kotlinx.serialization.Serializable

@Serializable
data class PersonRow(
    val id: String,
    val email: String,
    val firstName: String? = null,
    val lastName: String? = null,
    val displayName: String? = null,
    val orgId: String,
    val orgName: String,
    val role: String,
    val active: Boolean,
    val createdAt: String,
)

@Serializable
data class PaginatedPeople(
    val items: List<PersonRow> = emptyList(),
    val total: Long = 0,
    val page: Int = 1,
    val perPage: Int = 25,
    val totalPages: Int = 0,
)

@Serializable
data class PeopleDashboardStats(
    val signupsLast7Days: Long = 0,
    val activeAccounts: Long = 0,
    val totalAccounts: Long = 0,
    val recentlyActive30Days: Long = 0,
    val suspendedAccounts: Long = 0,
)

enum class PeopleListFilter(val apiValue: String) {
    Signups7d("signups_7d"),
    Active("active"),
    Recent30d("recent_30d"),
    Total("total"),
    Suspended("suspended"),
}

@Serializable
data class PersonEnrollment(
    val courseId: String,
    val courseCode: String,
    val courseTitle: String,
    val role: String,
    val active: Boolean,
    val state: String,
    val enrolledAt: String,
    val orgName: String? = null,
)

@Serializable
data class PersonActivity(
    val eventKind: String,
    val courseCode: String,
    val courseTitle: String,
    val occurredAt: String,
)

@Serializable
data class PersonReport(
    val id: String,
    val email: String,
    val firstName: String? = null,
    val lastName: String? = null,
    val displayName: String? = null,
    val orgId: String,
    val orgName: String,
    val role: String,
    val active: Boolean,
    val createdAt: String,
    val lastActivityAt: String? = null,
    val enrollmentCount: Int = 0,
    val enrollments: List<PersonEnrollment> = emptyList(),
    val recentActivity: List<PersonActivity> = emptyList(),
)

@Serializable
data class InvitePersonRequest(
    val email: String,
    val firstName: String? = null,
    val lastName: String? = null,
)

@Serializable
data class PatchPersonRequest(
    val active: Boolean,
)

@Serializable
data class ForgotPasswordRequest(
    val email: String,
)
