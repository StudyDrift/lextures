package com.lextures.android.core.lms

import kotlinx.serialization.Serializable

// Shapes documented in `docs/MOBILE_PLAN.md` §2.

// region Profile

/** GET `/api/v1/me`. */
@Serializable
data class MeProfile(
    val id: String,
    val email: String,
    val displayName: String? = null,
) {
    /** First name for greetings; falls back to the email local part. */
    val firstName: String
        get() {
            val name = displayName?.trim().orEmpty()
            if (name.isNotEmpty()) return name.split(" ").first()
            return email.substringBefore("@").ifEmpty { "there" }
        }

    /** Two-letter initials for the avatar chip. */
    val initials: String
        get() {
            val name = displayName?.trim().orEmpty()
            val source = name.ifEmpty { email.substringBefore("@") }
            val parts = source.split(" ").filter { it.isNotEmpty() }
            return if (parts.size >= 2) {
                "${parts.first().first()}${parts.last().first()}".uppercase()
            } else {
                source.take(2).uppercase()
            }
        }
}

// endregion

// region Account settings & accommodations

/** GET/PATCH `/api/v1/settings/account` — the server-backed, editable profile. */
@Serializable
data class AccountProfile(
    val email: String = "",
    val displayName: String? = null,
    val firstName: String? = null,
    val lastName: String? = null,
    val avatarUrl: String? = null,
    val phoneNumber: String? = null,
)

/** Body for PATCH `/api/v1/settings/account`. Only editable fields are sent. */
@Serializable
data class AccountProfilePatch(
    val firstName: String? = null,
    val lastName: String? = null,
    val avatarUrl: String? = null,
    val phoneNumber: String? = null,
)

/** First/last name for forms — falls back to splitting `displayName` (parity with web). */
fun nameFieldsFromProfile(profile: AccountProfile): Pair<String, String> {
    val first = profile.firstName?.trim().orEmpty()
    val last = profile.lastName?.trim().orEmpty()
    if (first.isNotEmpty() || last.isNotEmpty()) return first to last
    val display = profile.displayName?.trim().orEmpty()
    if (display.isEmpty()) return "" to ""
    val parts = display.split(Regex("\\s+")).filter { it.isNotEmpty() }
    if (parts.isEmpty()) return "" to ""
    if (parts.size == 1) return parts[0] to ""
    return parts[0] to parts.drop(1).joinToString(" ")
}

fun AccountProfile.resolvedDisplayName(): String {
    val (first, last) = nameFieldsFromProfile(this)
    val combined = listOf(first, last).filter { it.isNotEmpty() }.joinToString(" ")
    if (combined.isNotEmpty()) return combined
    return displayName?.trim()?.takeIf { it.isNotEmpty() } ?: email
}

fun AccountProfile.resolvedInitials(): String {
    val parts = resolvedDisplayName().split(Regex("\\s+")).filter { it.isNotEmpty() }
    return when {
        parts.size >= 2 -> "${parts.first().first()}${parts.last().first()}".uppercase()
        parts.size == 1 -> parts[0].take(1).uppercase()
        else -> email.take(1).uppercase()
    }
}

/** GET `/api/v1/me/accommodations` — the student's currently active supports. */
@Serializable
data class MyAccommodationsResponse(
    val accommodations: List<MyAccommodation> = emptyList(),
)

@Serializable
data class MyAccommodation(
    val courseCode: String? = null,
    val hasExtendedTime: Boolean = false,
    val hasExtraAttempts: Boolean = false,
    val hintsAlwaysAvailable: Boolean = false,
    val reducedDistractionRecommended: Boolean = false,
    val speechToTextEnabled: Boolean = false,
    val ttsEnabled: Boolean = false,
    val dyslexiaDisplayEnabled: Boolean = false,
    val highContrastEnabled: Boolean = false,
    val reducedMotionEnabled: Boolean = false,
    val separateSetting: Boolean = false,
    val effectiveFrom: String? = null,
    val effectiveUntil: String? = null,
) {
    /** True when this entry carries no active supports (defensive — server filters these out). */
    val isEmpty: Boolean
        get() = !hasExtendedTime && !hasExtraAttempts && !hintsAlwaysAvailable &&
            !reducedDistractionRecommended && !speechToTextEnabled && !ttsEnabled &&
            !dyslexiaDisplayEnabled && !highContrastEnabled && !reducedMotionEnabled &&
            !separateSetting
}

// endregion

// region Notifications

/** Row from GET `/api/v1/me/notifications`. */
@Serializable
data class AppNotification(
    val id: String,
    val eventType: String = "",
    val title: String = "",
    val body: String = "",
    val actionUrl: String? = null,
    val isRead: Boolean = false,
    val createdAt: String = "",
)

@Serializable
data class NotificationsPage(
    val notifications: List<AppNotification> = emptyList(),
    val unreadCount: Int = 0,
)

@Serializable
data class DeviceTokenRegistration(
    val token: String,
    val platform: String,
    val appBundleId: String? = null,
    val appVersion: String? = null,
)

