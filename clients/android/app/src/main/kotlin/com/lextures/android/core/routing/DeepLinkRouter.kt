package com.lextures.android.core.routing

/** Parsed navigation target from a push tap, app link, or in-app notification action URL. */
sealed class DeepLinkDestination {
    data object Home : DeepLinkDestination()
    data object Inbox : DeepLinkDestination()
    data object Review : DeepLinkDestination()
    data object Insights : DeepLinkDestination()
    data object Billing : DeepLinkDestination()
    data object Credentials : DeepLinkDestination()
    data class CheckoutSuccess(val courseId: String? = null) : DeepLinkDestination()
    data object CheckoutCancel : DeepLinkDestination()
    data class Course(
        val code: String,
        val section: CourseDeepLinkSection? = null,
        val itemId: String? = null,
    ) : DeepLinkDestination()

    data class Parent(
        val studentId: String? = null,
        val section: ParentDeepLinkSection = ParentDeepLinkSection.Dashboard,
    ) : DeepLinkDestination()
}

enum class ParentDeepLinkSection {
    Dashboard,
    Grades,
    Attendance,
    Conferences,
    NotificationPrefs,
}

enum class CourseDeepLinkSection {
    Overview,
    Modules,
    Grades,
    Feed,
    Discussions,
    OfficeHours,
    Live,
    Files,
    Attendance,
    People,
    Evaluations,
    Library,
    Groups,
    CollabDocs,
}

/** Maps web-style action URLs and `lextures://` links to native navigation intents. */
object DeepLinkRouter {
    fun resolve(raw: String?): DeepLinkDestination {
        val trimmed = raw?.trim().orEmpty()
        if (trimmed.isEmpty()) return DeepLinkDestination.Home
        resolveCheckout(trimmed)?.let { return it }
        resolveParent(trimmed)?.let { return it }
        val path = extractPath(trimmed) ?: return DeepLinkDestination.Home
        return resolvePath(path, trimmed)
    }

    private fun extractPath(value: String): String? {
        if (value.startsWith("lextures://")) {
            val stripped = value.removePrefix("lextures://")
            return if (stripped.startsWith("/")) stripped else "/$stripped"
        }
        if (value.startsWith("/")) return value
        if (value.startsWith("http://") || value.startsWith("https://")) {
            val uri = runCatching { java.net.URI(value) }.getOrNull() ?: return null
            val host = uri.host?.lowercase().orEmpty()
            if (host == "lextures.com" || host.endsWith(".lextures.com") || host == "localhost") {
                var path = uri.path.orEmpty()
                if (!path.startsWith("/")) path = "/$path"
                return path.ifEmpty { null }
            }
            return null
        }
        val uri = runCatching { android.net.Uri.parse(value) }.getOrNull() ?: return null
        val host = uri.host?.lowercase().orEmpty()
        if (host == "lextures.com" || host.endsWith(".lextures.com") || host == "localhost") {
            var path = uri.path.orEmpty()
            if (!path.startsWith("/")) path = "/$path"
            return path
        }
        return null
    }

    private fun resolveParent(raw: String): DeepLinkDestination? {
        val urlString = if (raw.startsWith("/")) "https://lextures.com$raw" else raw
        val uri = runCatching { java.net.URI(urlString) }.getOrNull() ?: return null
        val path = uri.path.orEmpty()
        if (path != "/parent" && !path.startsWith("/parent/")) return null
        val studentId = uri.rawQuery.orEmpty().split("&")
            .map { it.split("=", limit = 2) }
            .firstOrNull { it.getOrNull(0) == "student" }
            ?.getOrNull(1)
        val section = when {
            path.contains("conferences") -> ParentDeepLinkSection.Conferences
            path.contains("notification") -> ParentDeepLinkSection.NotificationPrefs
            path.contains("grades") -> ParentDeepLinkSection.Grades
            path.contains("attendance") -> ParentDeepLinkSection.Attendance
            else -> ParentDeepLinkSection.Dashboard
        }
        return DeepLinkDestination.Parent(studentId = studentId, section = section)
    }

