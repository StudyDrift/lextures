package com.lextures.android.core.lms

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import java.time.format.FormatStyle

/** Subset of web `CoursePublic` (camelCase JSON) used by the mobile app. */
@Serializable
data class CourseSummary(
    val id: String,
    val courseCode: String,
    val title: String,
    val description: String = "",
    val heroImageUrl: String? = null,
    val heroImageObjectPosition: String? = null,
    val startsAt: String? = null,
    val endsAt: String? = null,
    val visibleFrom: String? = null,
    val hiddenAt: String? = null,
    val scheduleMode: String? = null,
    val relativeEndAfter: String? = null,
    val relativeHiddenAfter: String? = null,
    val courseHomeLanding: String? = null,
    val courseHomeContentItemId: String? = null,
    val courseTimezone: String? = null,
    val gradeLevel: String? = null,
    val markdownThemePreset: String? = null,
    val markdownThemeCustom: MarkdownThemeCustom? = null,
    val published: Boolean? = null,
    val catalogNickname: String? = null,
    val catalogPinned: Boolean = false,
    val acquiredViaMarketplace: Boolean = false,
    val acquisitionSource: String? = null,
    val notebookEnabled: Boolean? = null,
    val calendarEnabled: Boolean? = null,
    val officeHoursEnabled: Boolean? = null,
    val orgId: String? = null,
    val termId: String? = null,
    /** `traditional` or `competency_based` when set by the server. */
    val courseType: String? = null,
    val viewerEnrollmentRoles: List<String>? = null,
    val feedEnabled: Boolean? = null,
    val discussionsEnabled: Boolean? = null,
    val liveSessionsEnabled: Boolean? = null,
    val filesEnabled: Boolean? = null,
    val attendanceEnabled: Boolean? = null,
    val sectionsEnabled: Boolean? = null,
    val resubmissionWorkflowEnabled: Boolean? = null,
    val aiTutorEnabled: Boolean? = null,
    val viewerStudentEnrollmentId: String? = null,
    val standardsAlignmentEnabled: Boolean? = null,
    val reportCardsEnabled: Boolean? = null,
    val groupSpacesEnabled: Boolean? = null,
    val collabDocsEnabled: Boolean? = null,
    val questionBankEnabled: Boolean? = null,
    val lockdownModeEnabled: Boolean? = null,
    val adaptivePathsEnabled: Boolean? = null,
    val srsEnabled: Boolean? = null,
    val diagnosticAssessmentsEnabled: Boolean? = null,
    val hintScaffoldingEnabled: Boolean? = null,
    val misconceptionDetectionEnabled: Boolean? = null,
    val multilingualMessagingEnabled: Boolean? = null,
    val whiteboardEnabled: Boolean? = null,
    /** VC.M1 — visual collaboration boards (opt-in; default off). */
    val visualBoardsEnabled: Boolean? = null,
    /** IQ / MOB.5 — interactive live quizzes (opt-in; default off). */
    val interactiveQuizzesEnabled: Boolean? = null,
    val requireCaptions: Boolean? = null,
    val viewerEnrollmentInvitationPending: Boolean? = null,
    val viewerPendingEnrollmentId: String? = null,
    val isBlueprint: Boolean? = null,
    val blueprintParentId: String? = null,
    val blueprintParentCourseCode: String? = null,
    val blueprintLastSyncAt: String? = null,
) {
    /**
     * True when the viewer has an unaccepted enrollment invitation for this course. The
     * enrolled-courses list surfaces such courses, but course content (structure, grades, …)
     * is gated until the invitation is accepted.
     */
    val hasPendingInvitation: Boolean
        get() = viewerEnrollmentInvitationPending == true && viewerPendingEnrollmentId != null

    val isAiTutorEnabled: Boolean get() = aiTutorEnabled == true
    val isMasteryEnabled: Boolean get() = standardsAlignmentEnabled == true
    val displayTitle: String
        get() = catalogNickname?.trim()?.takeIf { it.isNotEmpty() } ?: title

    val isPinned: Boolean
        get() = catalogPinned

    val viewerIsStudent: Boolean
        get() = viewerEnrollmentRoles?.any { it.equals("student", ignoreCase = true) } == true

    val viewerIsStaff: Boolean
        get() = viewerEnrollmentRoles?.any {
            it.lowercase() in setOf("teacher", "ta", "designer", "grader")
        } == true

    val isFeedEnabled: Boolean get() = feedEnabled != false

    val isDiscussionsEnabled: Boolean get() = discussionsEnabled == true

    val isLiveSessionsEnabled: Boolean get() = liveSessionsEnabled == true

    val isFilesEnabled: Boolean get() = filesEnabled != false

    val isAttendanceEnabled: Boolean get() = attendanceEnabled != false

    val isSectionsEnabled: Boolean get() = sectionsEnabled != false

    val isGroupSpacesEnabled: Boolean get() = groupSpacesEnabled == true

    val isCollabDocsEnabled: Boolean get() = collabDocsEnabled == true

    val isVisualBoardsEnabled: Boolean get() = visualBoardsEnabled == true
    val isInteractiveQuizzesEnabled: Boolean get() = interactiveQuizzesEnabled == true
}

@Serializable
data class CoursesResponse(
    val courses: List<CourseSummary> = emptyList(),
)

