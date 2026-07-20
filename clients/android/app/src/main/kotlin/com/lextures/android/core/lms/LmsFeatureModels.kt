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

/** Row from GET `/api/v1/me/notification-preferences`. */
@Serializable
data class NotificationPreference(
    val eventType: String,
    val emailEnabled: Boolean = true,
    val pushEnabled: Boolean = true,
    val smsEnabled: Boolean = false,
    val digestMode: String = "instant",
)

@Serializable
data class NotificationPreferencesResponse(
    val preferences: List<NotificationPreference> = emptyList(),
)

@Serializable
data class NotificationPreferencePatch(
    val eventType: String,
    val emailEnabled: Boolean? = null,
    val pushEnabled: Boolean? = null,
    val smsEnabled: Boolean? = null,
    val digestMode: String? = null,
)

@Serializable
data class NotificationPreferencesUpdate(
    val preferences: List<NotificationPreferencePatch>,
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

@Serializable
data class CreateBroadcastRequest(
    val type: String,
    val subject: String,
    val body: String,
)

@Serializable
data class CreateBroadcastResponse(
    val broadcast: Broadcast,
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
    val neverDrop: Boolean = false,
    val replaceWithFinal: Boolean = false,
    val rubric: RubricDefinition? = null,
)

@Serializable
data class AssignmentGroup(
    val id: String,
    val name: String = "",
    val weightPercent: Double = 0.0,
    val dropLowest: Int = 0,
    val dropHighest: Int = 0,
    val replaceLowestWithFinal: Boolean = false,
)

@Serializable
data class RubricLevel(
    val label: String = "",
    val points: Double = 0.0,
    val description: String? = null,
)

@Serializable
data class RubricCriterion(
    val id: String,
    val title: String = "",
    val description: String? = null,
    val levels: List<RubricLevel> = emptyList(),
)

@Serializable
data class RubricDefinition(
    val title: String? = null,
    val criteria: List<RubricCriterion> = emptyList(),
)

@Serializable
data class GradeComment(
    val id: String? = null,
    val displayName: String? = null,
    val body: String = "",
    val createdAt: String? = null,
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
data class SubmitAssignmentTextRequest(
    val text: String,
)

@Serializable
data class SubmitAssignmentResponse(
    val submission: AssignmentSubmission,
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
    val rubricScores: Map<String, Double>? = null,
    val instructorComment: String? = null,
    val comments: List<GradeComment>? = null,
    val posted: Boolean? = null,
    val excused: Boolean? = null,
    val gradedByAi: Boolean? = null,
)

@Serializable
data class SubmissionAnnotation(
    val id: String,
    val submissionId: String? = null,
    val page: Int = 1,
    val toolType: String = "",
    val colour: String = "#facc15",
    val coordsJson: AnnotationCoords? = null,
    val body: String? = null,
    val createdAt: String? = null,
)

@Serializable
data class AnnotationCoords(
    val x1: Double? = null,
    val y1: Double? = null,
    val x2: Double? = null,
    val y2: Double? = null,
    val x: Double? = null,
    val y: Double? = null,
    val points: List<AnnotationPoint>? = null,
    val rects: List<AnnotationRect>? = null,
)

@Serializable
data class AnnotationPoint(val x: Double = 0.0, val y: Double = 0.0)

@Serializable
data class AnnotationRect(val x1: Double = 0.0, val y1: Double = 0.0, val x2: Double = 0.0, val y2: Double = 0.0)

@Serializable
data class SubmissionFeedbackMedia(
    val id: String,
    val mediaType: String = "audio",
    val mimeType: String = "",
    val durationSecs: Double? = null,
    val contentPath: String = "",
    val createdAt: String? = null,
)

@Serializable
data class SubmissionAnnotationsResponse(val annotations: List<SubmissionAnnotation> = emptyList())

@Serializable
data class SubmissionFeedbackMediaResponse(val items: List<SubmissionFeedbackMedia> = emptyList())

@Serializable
data class FeedbackPlaybackInfo(
    val contentPath: String,
    val captionPath: String? = null,
    val expiresAt: String? = null,
)

/** GET `/api/v1/me/permissions`. */
@Serializable
data class MyPermissionsResponse(
    val permissionStrings: List<String> = emptyList(),
)

@Serializable
data class PlatformFeatures(
    val ffWhatifGrades: Boolean? = null,
    val feedbackMediaEnabled: Boolean? = null,
    val ffLibrary: Boolean? = null,
    val ffCourseEvaluations: Boolean? = null,
    val ffMobileCourseEvaluations: Boolean? = null,
    val ffMobileIaRedesign: Boolean? = null,
    val ffMobileVibeActivities: Boolean? = null,
    val ffMobileUniversalSearch: Boolean? = null,
    val ffMobileProfileDepth: Boolean? = null,
    val ffMobileLibraryEreserves: Boolean? = null,
    val ffMobileImmersiveReader: Boolean? = null,
    val ffMobileLiveMeetings: Boolean? = null,
    val readAloudEnabled: Boolean? = null,
    val ffReadAloud: Boolean? = null,
    val videoCaptionsEnabled: Boolean? = null,
    val autoCaptioningEnabled: Boolean? = null,
    val translationMemoryEnabled: Boolean? = null,
    val ffReadingPreferences: Boolean? = null,
    val ffMotionNavigation: Boolean? = null,
    val ffMotionReveal: Boolean? = null,
    val ffMotionLists: Boolean? = null,
    val ffMotionOverlays: Boolean? = null,
    val ffMotionControls: Boolean? = null,
    val oerLibraryEnabled: Boolean? = null,
    val xapiEmissionEnabled: Boolean? = null,
    val customFieldsEnabled: Boolean? = null,
    val ffDemographics: Boolean? = null,
    val ffResearchConsent: Boolean? = null,
    val ffPersistentTutor: Boolean? = null,
    val ffAiStudyBuddy: Boolean? = null,
    val ragNotebookEnabled: Boolean? = null,
    val aiStudyBuddyEnabled: Boolean? = null,
    val aiDisclosureEnabled: Boolean? = null,
    val ffPeerReview: Boolean? = null,
    val ffLearningPaths: Boolean? = null,
    val selfReflectionEnabled: Boolean? = null,
    val ffPublicCatalog: Boolean? = null,
    val ffCourseMarketplace: Boolean? = null,
    val ffSelfPacedMode: Boolean? = null,
    val ffCourseReviews: Boolean? = null,
    val ffCompletionCredentials: Boolean? = null,
    val ffCoCurricularTranscript: Boolean? = null,
    val ffTranscripts: Boolean? = null,
    val ffCeuTracking: Boolean? = null,
    val ffEportfolio: Boolean? = null,
    val ffGamification: Boolean? = null,
    val ffStripeBilling: Boolean? = null,
    val ffPaymentsEnabled: Boolean? = null,
    val ffTaxCollection: Boolean? = null,
    val ffAdvisingIntegration: Boolean? = null,
    val ffMobileAdvising: Boolean? = null,
    val ffParentPortal: Boolean? = null,
    val ffConferenceScheduling: Boolean? = null,
    val ffClassroomSignals: Boolean? = null,
    val ffBroadcasts: Boolean? = null,
    val ffUiMode: Boolean? = null,
    val atRiskAlertsEnabled: Boolean? = null,
    val instructorInsightsEnabled: Boolean? = null,
    val studentProgressEnabled: Boolean? = null,
    val ffMobileInstructorInsights: Boolean? = null,
    val ffMobileCourseSettings: Boolean? = null,
    val ffMobileCreateCourse: Boolean? = null,
    val ffMobileCourseCreateV2: Boolean? = null,
    val ffMobileCanvasImport: Boolean? = null,
    val ffMobileAdminConsole: Boolean? = null,
    val ffMobileEnrollmentAdd: Boolean? = null,
    val ffMobileLiveQuiz: Boolean? = null,
    val ffMobileWhiteboardEdit: Boolean? = null,
    val ffMobileMarketplacePurchase: Boolean? = null,
    val ffMobileBoardsAdvanced: Boolean? = null,
    val ffEnrollmentStateMachine: Boolean? = null,
    val adminConsoleEnabled: Boolean? = null,
    val adminAuditLogEnabled: Boolean? = null,
    val ffConsortiumSharing: Boolean? = null,
    val graderAgentEnabled: Boolean? = null,
    val ffPlagiarismChecks: Boolean? = null,
    val altTextEnforcementEnabled: Boolean? = null,
    val learnerProfileEnabled: Boolean? = null,
    val ffMobileLearnerProfile: Boolean? = null,
    val introCourseEnabled: Boolean? = null,
    val ffApiTokens: Boolean? = null,
    val ffCalendarFeeds: Boolean? = null,
    val ffMobileSettingsIntegrations: Boolean? = null,
    val ffMobileAdminSettings: Boolean? = null,
    val ffFeedback: Boolean? = null,
)

@Serializable
data class LibraryCatalogResult(
    val mmsId: String? = null,
    val title: String,
    val author: String? = null,
    val issn: String? = null,
    val isbn: String? = null,
)

@Serializable
data class LibrarySearchResponse(
    val results: List<LibraryCatalogResult> = emptyList(),
)

@Serializable
data class LibraryResourceMeta(
    val title: String? = null,
    val author: String? = null,
    val issn: String? = null,
    val isbn: String? = null,
    val source: String? = null,
    val almaMmsId: String? = null,
    val legantoListId: String? = null,
    val ezproxyUrl: String? = null,
)

@Serializable
data class LibraryResourcePayload(
    val itemId: String,
    val resourceType: String,
    val metadata: LibraryResourceMeta? = null,
    val ezproxyUrl: String? = null,
    val updatedAt: String? = null,
)

@Serializable
data class OERSearchResult(
    val id: String,
    val title: String,
    val description: String? = null,
    val url: String,
    val previewUrl: String? = null,
    val provider: String,
    val licenseSpdx: String? = null,
    val licenseLabel: String? = null,
    val gradeLevel: String? = null,
    val subject: String? = null,
    val attribution: String? = null,
)

@Serializable
data class OERSearchResponse(
    val results: List<OERSearchResult> = emptyList(),
    val provider: String? = null,
    val fromCache: Boolean? = null,
    val cacheAsOf: String? = null,
    val staleCache: Boolean? = null,
)

@Serializable
data class OERProviderRow(
    val provider: String,
)

@Serializable
data class SearchCourseItem(
    val courseCode: String,
    val title: String,
)

@Serializable
data class SearchPersonItem(
    val userId: String,
    val email: String,
    val displayName: String? = null,
    val role: String,
    val courseCode: String,
    val courseTitle: String,
)

@Serializable
data class SearchIndexResponse(
    val courses: List<SearchCourseItem> = emptyList(),
    val people: List<SearchPersonItem> = emptyList(),
)

@Serializable
data class SearchQueryResultItem(
    val id: String,
    val type: String,
    val title: String,
    val subtitle: String,
    val path: String,
    val score: Double? = null,
)

@Serializable
data class SearchQueryGroup(
    val type: String,
    val label: String,
    val total: Int,
    val items: List<SearchQueryResultItem> = emptyList(),
)

@Serializable
data class SearchQueryResponse(
    val groups: List<SearchQueryGroup> = emptyList(),
    val tookMs: Long = 0,
)

/** Navigation target for grade feedback detail (M6.1). */
data class GradeFeedbackRoute(val column: GradeColumn)

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

@Serializable
data class CreateAttendanceSessionBody(
    val collectionMethod: String,
    val title: String? = null,
    val sessionDate: String? = null,
    val sectionId: String? = null,
)

@Serializable
data class AttendanceRecordUpsert(
    val studentUserId: String,
    val status: String,
    val source: String = "instructor",
)

@Serializable
data class SaveAttendanceRecordsBody(
    val records: List<AttendanceRecordUpsert>,
)

@Serializable
data class CloseAttendanceSessionBody(
    val finalizeMissingAsAbsent: Boolean = true,
)

@Serializable
data class SaveAttendanceRecordsResponse(
    val saved: Int? = null,
    val message: String? = null,
)

@Serializable
data class CourseSection(
    val id: String,
    val sectionCode: String,
    val name: String? = null,
    val status: String? = null,
    val courseId: String? = null,
) {
    val isActive: Boolean get() = (status ?: "active") == "active"

    val displayName: String
        get() = name?.trim()?.takeIf { it.isNotEmpty() } ?: sectionCode

    val displayLabel: String
        get() {
            val trimmed = name?.trim().orEmpty()
            return if (trimmed.isNotEmpty()) "$sectionCode — $trimmed" else sectionCode
        }
}

@Serializable
data class CourseSectionsResponse(
    val sections: List<CourseSection> = emptyList(),
)

// region Course roster (M11.4)

@Serializable
data class CourseEnrollment(
    val id: String,
    val userId: String,
    val displayName: String? = null,
    val avatarUrl: String? = null,
    val role: String,
    val roleDisplay: String? = null,
    val lastCourseAccessAt: String? = null,
    val sectionId: String? = null,
    val sectionCode: String? = null,
    val sectionName: String? = null,
    val state: String? = null,
    val invitationPending: Boolean? = null,
)

@Serializable
data class CourseEnrollmentsResponse(
    val enrollments: List<CourseEnrollment> = emptyList(),
)

@Serializable
data class AddCourseEnrollmentsRequest(
    val emails: String,
    val courseRole: String,
)

@Serializable
data class AddCourseEnrollmentsResponse(
    val added: List<String> = emptyList(),
    val alreadyEnrolled: List<String> = emptyList(),
    val notFound: List<String> = emptyList(),
)

@Serializable
data class PatchEnrollmentStateRequest(
    val state: String,
    val reason: String? = null,
)

@Serializable
data class PatchEnrollmentStateResponse(
    val id: String? = null,
    val state: String? = null,
    val stateChangedAt: String? = null,
    val stateReason: String? = null,
)

@Serializable
data class EnrollmentMessageBody(
    val subject: String,
    val body: String,
)

@Serializable
data class EnrollmentMessageResponse(
    val id: String? = null,
)

// endregion

/** Staff navigation into take-attendance mode (optional existing session). */
data class TakeAttendanceRequest(
    val sessionId: String? = null,
)

// region Office hours (M7.3)

@Serializable
data class AvailabilityWindow(
    val id: String,
    val instructorId: String,
    val courseId: String? = null,
    val dayOfWeek: Int? = null,
    val windowDate: String? = null,
    val startTime: String,
    val endTime: String,
    val slotDurationMinutes: Int = 15,
    val location: String? = null,
    val isVirtual: Boolean = false,
    val status: String = "active",
    val createdAt: String? = null,
)

@Serializable
data class AppointmentSlot(
    val id: String,
    val windowId: String,
    val slotStart: String,
    val slotEnd: String,
    val studentId: String? = null,
    val studentNote: String? = null,
    val meetingId: String? = null,
    val status: String,
    val bookedAt: String? = null,
)

@Serializable
data class OfficeHoursAvailability(
    val windows: List<AvailabilityWindow> = emptyList(),
    val slots: List<AppointmentSlot> = emptyList(),
)

@Serializable
data class OfficeHoursAvailabilityResponse(
    val windows: List<AvailabilityWindow>? = null,
    val slots: List<AppointmentSlot>? = null,
)

@Serializable
data class MyAppointmentsResponse(
    val appointments: List<AppointmentSlot>? = null,
)

@Serializable
data class BookOfficeHoursSlotBody(
    val note: String? = null,
)

@Serializable
data class MeetingJoinResponse(
    val joinUrl: String? = null,
    val hostUrl: String? = null,
    val meetingId: String? = null,
    val status: String? = null,
)

@Serializable
data class VirtualMeeting(
    val id: String,
    val courseId: String,
    val sectionId: String? = null,
    val provider: String,
    val title: String,
    val scheduledStart: String? = null,
    val scheduledEnd: String? = null,
    val joinUrl: String? = null,
    val hostUrl: String? = null,
    val externalMeetingId: String? = null,
    val status: String,
    val createdBy: String,
    val createdAt: String,
)

@Serializable
data class CourseMeetingsResponse(
    val meetings: List<VirtualMeeting>? = null,
)

@Serializable
data class MeetingJoinInfo(
    val joinUrl: String,
    val hostUrl: String? = null,
    val meetingId: String,
    val status: String,
)

@Serializable
data class MeetingAttendanceRecord(
    val id: String,
    val meetingId: String,
    val userId: String,
    val joinedAt: String,
    val leftAt: String? = null,
    val durationSeconds: Int? = null,
)

@Serializable
data class MeetingAttendanceResponse(
    val attendance: List<MeetingAttendanceRecord>? = null,
)

@Serializable
data class PatchMeetingBody(
    val status: String? = null,
)

@Serializable
data class WhiteboardElement(
    val type: String,
    val color: String,
    val width: Double,
    val pts: List<List<Double>>? = null,
    val x: Double? = null,
    val y: Double? = null,
    val w: Double? = null,
    val h: Double? = null,
    val cx: Double? = null,
    val cy: Double? = null,
    val rx: Double? = null,
    val ry: Double? = null,
    val x1: Double? = null,
    val y1: Double? = null,
    val x2: Double? = null,
    val y2: Double? = null,
    val x3: Double? = null,
    val y3: Double? = null,
)

@Serializable
data class CourseWhiteboard(
    val id: String,
    val courseId: String,
    val title: String,
    val canvasData: List<WhiteboardElement>? = null,
    val createdBy: String? = null,
    val createdAt: String,
    val updatedAt: String,
)

@Serializable
data class CourseWhiteboardsResponse(
    val whiteboards: List<CourseWhiteboard>? = null,
)

@Serializable
data class WhiteboardUpsertRequest(
    val title: String,
    val canvasData: List<WhiteboardElement> = emptyList(),
)

// endregion

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

// region Planner (todos + calendar, M2.1)

@Serializable
data class CalendarCourseFeed(
    val courseId: String,
    val courseCode: String,
    val title: String,
    val feedUrl: String,
)

@Serializable
data class CalendarTokenInfo(
    val hasToken: Boolean? = null,
    val personalFeedUrl: String? = null,
    val expiresAt: String? = null,
    val courseFeeds: List<CalendarCourseFeed>? = null,
)

@Serializable
data class CalendarTokenCreated(
    val token: String,
    val feedUrl: String? = null,
    val expiresAt: String? = null,
)

@Serializable
data class AcademicCalendarEvent(
    val id: String,
    val orgId: String,
    val termId: String? = null,
    val eventType: String,
    val eventName: String,
    val startDate: String,
    val endDate: String? = null,
    val allDay: Boolean = false,
    val notes: String? = null,
)

@Serializable
data class AcademicCalendarEventsResponse(
    val events: List<AcademicCalendarEvent> = emptyList(),
)

// endregion

// region Course files (M3.2)

@Serializable
data class CourseFileFolder(
    val id: String,
    val courseId: String,
    val parentId: String? = null,
    val name: String,
    val createdAt: String = "",
    val updatedAt: String = "",
)

@Serializable
data class CourseFileItem(
    val id: String,
    val courseId: String,
    val folderId: String? = null,
    val storageKey: String = "",
    val originalFilename: String = "",
    val displayName: String = "",
    val mimeType: String = "",
    val byteSize: Long = 0,
    val createdAt: String = "",
    val updatedAt: String = "",
) {
    val title: String
        get() = displayName.trim().ifEmpty { originalFilename }
}

@Serializable
data class CourseFileBreadcrumb(
    val id: String,
    val name: String,
)

@Serializable
data class CourseFileFolderContents(
    val folderId: String? = null,
    val breadcrumbs: List<CourseFileBreadcrumb>? = null,
    val folders: List<CourseFileFolder> = emptyList(),
    val files: List<CourseFileItem> = emptyList(),
)

enum class CourseFileContentSource {
    FileManager,
    CourseFile,
    /** Absolute API path from submission `attachmentContentPath`. */
    DirectPath,
}

data class FilePreviewTarget(
    val courseCode: String,
    val displayName: String,
    val mimeType: String?,
    val byteSize: Long?,
    val source: CourseFileContentSource,
    val sourceId: String,
) {
    val id: String
        get() = when (source) {
            CourseFileContentSource.FileManager -> "fm:$sourceId"
            CourseFileContentSource.CourseFile -> "cf:$sourceId"
            CourseFileContentSource.DirectPath -> "dp:${sourceId.hashCode()}"
        }

    companion object {
        fun from(file: CourseFileItem, courseCode: String) = FilePreviewTarget(
            courseCode = courseCode,
            displayName = file.title,
            mimeType = file.mimeType,
            byteSize = file.byteSize,
            source = CourseFileContentSource.FileManager,
            sourceId = file.id,
        )

        fun from(moduleItem: CourseStructureItem, courseCode: String) = FilePreviewTarget(
            courseCode = courseCode,
            displayName = moduleItem.title,
            mimeType = CourseFileLogic.guessMimeType(moduleItem.title),
            byteSize = null,
            source = CourseFileContentSource.FileManager,
            sourceId = moduleItem.id,
        )

        fun submissionAttachment(
            courseCode: String,
            fileId: String,
            fileName: String,
            mimeType: String?,
        ) = FilePreviewTarget(
            courseCode = courseCode,
            displayName = fileName,
            mimeType = mimeType,
            byteSize = null,
            source = CourseFileContentSource.CourseFile,
            sourceId = fileId,
        )

        fun submissionContentPath(
            courseCode: String,
            contentPath: String,
            fileName: String,
            mimeType: String?,
        ) = FilePreviewTarget(
            courseCode = courseCode,
            displayName = fileName,
            mimeType = mimeType,
            byteSize = null,
            source = CourseFileContentSource.DirectPath,
            sourceId = contentPath,
        )

        fun portfolioArtifact(
            portfolioId: String,
            artifactId: String,
            fileName: String,
            mimeType: String?,
        ) = FilePreviewTarget(
            courseCode = "portfolio",
            displayName = fileName,
            mimeType = mimeType,
            byteSize = null,
            source = CourseFileContentSource.DirectPath,
            sourceId = "/api/v1/me/portfolios/${CourseFileLogic.encodePath(portfolioId)}" +
                "/artifacts/${CourseFileLogic.encodePath(artifactId)}/content",
        )
    }
}

enum class FilePreviewKind {
    Image,
    Pdf,
    Audio,
    Video,
    DownloadOnly,
}

// endregion

// region Module progress & conditional release (M3.1)

@Serializable
data class LockReason(
    val code: String = "",
    val message: String = "",
    val itemId: String? = null,
    val title: String? = null,
)

@Serializable
data class ItemLockState(
    val itemId: String,
    val locked: Boolean = false,
    val complete: Boolean = false,
    val reason: LockReason? = null,
)

@Serializable
data class ModuleLockState(
    val moduleId: String,
    val title: String = "",
    val sortOrder: Int = 0,
    val locked: Boolean = false,
    val complete: Boolean = false,
    val reason: LockReason? = null,
    val items: List<ItemLockState>? = null,
)

@Serializable
data class ModulesProgressSnapshot(
    val enrollmentId: String = "",
    val modules: List<ModuleLockState> = emptyList(),
)

@Serializable
data class MarkItemCompleteResponse(
    val enrollmentId: String? = null,
    val justComplete: Boolean? = null,
)

// endregion

// region Interactive content (M3.3)

@Serializable
data class ModuleH5PPayload(
    val packageId: String = "",
    val itemId: String? = null,
    val title: String = "",
    val contentType: String? = null,
    val extractStatus: String = "",
    val assetsBaseUrl: String? = null,
    val downloadUrl: String? = null,
)

@Serializable
data class ModuleScormSco(
    val id: String = "",
    val identifier: String? = null,
    val title: String? = null,
    val launchHref: String? = null,
)

@Serializable
data class ModuleScormPayload(
    val packageId: String = "",
    val itemId: String? = null,
    val title: String = "",
    val packageType: String? = null,
    val extractStatus: String = "",
    val assetsBaseUrl: String? = null,
    val downloadUrl: String? = null,
    val scos: List<ModuleScormSco> = emptyList(),
)

@Serializable
data class ScormLaunchResponse(
    val registrationId: String = "",
    val launchUrl: String? = null,
    val renderUrl: String = "",
    val initialCmi: Map<String, String> = emptyMap(),
)

@Serializable
data class ModuleLtiLinkPayload(
    val itemId: String = "",
    val title: String = "",
    val externalToolId: String? = null,
    val externalToolName: String? = null,
    val resourceLinkId: String? = null,
    val lineItemUrl: String? = null,
)

@Serializable
data class LtiEmbedTicketResponse(
    val ticket: String = "",
)

@Serializable
data class ModuleVibeActivityPayload(
    val id: String = "",
    val title: String = "",
    val html: String? = null,
    val published: Boolean? = null,
    val archived: Boolean? = null,
)

@Serializable
data class XapiStatementBody(
    val courseCode: String,
    val packageId: String,
    val statement: kotlinx.serialization.json.JsonElement,
)

// region Discussions (M7.1)

@Serializable
data class DiscussionForum(
    val id: String = "",
    val name: String = "",
    val description: String? = null,
    val position: Int = 0,
    val createdAt: String = "",
)

@Serializable
data class DiscussionThreadSummary(
    val id: String = "",
    val forumId: String = "",
    val authorId: String = "",
    val title: String = "",
    val isPinned: Boolean = false,
    val isLocked: Boolean = false,
    val requirePostFirst: Boolean = false,
    val assignmentStructureItemId: String? = null,
    val createdAt: String = "",
    val updatedAt: String = "",
    val replyCount: Int = 0,
)

@Serializable
data class DiscussionThreadDetail(
    val id: String = "",
    val forumId: String = "",
    val authorId: String = "",
    val title: String = "",
    val isPinned: Boolean = false,
    val isLocked: Boolean = false,
    val requirePostFirst: Boolean = false,
    val assignmentStructureItemId: String? = null,
    val createdAt: String = "",
    val updatedAt: String = "",
    val replyCount: Int = 0,
    val body: kotlinx.serialization.json.JsonElement = DiscussionLogic.encodeBody(""),
) {
    val bodyPlainText: String get() = DiscussionLogic.plainText(body)
}

@Serializable
data class DiscussionPost(
    val id: String = "",
    val threadId: String = "",
    val parentPostId: String? = null,
    val authorId: String = "",
    val body: kotlinx.serialization.json.JsonElement = DiscussionLogic.encodeBody(""),
    val upvoteCount: Int = 0,
    val viewerUpvoted: Boolean = false,
    val createdAt: String = "",
    val updatedAt: String = "",
) {
    val bodyPlainText: String get() = DiscussionLogic.plainText(body)
}

@Serializable
data class DiscussionForumsResponse(val forums: List<DiscussionForum>? = null)

@Serializable
data class DiscussionThreadsResponse(val threads: List<DiscussionThreadSummary>? = null)

@Serializable
data class DiscussionPostsResponse(
    val posts: List<DiscussionPost>? = null,
    val hiddenUntilFirstPost: Boolean = false,
)

@Serializable
data class CreateDiscussionThreadBody(
    val title: String,
    val body: kotlinx.serialization.json.JsonElement,
    val assignmentStructureItemId: String? = null,
    val requirePostFirst: Boolean? = null,
)

@Serializable
data class CreateDiscussionPostBody(
    val parentPostId: String? = null,
    val body: kotlinx.serialization.json.JsonElement,
    val idempotencyKey: String? = null,
)

@Serializable
data class DiscussionUpvoteResponse(
    val wasAdded: Boolean = false,
    val upvoteCount: Int = 0,
)

// endregion

// region AI tutor (M7.2)

@Serializable
data class TutorCitation(
    val sourceId: String,
    val chunkId: String,
    val excerpt: String,
    val title: String? = null,
)

@Serializable
data class TutorMessage(
    val role: String,
    val content: String,
    val citations: List<TutorCitation>? = null,
    val id: String? = null,
)

@Serializable
data class TutorConversationResponse(
    val conversationId: String,
    val messages: List<TutorMessage> = emptyList(),
    val tokensUsed: Int = 0,
    val tokenLimit: Int = 0,
    val periodMonth: String = "",
)

@Serializable
data class TutorSessionSummary(
    val id: String,
    val title: String? = null,
    val createdAt: String,
    val lastActive: String,
)

@Serializable
data class TutorSessionDetailResponse(
    val id: String,
    val title: String? = null,
    val createdAt: String,
    val lastActive: String,
    val messages: List<TutorMessage> = emptyList(),
)

@Serializable
data class TutorTokenBudgetResponse(
    val tokensUsed: Int = 0,
    val tokenLimit: Int = 0,
    val periodMonth: String = "",
)

@Serializable
data class TutorMessageBody(val message: String)

@Serializable
data class TutorSessionMessageBody(val content: String)

@Serializable
data class StudyBuddyMessageBody(
    val message: String,
    val sessionId: String = "",
)

@Serializable
data class NotebookRagNotebookInput(
    val courseCode: String,
    val courseTitle: String,
    val markdown: String,
)

@Serializable
data class NotebookRagQueryBody(
    val question: String,
    val notebooks: List<NotebookRagNotebookInput>,
)

@Serializable
data class NotebookRagSource(
    val courseCode: String,
    val courseTitle: String,
    val excerpt: String,
)

@Serializable
data class NotebookRagQueryResponse(
    val answerMarkdown: String,
    val sources: List<NotebookRagSource>? = null,
)

// endregion

// region Instructor insights (M11.3)

@Serializable
data class AtRiskAlert(
    val id: String,
    val enrollmentId: String,
    val userId: String,
    val displayName: String,
    val score: Float,
    val status: String,
    val topFactor: String,
    val topFactorLabel: String,
    val snoozeUntil: String? = null,
    val notes: String? = null,
    val triggeredDate: String,
    val missingPct: Float? = null,
    val quizAvg: Float? = null,
    val daysInactive: Int? = null,
)

@Serializable
data class AtRiskListResponse(
    val alerts: List<AtRiskAlert> = emptyList(),
    val resolved: List<AtRiskAlert>? = null,
)

@Serializable
data class InstructorSignalItem(
    val itemId: String,
    val title: String,
    val kind: String,
    val completionRate: Double,
    val avgScore: Double? = null,
    val engagement: Double,
    val difficulty: Double? = null,
    val compositeScore: Double,
    val narrative: String,
)

@Serializable
data class InstructorScatterPoint(
    val itemId: String,
    val title: String,
    val kind: String,
    val difficulty: Double,
    val engagement: Double,
    val flag: String? = null,
)

@Serializable
data class InstructorInsightsResponse(
    val courseId: String,
    val weekOf: String,
    val generatedAt: String,
    val workingWell: List<InstructorSignalItem> = emptyList(),
    val needsAttention: List<InstructorSignalItem> = emptyList(),
    val scatter: List<InstructorScatterPoint>? = null,
)

@Serializable
data class StudentProgressSummary(
    val enrollmentId: String,
    val courseId: String,
    val studentUserId: String,
    val studentDisplayName: String,
    val studentAvatarUrl: String? = null,
    val assignmentsSubmittedPct: Double,
    val modulesViewedPct: Double,
    val avgQuizScore: Double? = null,
    val avgGradePercent: Double? = null,
    val lastActiveAt: String? = null,
    val missingCount: Int,
    val dataAsOf: String,
    val staleMinutes: Int,
    val canManageNotes: Boolean,
)

@Serializable
data class StudentProgressMissingItem(
    val itemId: String,
    val title: String,
    val kind: String,
    val dueAt: String? = null,
    val daysOverdue: Int,
    val gradeStatus: String,
)

@Serializable
data class StudentProgressAssignmentRow(
    val itemId: String,
    val title: String,
    val dueAt: String? = null,
    val submittedAt: String? = null,
    val grade: String,
    val status: String,
)

@Serializable
data class StudentProgressQuizRow(
    val attemptId: String,
    val itemId: String,
    val title: String,
    val submittedAt: String,
    val scorePercent: Double? = null,
)

@Serializable
data class StudentProgressResponse(
    val summary: StudentProgressSummary,
    val missing: List<StudentProgressMissingItem> = emptyList(),
    val assignments: List<StudentProgressAssignmentRow> = emptyList(),
    val quizzes: List<StudentProgressQuizRow> = emptyList(),
)

@Serializable
data class StudentProgressActivityEvent(
    val occurredAt: String,
    val kind: String,
    val label: String,
    val detail: String? = null,
)

@Serializable
data class StudentProgressActivityResponse(
    val events: List<StudentProgressActivityEvent> = emptyList(),
    val nextCursor: String? = null,
)

data class CourseHealthSnapshot(
    val atRiskCount: Int,
    val ungradedCount: Int,
    val engagementHighlightCount: Int,
)

// endregion