@Serializable
data class DeviceTokenResponse(
    val id: String,
)

// endregion

// region Announcements (org broadcasts)

/** Row from GET `/api/v1/me/broadcasts`. */
@Serializable
data class Broadcast(
    val id: String,
    val type: String = "announcement", // "announcement" | "emergency"
    val subject: String = "",
    val body: String = "",
    val sentAt: String? = null,
    val createdAt: String = "",
) {
    val isEmergency: Boolean get() = type == "emergency"
}

@Serializable
data class BroadcastsResponse(
    val broadcasts: List<Broadcast> = emptyList(),
)

// endregion

// region My grades

/** Column from `/my-grades` (subset used on mobile). */
@Serializable
data class GradeColumn(
    val id: String,
    val kind: String = "",
    val title: String = "",
    val maxPoints: Double? = null,
    val dueAt: String? = null,
    val assignmentGroupId: String? = null,
)

@Serializable
data class AssignmentGroup(
    val id: String,
    val name: String = "",
    val weightPercent: Double = 0.0,
)

/** GET `/courses/{code}/my-grades` (student only). */
@Serializable
data class MyGradesResponse(
    val columns: List<GradeColumn> = emptyList(),
    val grades: Map<String, String> = emptyMap(),
    val displayGrades: Map<String, String> = emptyMap(),
    val assignmentGroups: List<AssignmentGroup> = emptyList(),
    val heldGradeItemIds: List<String> = emptyList(),
    val droppedGrades: Map<String, Boolean> = emptyMap(),
    val gradeStatuses: Map<String, String> = emptyMap(),
)

/**
 * Weighted-total math for `/my-grades` (simplified port of the web logic:
 * per-group earned/possible, weights renormalized over groups that have grades).
 */
object GradeMath {
    fun overallPercent(response: MyGradesResponse): Double? {
        var flatEarned = 0.0
        var flatPossible = 0.0
        val groupEarned = mutableMapOf<String, Double>()
        val groupPossible = mutableMapOf<String, Double>()

        for (column in response.columns) {
            val max = column.maxPoints ?: continue
            if (max <= 0) continue
            if (response.droppedGrades[column.id] == true) continue
            if (response.gradeStatuses[column.id] == "excused") continue
            val earned = response.grades[column.id]?.toDoubleOrNull() ?: continue

            flatEarned += earned
            flatPossible += max
            val key = column.assignmentGroupId ?: ""
            groupEarned[key] = (groupEarned[key] ?: 0.0) + earned
            groupPossible[key] = (groupPossible[key] ?: 0.0) + max
        }

        if (flatPossible <= 0) return null

        val weighted = response.assignmentGroups.filter { it.weightPercent > 0 }
        if (weighted.isEmpty()) return flatEarned / flatPossible * 100

        var weightTotal = 0.0
        var weightedSum = 0.0
        for (group in weighted) {
            val possible = groupPossible[group.id] ?: continue
            if (possible <= 0) continue
            weightTotal += group.weightPercent
            weightedSum += (groupEarned[group.id] ?: 0.0) / possible * group.weightPercent
        }
        if (weightTotal <= 0) return flatEarned / flatPossible * 100
        return weightedSum / weightTotal * 100
    }
}

// endregion

// region Syllabus

@Serializable
data class SyllabusSection(
    val id: String,
    val heading: String = "",
    val markdown: String = "",
)

/** GET `/courses/{code}/syllabus`. */
@Serializable
data class SyllabusPayload(
    val sections: List<SyllabusSection> = emptyList(),
    val updatedAt: String? = null,
    val requireSyllabusAcceptance: Boolean? = null,
    val syllabusAcceptancePending: Boolean? = null,
) {
    val hasContent: Boolean get() = sections.any { it.markdown.isNotBlank() }
}

// endregion

// region Assignment submissions

/** Row from `/assignments/{item}/submissions` and `/submissions/mine`. */
@Serializable
data class AssignmentSubmission(
    // Defaulted so roster placeholder rows (enrolled students with no submission yet,
    // which the list endpoint returns without an `id`) decode instead of failing the
    // whole list; callers drop the blank-id rows since they aren't gradeable.
    val id: String = "",
    val submittedBy: String? = null,
    val submittedByDisplayName: String? = null,
    val blindLabel: String? = null,
    val attachmentFilename: String? = null,
    val attachmentMimeType: String? = null,
    val attachmentContentPath: String? = null,
    val bodyText: String? = null,
    val submittedAt: String = "",
    val updatedAt: String? = null,
    val versionNumber: Int? = null,
    val resubmissionRequested: Boolean? = null,
    val revisionDueAt: String? = null,
    val revisionFeedback: String? = null,
    val isGraded: Boolean? = null,
) {
    /** Name shown in staff lists; respects blind grading. */
    val displayName: String
        get() = blindLabel?.takeIf { it.isNotEmpty() }
            ?: submittedByDisplayName?.takeIf { it.isNotEmpty() }
            ?: "Student"
}

