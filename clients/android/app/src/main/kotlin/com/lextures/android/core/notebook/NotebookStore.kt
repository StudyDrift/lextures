package com.lextures.android.core.notebook

import android.content.Context
import android.util.Base64
import kotlinx.serialization.Serializable
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.jsonPrimitive
import java.time.Instant
import java.util.UUID

/** Mirrors web `student-notebook-storage` (device-local notebooks, format v2). */
@Serializable
data class NotebookPage(
    val id: String,
    val title: String = "Untitled",
    val parentId: String? = null,
    val sortOrder: Int = 0,
    val kind: String = "page",
    val contentMd: String = "",
) {
    companion object {
        fun new(title: String = "Untitled", sortOrder: Int = 0): NotebookPage =
            NotebookPage(id = UUID.randomUUID().toString(), title = title, sortOrder = sortOrder)
    }
}

@Serializable
data class CourseNotebook(
    val formatVersion: Int = 2,
    val updatedAt: String = "",
    val courseTitle: String? = null,
    val pages: List<NotebookPage> = emptyList(),
    val activePageId: String? = null,
) {
    val previewText: String
        get() = pages.joinToString("\n\n") { it.contentMd }.trim()

    companion object {
        fun empty(): CourseNotebook {
            val page = NotebookPage.new()
            return CourseNotebook(
                updatedAt = Instant.now().toString(),
                pages = listOf(page),
                activePageId = page.id,
            )
        }
    }
}

/** Device-local notebook persistence keyed per signed-in user (JWT `sub`), parity with web localStorage. */
class NotebookStore(context: Context, accessToken: String?) {
    private val prefs = context.applicationContext.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
    private val ownerKey = jwtSubject(accessToken) ?: "anonymous"
    private val json = Json { ignoreUnknownKeys = true }

    private val storageKey: String
        get() = "lextures.studentNotebooks.v1:$ownerKey"

    private fun readAll(): Map<String, CourseNotebook> {
        val raw = prefs.getString(storageKey, null) ?: return emptyMap()
        return runCatching { json.decodeFromString<Map<String, CourseNotebook>>(raw) }.getOrDefault(emptyMap())
    }

    private fun writeAll(notebooks: Map<String, CourseNotebook>) {
        prefs.edit().putString(storageKey, json.encodeToString(notebooks)).apply()
    }

    fun load(courseCode: String): CourseNotebook = readAll()[courseCode] ?: CourseNotebook.empty()

    fun save(courseCode: String, notebook: CourseNotebook) {
        val all = readAll().toMutableMap()
        all[courseCode] = notebook.copy(updatedAt = Instant.now().toString())
        writeAll(all)
    }

    fun exists(courseCode: String): Boolean = readAll().containsKey(courseCode)

    /** Every stored course code, including the global key. */
    fun allCourseCodes(): List<String> = readAll().keys.toList()

    /** Write a server copy verbatim — keeps the server `updatedAt` so last-write-wins stays stable. */
    fun saveFromServer(courseCode: String, notebook: CourseNotebook) {
        val all = readAll().toMutableMap()
        all[courseCode] = notebook
        writeAll(all)
    }

    /** Course-scoped notebooks with content (excludes the global key). */
    fun listCourseNotebooks(): Map<String, CourseNotebook> =
        readAll().filter { (key, notebook) -> key != GLOBAL_KEY && notebook.previewText.isNotEmpty() }

    companion object {
        /** Learner-wide notebook key — must not collide with real course codes (same value as web). */
        const val GLOBAL_KEY = "__lextures_global__"
        const val GLOBAL_TITLE = "Global notebook"
        private const val PREFS_NAME = "lextures_student_notebooks"

        fun jwtSubject(token: String?): String? {
            if (token.isNullOrBlank()) return null
            val parts = token.split(".")
            if (parts.size < 2) return null
            return runCatching {
                val payload = Base64.decode(parts[1], Base64.URL_SAFE or Base64.NO_PADDING or Base64.NO_WRAP)
                val obj = Json.parseToJsonElement(String(payload)) as? JsonObject ?: return null
                obj["sub"]?.jsonPrimitive?.content?.takeIf { it.isNotEmpty() }
            }.getOrNull()
        }
    }
}
