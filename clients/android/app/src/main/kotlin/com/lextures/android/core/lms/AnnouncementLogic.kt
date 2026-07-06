package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures

enum class AnnouncementAudience {
    WholeCourse,
    Section,
}

enum class BroadcastComposeType(val wire: String) {
    Announcement("announcement"),
    Emergency("emergency"),
}

object AnnouncementLogic {
    const val ORG_BROADCAST_MANAGE_PERMISSION = "tenant:org:roles:manage"
    const val GLOBAL_ADMIN_PERMISSION = "global:app:rbac:manage"

    fun canComposeCourseAnnouncement(course: CourseSummary): Boolean =
        course.viewerIsStaff && course.isFeedEnabled

    fun canComposeBroadcast(permissions: List<String>, features: MobilePlatformFeatures): Boolean {
        if (!features.ffBroadcasts) return false
        return ORG_BROADCAST_MANAGE_PERMISSION in permissions || GLOBAL_ADMIN_PERMISSION in permissions
    }

    fun resolveOrgId(courses: List<CourseSummary>): String? =
        courses.mapNotNull { it.orgId?.trim() }.firstOrNull { it.isNotEmpty() }

    fun announcementsChannelId(channels: List<FeedChannel>): String? =
        channels.firstOrNull { it.name.equals("announcements", ignoreCase = true) }?.id

    fun formatAnnouncementBody(
        title: String,
        body: String,
        sectionName: String?,
        mentionsEveryone: Boolean,
    ): String {
        val trimmedTitle = title.trim()
        val trimmedBody = body.trim()
        var text = "**$trimmedTitle**\n\n$trimmedBody"
        val section = sectionName?.trim().orEmpty()
        if (section.isNotEmpty()) {
            text += "\n\n_($section)_"
        }
        if (mentionsEveryone) {
            text += "\n\n@everyone"
        }
        return text
    }

    fun audienceLabel(
        course: CourseSummary,
        audience: AnnouncementAudience,
        sectionName: String?,
    ): String = when (audience) {
        AnnouncementAudience.WholeCourse -> course.displayTitle
        AnnouncementAudience.Section -> {
            val name = sectionName?.trim().orEmpty()
            if (name.isEmpty()) course.displayTitle else "${course.displayTitle} · $name"
        }
    }

    fun canSubmitCourseAnnouncement(title: String, body: String): Boolean =
        title.trim().isNotEmpty() && body.trim().isNotEmpty()

    fun canSubmitBroadcast(subject: String, body: String): Boolean =
        subject.trim().isNotEmpty() && body.trim().isNotEmpty()
}