@Serializable
data class MySubmissionResponse(
    val submission: AssignmentSubmission? = null,
)

@Serializable
data class SubmissionsListResponse(
    val submissions: List<AssignmentSubmission> = emptyList(),
)

/** GET/PUT `.../submissions/{id}/grade`. */
@Serializable
data class SubmissionGrade(
    val submissionId: String? = null,
    val pointsEarned: Double? = null,
    val maxPoints: Double? = null,
    val instructorComment: String? = null,
    val posted: Boolean? = null,
    val excused: Boolean? = null,
)

@Serializable
data class SubmissionGradePut(
    val pointsEarned: Double? = null,
    val instructorComment: String? = null,
    val clearGrade: Boolean? = null,
)

// endregion

// region Quiz attempts (staff)

/** Row from GET `/quizzes/{item}/attempts`. */
@Serializable
data class QuizAttemptSummary(
    val id: String,
    val studentUserId: String? = null,
    val attemptNumber: Int = 1,
    val submittedAt: String = "",
    val scorePercent: Double? = null,
    val pointsEarned: Double = 0.0,
    val pointsPossible: Double = 0.0,
    val studentName: String? = null,
    val needsManualGrading: Boolean? = null,
)

@Serializable
data class QuizAttemptsListResponse(
    val attempts: List<QuizAttemptSummary> = emptyList(),
)

// endregion

// region Grading backlog (staff)

/** Row from GET `/courses/{code}/grading-backlog`. */
@Serializable
data class GradingBacklogItem(
    val itemId: String? = null,
    val itemType: String? = null, // "assignment" | "quiz"
    val assignmentId: String,
    val assignmentTitle: String = "",
    val ungradedCount: Int = 0,
) {
    val resolvedItemId: String get() = itemId ?: assignmentId
    val isQuiz: Boolean get() = itemType == "quiz"
}

object GradingSubmissionMapper {
    fun quizAttemptsToSubmissions(attempts: List<QuizAttemptSummary>): List<AssignmentSubmission> {
        val byStudent = linkedMapOf<String, QuizAttemptSummary>()
        for (attempt in attempts) {
            val key = attempt.studentUserId?.trim()?.takeIf { it.isNotEmpty() } ?: attempt.id
            val existing = byStudent[key]
            if (existing == null || attempt.attemptNumber >= existing.attemptNumber) {
                byStudent[key] = attempt
            }
        }
        return byStudent.values
            .sortedBy { it.studentName.orEmpty() }
            .map { attempt ->
                AssignmentSubmission(
                    id = attempt.id,
                    submittedBy = attempt.studentUserId,
                    submittedByDisplayName = attempt.studentName,
                    submittedAt = attempt.submittedAt,
                    versionNumber = attempt.attemptNumber.takeIf { it > 1 },
                    isGraded = attempt.needsManualGrading == false,
                )
            }
    }

    fun filterSubmissions(submissions: List<AssignmentSubmission>, graded: String?): List<AssignmentSubmission> {
        if (graded.isNullOrEmpty() || graded == "all") return submissions
        return if (graded == "graded") {
            submissions.filter { it.isGraded == true }
        } else {
            submissions.filter { it.isGraded != true }
        }
    }
}

@Serializable
data class GradingBacklogResponse(
    val items: List<GradingBacklogItem> = emptyList(),
)

// endregion

// region Attendance

/** Session from GET `/courses/{code}/attendance/sessions`. */
@Serializable
data class AttendanceSession(
    val id: String,
    val title: String? = null,
    val collectionMethod: String = "roll_call", // "roll_call" | "self_report"
    val sessionDate: String? = null,
    val status: String = "open", // "open" | "closed"
) {
    val isOpen: Boolean get() = status == "open"
    val isSelfReport: Boolean get() = collectionMethod == "self_report"
    val displayTitle: String get() = title?.takeIf { it.isNotEmpty() } ?: "Attendance"
}

@Serializable
data class AttendanceRecord(
    val studentUserId: String = "",
    val displayName: String? = null,
    val status: String = "not_recorded",
    val recordedAt: String? = null,
)

/** GET `.../attendance/sessions/{id}` — session plus viewer-specific fields. */
@Serializable
data class AttendanceSessionDetail(
    val id: String,
    val title: String? = null,
    val collectionMethod: String = "roll_call",
    val sessionDate: String? = null,
    val status: String = "open",
    val records: List<AttendanceRecord>? = null,
    val myRecord: AttendanceRecord? = null,
    val canSelfReport: Boolean? = null,
)

@Serializable
data class AttendanceSessionsResponse(
    val sessions: List<AttendanceSession> = emptyList(),
)

@Serializable
data class SelfReportBody(val status: String)

object AttendanceStatusInfo {
    fun label(status: String): String = when (status) {
        "present" -> "Present"
        "absent" -> "Absent"
        "tardy" -> "Tardy"
        "excused" -> "Excused"
        "not_recorded" -> "Not recorded"
        else -> status.replace('_', ' ').replaceFirstChar { it.uppercase() }
    }
}

// endregion
