package com.lextures.android.core.routing

/** Parsed navigation target from a push tap, app link, or in-app notification action URL. */
sealed class DeepLinkDestination {
    data object Home : DeepLinkDestination()
    data object Inbox : DeepLinkDestination()
    data class Course(
        val code: String,
        val section: CourseDeepLinkSection? = null,
        val itemId: String? = null,
    ) : DeepLinkDestination()
}

enum class CourseDeepLinkSection {
    Overview,
    Modules,
    Grades,
    Feed,
    Discussions,
}

/** Maps web-style action URLs and `lextures://` links to native navigation intents. */
object DeepLinkRouter {
    fun resolve(raw: String?): DeepLinkDestination {
        val trimmed = raw?.trim().orEmpty()
        if (trimmed.isEmpty()) return DeepLinkDestination.Home
        val path = extractPath(trimmed) ?: return DeepLinkDestination.Home
        return resolvePath(path)
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

    private fun resolvePath(path: String): DeepLinkDestination {
        val segments = path.trim('/').split('/').filter { it.isNotEmpty() }
        if (segments.firstOrNull()?.lowercase() != "courses" || segments.size < 2) {
            return if (segments.firstOrNull()?.lowercase() == "inbox") {
                DeepLinkDestination.Inbox
            } else {
                DeepLinkDestination.Home
            }
        }

        val courseCode = segments[1]
        if (segments.size == 2) {
            return DeepLinkDestination.Course(courseCode, CourseDeepLinkSection.Overview)
        }

        return when (segments[2].lowercase()) {
            "grades" -> DeepLinkDestination.Course(courseCode, CourseDeepLinkSection.Grades)
            "feed" -> DeepLinkDestination.Course(courseCode, CourseDeepLinkSection.Feed)
            "discussions" -> DeepLinkDestination.Course(courseCode, CourseDeepLinkSection.Discussions)
            "assignments", "quizzes", "modules" -> DeepLinkDestination.Course(
                code = courseCode,
                section = CourseDeepLinkSection.Modules,
                itemId = segments.getOrNull(3),
            )
            else -> DeepLinkDestination.Course(courseCode, CourseDeepLinkSection.Overview)
        }
    }
}
