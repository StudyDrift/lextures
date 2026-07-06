package com.lextures.android.core.lms

import kotlinx.serialization.Serializable

@Serializable
data class ParentChildSummary(
    val linkId: String = "",
    val studentUserId: String = "",
    val displayName: String? = null,
    val email: String = "",
    val relationship: String = "",
    val status: String = "",
    val linkedAt: String = "",
)

@Serializable
data class ParentChildrenResponse(
    val children: List<ParentChildSummary> = emptyList(),
)

@Serializable
data class ParentCourseGradesRow(
    val courseCode: String = "",
    val title: String = "",
    val grades: Map<String, String> = emptyMap(),
)

@Serializable
data class ParentGradesResponse(
    val courses: List<ParentCourseGradesRow> = emptyList(),
)

@Serializable
data class ParentAssignmentRow(
    val courseCode: String = "",
    val courseTitle: String = "",
    val itemId: String = "",
    val kind: String = "",
    val title: String = "",
    val dueAt: String? = null,
)

@Serializable
data class ParentAssignmentsResponse(
    val assignments: List<ParentAssignmentRow> = emptyList(),
)

@Serializable
data class ParentAttendanceRecord(
    val id: String = "",
    val studentId: String? = null,
    val sectionId: String? = null,
    val date: String = "",
    val codeId: String? = null,
    val code: String? = null,
    val codeLabel: String? = null,
    val category: String? = null,
    val recordedAt: String? = null,
    val updatedAt: String? = null,
    val schoolId: String? = null,
    val period: String? = null,
    val note: String? = null,
)

@Serializable
data class ParentAttendanceResponse(
    val records: List<ParentAttendanceRecord> = emptyList(),
)

@Serializable
data class ParentBehaviorAward(
    val id: String = "",
    val studentId: String? = null,
    val categoryName: String? = null,
    val points: Int? = null,
    val awardedAt: String? = null,
)

@Serializable
data class ParentBehaviorReferral(
    val id: String = "",
    val studentId: String? = null,
    val categoryName: String? = null,
    val incidentAt: String? = null,
    val createdAt: String? = null,
)

@Serializable
data class ParentBehaviorResponse(
    val studentId: String? = null,
    val totalPoints: Int? = null,
    val awards: List<ParentBehaviorAward> = emptyList(),
    val referrals: List<ParentBehaviorReferral> = emptyList(),
)

@Serializable
data class ParentWeeklySummaryItem(
    val childName: String = "",
    val courseCode: String = "",
    val courseTitle: String = "",
    val itemId: String = "",
    val kind: String = "",
    val title: String = "",
    val dueAt: String? = null,
)

@Serializable
data class ParentWeeklySummaryResponse(
    val items: List<ParentWeeklySummaryItem> = emptyList(),
    val weekStart: String? = null,
    val weekEnd: String? = null,
)

@Serializable
data class ParentNotificationPrefs(
    val gradePosted: Boolean = true,
    val missingAssignment: Boolean = true,
    val lowGradeThreshold: Int? = null,
    val attendanceEvent: Boolean = false,
)

@Serializable
data class PatchParentNotificationPrefsBody(
    val gradePosted: Boolean? = null,
    val missingAssignment: Boolean? = null,
    val lowGradeThreshold: Int? = null,
    val clearThreshold: Boolean? = null,
    val attendanceEvent: Boolean? = null,
)

@Serializable
data class ConferenceTeacher(
    val teacherId: String = "",
    val displayName: String? = null,
)

@Serializable
data class ConferenceTeachersResponse(
    val teachers: List<ConferenceTeacher> = emptyList(),
)

@Serializable
data class ConferenceAvailability(
    val id: String = "",
    val teacherId: String? = null,
    val schoolId: String? = null,
    val date: String? = null,
    val slotDuration: Int? = null,
    val gapDuration: Int? = null,
    val windowStart: String? = null,
    val windowEnd: String? = null,
    val location: String? = null,
    val videoLink: String? = null,
    val createdAt: String? = null,
)

@Serializable
data class ConferenceSlot(
    val id: String = "",
    val availabilityId: String = "",
    val startAt: String = "",
    val endAt: String = "",
    val status: String = "",
    val bookedByParent: String? = null,
    val bookedForChild: String? = null,
    val bookedAt: String? = null,
)

@Serializable
data class ConferenceSlotsResponse(
    val availability: ConferenceAvailability? = null,
    val slots: List<ConferenceSlot> = emptyList(),
)

@Serializable
data class ConferenceSlotResponse(
    val slot: ConferenceSlot? = null,
)

@Serializable
data class BookConferenceSlotBody(
    val studentId: String,
)
