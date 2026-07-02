package com.lextures.android.core.lms

import kotlinx.serialization.Serializable

@Serializable
data class StudyTimeAllocationRow(
    val moduleId: String,
    val moduleTitle: String,
    val minutes: Double = 0.0,
)

@Serializable
data class StudyStats(
    val optedIn: Boolean = false,
    val loginStreakDays: Int = 0,
    val timeOnTaskSecondsThisWeek: Int = 0,
    val weeklyGoalHours: Float? = null,
    val goalProgressHours: Float = 0f,
    val goalRemainingHours: Float? = null,
    val studyEfficiency: Double? = null,
    val lowStudyEfficiency: Boolean = false,
    val timeAllocation: List<StudyTimeAllocationRow> = emptyList(),
    val weekStart: String = "",
    val weekEnd: String = "",
)

@Serializable
data class StudyGoal(
    val weeklyHours: Float = 0f,
    val optedIn: Boolean = false,
)

@Serializable
data class PutStudyGoalBody(
    val weeklyHours: Float? = null,
    val optedIn: Boolean? = null,
)

@Serializable
data class ReflectionJournalEntry(
    val id: String,
    val courseId: String? = null,
    val entryText: String,
    val createdAt: String,
)

@Serializable
data class ReflectionJournalListResponse(
    val entries: List<ReflectionJournalEntry> = emptyList(),
)

@Serializable
data class PostReflectionJournalBody(
    val entryText: String,
    val courseId: String? = null,
)

@Serializable
data class PostReflectionJournalResponse(
    val id: String,
)

@Serializable
data class CoachingTip(
    val id: String,
    val tipText: String,
    val weekOf: String,
    val rating: Int? = null,
    val createdAt: String,
)

@Serializable
data class CoachingTipsResponse(
    val latest: CoachingTip? = null,
    val history: List<CoachingTip> = emptyList(),
)

@Serializable
data class RateCoachingTipBody(
    val rating: Int,
)

@Serializable
data class ReminderConfig(
    val dailyGoalMinutes: Int = 0,
    val reminderTime: String = "18:00",
    val reminderChannels: List<String> = emptyList(),
    val weeklySummary: Boolean = false,
    val enabled: Boolean = false,
    val pausedUntil: String? = null,
    val minutesStudiedToday: Int = 0,
    val goalMetToday: Boolean = false,
    val streakAtRiskBanner: Boolean = false,
)

@Serializable
data class PatchReminderConfigBody(
    val enabled: Boolean? = null,
    val dailyGoalMinutes: Int? = null,
    val reminderTime: String? = null,
    val weeklySummary: Boolean? = null,
)

data class CourseProgressSummary(
    val courseCode: String,
    val title: String,
    val percentComplete: Int,
)