    private fun resolvePath(path: String, raw: String = path): DeepLinkDestination {
        val segments = path.trim('/').split('/').filter { it.isNotEmpty() }
        if (segments.firstOrNull()?.lowercase() != "courses" || segments.size < 2) {
            return when {
                segments.firstOrNull()?.lowercase() == "inbox" -> DeepLinkDestination.Inbox
                segments.firstOrNull()?.lowercase() == "review" -> DeepLinkDestination.Review
                segments.firstOrNull()?.lowercase() == "parent" -> {
                    val studentId = runCatching { android.net.Uri.parse(raw) }.getOrNull()?.getQueryParameter("student")
                    val section = when (segments.getOrNull(1)?.lowercase()) {
                        "conferences" -> ParentDeepLinkSection.Conferences
                        "notification-prefs" -> ParentDeepLinkSection.NotificationPrefs
                        "grades" -> ParentDeepLinkSection.Grades
                        "attendance" -> ParentDeepLinkSection.Attendance
                        else -> ParentDeepLinkSection.Dashboard
                    }
                    DeepLinkDestination.Parent(studentId = studentId, section = section)
                }
                segments.size >= 2 && segments[0].equals("me", ignoreCase = true) -> when {
                    segments[1].equals("study-insights", ignoreCase = true) -> DeepLinkDestination.Insights
                    segments[1].equals("credentials", ignoreCase = true) -> DeepLinkDestination.Credentials
                    else -> DeepLinkDestination.Home
                }
                else -> DeepLinkDestination.Home
            }
        }

        val courseCode = segments[1]
        if (segments.size == 2) {
            return DeepLinkDestination.Course(courseCode, CourseDeepLinkSection.Overview)
        }

        return when (segments[2].lowercase()) {
            "grades" -> DeepLinkDestination.Course(courseCode, CourseDeepLinkSection.Grades)
            "office-hours" -> DeepLinkDestination.Course(courseCode, CourseDeepLinkSection.OfficeHours)
            "feed" -> DeepLinkDestination.Course(courseCode, CourseDeepLinkSection.Feed)
            "discussions" -> {
                if (segments.size >= 5 && segments[3].equals("threads", ignoreCase = true)) {
                    DeepLinkDestination.Course(courseCode, CourseDeepLinkSection.Discussions, segments[4])
                } else {
                    DeepLinkDestination.Course(courseCode, CourseDeepLinkSection.Discussions)
                }
            }
            "live", "live-sessions" -> DeepLinkDestination.Course(courseCode, CourseDeepLinkSection.Live)
            "files" -> DeepLinkDestination.Course(courseCode, CourseDeepLinkSection.Files)
            "attendance" -> DeepLinkDestination.Course(courseCode, CourseDeepLinkSection.Attendance)
            "people", "enrollments" -> DeepLinkDestination.Course(courseCode, CourseDeepLinkSection.People)
            "evaluations", "evaluation-results" -> DeepLinkDestination.Course(courseCode, CourseDeepLinkSection.Evaluations)
            "library", "reading-dashboard" -> DeepLinkDestination.Course(courseCode, CourseDeepLinkSection.Library)
            "groups" -> DeepLinkDestination.Course(courseCode, CourseDeepLinkSection.Groups)
            "collab-docs" -> DeepLinkDestination.Course(
                code = courseCode,
                section = CourseDeepLinkSection.CollabDocs,
                itemId = segments.getOrNull(3),
            )
            "gradebook" -> DeepLinkDestination.Course(courseCode, CourseDeepLinkSection.Grades)
            "assignments", "quizzes", "modules" -> DeepLinkDestination.Course(
                code = courseCode,
                section = CourseDeepLinkSection.Modules,
                itemId = segments.getOrNull(3),
            )
            else -> DeepLinkDestination.Course(courseCode, CourseDeepLinkSection.Overview)
        }
    }

    private fun resolveCheckout(raw: String): DeepLinkDestination? {
        val urlString = if (raw.startsWith("/")) "https://lextures.com$raw" else raw
        val uri = runCatching { java.net.URI(urlString) }.getOrNull() ?: return null
        val query = uri.rawQuery.orEmpty()
        return when (uri.path) {
            "/checkout/success" -> {
                val courseId = query.split("&")
                    .map { it.split("=", limit = 2) }
                    .firstOrNull { it.getOrNull(0) == "course_id" }
                    ?.getOrNull(1)
                DeepLinkDestination.CheckoutSuccess(courseId)
            }
            "/checkout/cancel" -> DeepLinkDestination.CheckoutCancel
            "/me/billing" -> DeepLinkDestination.Billing
            "/me/credentials" -> DeepLinkDestination.Credentials
            else -> null
        }
    }
}
