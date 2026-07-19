package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive

/**
 * Canvas course import helpers (MOB.2) — credentials, include map, WS parsing, gates.
 *
 * The Canvas access token is held only in memory for the request lifetime and must never
 * be written to DataStore, SharedPreferences, logs, or crash reports.
 */
object CanvasImportLogic {
    const val COURSE_CREATE_PERMISSION = CourseCreateLogic.COURSE_CREATE_PERMISSION
    const val CANCELLED_MESSAGE = "Import cancelled."
    const val TOKEN_MUST_NOT_PERSIST_POLICY =
        "Canvas access tokens stay in memory for the active request only and are never persisted."

    enum class ImportStep(val number: Int) {
        Credentials(0),
        Select(1),
        Importing(2),
        ;

        companion object {
            fun fromNumber(n: Int): ImportStep = entries.firstOrNull { it.number == n } ?: Credentials
        }
    }

    enum class TargetMode(val value: String) {
        NewCourse("newCourse"),
        ExistingCourse("existingCourse"),
    }

    enum class IncludeCategory(val value: String) {
        Modules("modules"),
        Assignments("assignments"),
        Quizzes("quizzes"),
        Enrollments("enrollments"),
        Grades("grades"),
        Settings("settings"),
        Files("files"),
        Announcements("announcements"),
    }

    data class Include(
        val modules: Boolean = true,
        val assignments: Boolean = true,
        val quizzes: Boolean = true,
        val enrollments: Boolean = true,
        val grades: Boolean = true,
        val settings: Boolean = true,
        val files: Boolean = true,
        val announcements: Boolean = true,
    ) {
        fun value(category: IncludeCategory): Boolean = when (category) {
            IncludeCategory.Modules -> modules
            IncludeCategory.Assignments -> assignments
            IncludeCategory.Quizzes -> quizzes
            IncludeCategory.Enrollments -> enrollments
            IncludeCategory.Grades -> grades
            IncludeCategory.Settings -> settings
            IncludeCategory.Files -> files
            IncludeCategory.Announcements -> announcements
        }

        fun set(category: IncludeCategory, enabled: Boolean): Include = when (category) {
            IncludeCategory.Modules -> copy(modules = enabled)
            IncludeCategory.Assignments -> copy(assignments = enabled)
            IncludeCategory.Quizzes -> copy(quizzes = enabled)
            IncludeCategory.Enrollments -> copy(enrollments = enabled)
            IncludeCategory.Grades -> copy(grades = enabled)
            IncludeCategory.Settings -> copy(settings = enabled)
            IncludeCategory.Files -> copy(files = enabled)
            IncludeCategory.Announcements -> copy(announcements = enabled)
        }

        /** Category counts for telemetry — never includes the token. */
        fun enabledCategoryCounts(): Map<String, Int> =
            IncludeCategory.entries.associate { it.value to if (value(it)) 1 else 0 }

        companion object {
            val ALL = Include()
        }
    }

    enum class WSMessageType {
        Progress,
        Complete,
        CoursesUpdated,
        Error,
        Unknown,
    }

    data class WSMessage(
        val type: WSMessageType,
        val message: String? = null,
        val courseCode: String? = null,
    )

    fun canvasImportEnabled(features: MobilePlatformFeatures): Boolean =
        features.ffMobileCanvasImport

    fun shouldShowCanvasImportEntry(
        permissions: List<String>,
        features: MobilePlatformFeatures,
        isOnline: Boolean,
    ): Boolean {
        if (!canvasImportEnabled(features)) return false
        if (!CourseCreateLogic.courseCreateV2Enabled(features)) return false
        if (!CourseCreateLogic.canCreateCourses(permissions)) return false
        return isOnline
    }

    fun normalizeBaseUrl(raw: String): String {
        var value = raw.trim()
        while (value.endsWith("/")) value = value.dropLast(1)
        return value
    }

    /** Returns a localization key when credentials are incomplete/invalid. */
    fun validateCredentials(baseUrl: String, accessToken: String): String? {
        val url = normalizeBaseUrl(baseUrl)
        val token = accessToken.trim()
        if (url.isEmpty()) return "mobile.canvasImport.error.urlRequired"
        val lower = url.lowercase()
        if (!(lower.startsWith("https://") || lower.startsWith("http://"))) {
            return "mobile.canvasImport.error.urlInvalid"
        }
        if (token.isEmpty()) return "mobile.canvasImport.error.tokenRequired"
        return null
    }

    fun filterCourses(courses: List<CanvasCourseListItem>, query: String): List<CanvasCourseListItem> {
        val q = query.trim().lowercase()
        if (q.isEmpty()) return courses
        return courses.filter { course ->
            listOf(
                course.name,
                course.courseCode.orEmpty(),
                course.termName.orEmpty(),
                course.id.toString(),
            ).joinToString(" ").lowercase().contains(q)
        }
    }

    fun isUnpublished(workflowState: String?): Boolean =
        workflowState?.trim()?.lowercase() == "unpublished"

    fun parseWSMessage(raw: String): WSMessage? {
        return try {
            val obj = jsonParser.parseToJsonElement(raw).jsonObject
            val typeRaw = obj["type"]?.jsonPrimitive?.contentOrNull.orEmpty()
            val type = when (typeRaw) {
                "progress" -> WSMessageType.Progress
                "complete" -> WSMessageType.Complete
                "courses_updated" -> WSMessageType.CoursesUpdated
                "error" -> WSMessageType.Error
                else -> WSMessageType.Unknown
            }
            WSMessage(
                type = type,
                message = obj["message"]?.jsonPrimitive?.contentOrNull?.takeIf { it.isNotBlank() },
                courseCode = obj["courseCode"]?.jsonPrimitive?.contentOrNull?.takeIf { it.isNotBlank() },
            )
        } catch (_: Exception) {
            null
        }
    }

    private val jsonParser = Json { ignoreUnknownKeys = true }

    fun isTerminal(type: WSMessageType): Boolean = when (type) {
        WSMessageType.Complete, WSMessageType.CoursesUpdated, WSMessageType.Error -> true
        WSMessageType.Progress, WSMessageType.Unknown -> false
    }

    /** Security helper for tests: forbidden persistence keys must never contain a token. */
    fun storageContainsToken(haystacks: List<String>, token: String): Boolean {
        val trimmed = token.trim()
        if (trimmed.isEmpty()) return false
        return haystacks.any { it.contains(trimmed) }
    }

    fun jobWebSocketPath(jobId: String): String =
        "/api/v1/ws/canvas-import/${encodePathComponent(jobId)}"

    fun defaultImportMode(target: TargetMode): CourseImportExportLogic.ImportMode = when (target) {
        TargetMode.NewCourse -> CourseImportExportLogic.ImportMode.erase
        TargetMode.ExistingCourse -> CourseImportExportLogic.ImportMode.mergeAdd
    }

    sealed class CanvasImportError(message: String) : Exception(message) {
        data object Cancelled : CanvasImportError(CANCELLED_MESSAGE)
        data object MissingJobId : CanvasImportError("Server did not return an import job id.")
        data object ConnectionClosed : CanvasImportError("Connection closed before import finished.")
        data object ConnectionError : CanvasImportError("Connection error during Canvas import.")
        data class Server(val detail: String) : CanvasImportError(detail)
    }

    private fun encodePathComponent(value: String): String =
        java.net.URLEncoder.encode(value, Charsets.UTF_8.name()).replace("+", "%20")
}
