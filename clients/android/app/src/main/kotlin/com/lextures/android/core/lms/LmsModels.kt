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
    val startsAt: String? = null,
    val endsAt: String? = null,
    val published: Boolean? = null,
    val catalogNickname: String? = null,
    val notebookEnabled: Boolean? = null,
    val viewerEnrollmentRoles: List<String>? = null,
) {
    val displayTitle: String
        get() = catalogNickname?.trim()?.takeIf { it.isNotEmpty() } ?: title

    val viewerIsStudent: Boolean
        get() = viewerEnrollmentRoles?.any { it.equals("student", ignoreCase = true) } == true
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
) {
    val isModule: Boolean get() = kind == "module"

    val isGradable: Boolean
        get() = kind == "assignment" || kind == "quiz" || kind == "content_page"
}

@Serializable
data class CourseStructureResponse(
    val items: List<CourseStructureItem> = emptyList(),
)

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