/** Mirrors web `CourseStructureItem` (subset). */
@Serializable
data class CourseStructureItem(
    val id: String,
    val sortOrder: Int = 0,
    val kind: String,
    val title: String,
    val parentId: String? = null,
    val published: Boolean = true,
    val dueAt: String? = null,
    val pointsWorth: Double? = null,
    val pointsPossible: Double? = null,
    val archived: Boolean? = null,
    val updatedAt: String? = null,
    val assignmentGroupId: String? = null,
) {
    val isModule: Boolean get() = kind == "module"

    val isGradable: Boolean
        get() = kind == "assignment" || kind == "quiz" || kind == "content_page"
}

@Serializable
data class CourseStructureResponse(
    val items: List<CourseStructureItem> = emptyList(),
)

@Serializable
data class BlueprintChildRow(
    val courseCode: String,
    val title: String,
    val lastSyncAt: String? = null,
)

@Serializable
data class BlueprintPushDetailRow(
    val courseCode: String? = null,
    val ok: Boolean? = null,
    val error: String? = null,
)

@Serializable
data class BlueprintPushResult(
    val childrenTotal: Int,
    val childrenSuccess: Int,
    val childrenError: Int,
    val detail: List<BlueprintPushDetailRow> = emptyList(),
)

@Serializable
data class BlueprintSyncLogRow(
    val id: String,
    val triggeredBy: String,
    val triggeredAt: String,
    val childrenTotal: Int,
    val childrenSuccess: Int,
    val childrenError: Int,
)

@Serializable
data class BlueprintCachedPayload(
    val children: List<BlueprintChildRow> = emptyList(),
    val syncLogs: List<BlueprintSyncLogRow> = emptyList(),
)

/**
 * Tolerant union of the per-kind item GET responses
 * (`/content-pages/{id}`, `/assignments/{id}`, `/quizzes/{id}`, `/external-links/{id}`).
 */
@Serializable
data class ModuleItemDetail(
    val title: String? = null,
    val markdown: String? = null,
    val dueAt: String? = null,
    val availableFrom: String? = null,
    val availableUntil: String? = null,
    val updatedAt: String? = null,
    val pointsWorth: Int? = null,

    // Quiz settings (the web "preview box")
    val unlimitedAttempts: Boolean? = null,
    val maxAttempts: Int? = null,
    val gradeAttemptPolicy: String? = null,
    val oneQuestionAtATime: Boolean? = null,
    val shuffleQuestions: Boolean? = null,
    val lockdownMode: String? = null,
    val adaptiveDeliveryMode: String? = null,
    val timeLimitMinutes: Int? = null,
    val passingScorePercent: Int? = null,
    val requiresQuizAccessCode: Boolean? = null,
    val questions: List<QuestionStub>? = null,

    // Assignment submission settings
    val submissionAllowText: Boolean? = null,
    val submissionAllowFileUpload: Boolean? = null,
    val submissionAllowUrl: Boolean? = null,
    val lateSubmissionPolicy: String? = null,
    val latePenaltyPercent: Int? = null,

    // External link
    val url: String? = null,
    val provider: String? = null,
) {
    @Serializable
    data class QuestionStub(val id: String? = null)

    val questionCount: Int get() = questions?.size ?: 0
}

/** Mirrors web `MailboxMessage` (snake_case JSON from the communication API). */
@Serializable
data class MailboxParty(
    val name: String = "",
    val email: String = "",
)

@Serializable
data class MailboxMessage(
    val id: String,
    val from: MailboxParty = MailboxParty(),
    val to: String = "",
    val subject: String = "",
    val snippet: String = "",
    val body: String = "",
    @SerialName("sent_at") val sentAt: String = "",
    val read: Boolean = false,
    val starred: Boolean = false,
    val folder: String = "inbox",
    @SerialName("has_attachment") val hasAttachment: Boolean = false,
)

@Serializable
data class MailboxMessagesResponse(
    val messages: List<MailboxMessage> = emptyList(),
)

@Serializable
data class UnreadInboxResponse(
    @SerialName("unread_inbox") val unreadInbox: Int? = null,
)

@Serializable
data class MailboxPatchRequest(
    val read: Boolean? = null,
    val starred: Boolean? = null,
    val folder: String? = null,
)

@Serializable
data class SendMessageRequest(
    @SerialName("to_email") val toEmail: String? = null,
    val subject: String,
    val body: String,
    val draft: Boolean? = null,
)

enum class MailboxFolder(val wire: String, val label: String) {
    Inbox("inbox", "Inbox"),
    Starred("starred", "Starred"),
    Sent("sent", "Sent"),
    Drafts("drafts", "Drafts"),
    Trash("trash", "Trash"),
}

object LmsDates {
    fun parse(raw: String?): Instant? {
        if (raw.isNullOrBlank()) return null
        return runCatching { Instant.parse(raw) }.getOrNull()
    }

    fun shortDateTime(raw: String?): String {
        val instant = parse(raw) ?: return ""
        return DateTimeFormatter.ofLocalizedDateTime(FormatStyle.MEDIUM, FormatStyle.SHORT)
            .withZone(ZoneId.systemDefault())
            .format(instant)
    }

    fun shortDate(raw: String?): String {
        val instant = parse(raw) ?: return ""
        return DateTimeFormatter.ofLocalizedDate(FormatStyle.MEDIUM)
            .withZone(ZoneId.systemDefault())
            .format(instant)
    }

    fun relative(raw: String?): String {
        val instant = parse(raw) ?: return ""
        val seconds = java.time.Duration.between(instant, Instant.now()).seconds
        return when {
            seconds < 60 -> "now"
            seconds < 3600 -> "${seconds / 60}m ago"
            seconds < 86_400 -> "${seconds / 3600}h ago"
            seconds < 7 * 86_400 -> "${seconds / 86_400}d ago"
            else -> shortDate(raw)
        }
    }
}
