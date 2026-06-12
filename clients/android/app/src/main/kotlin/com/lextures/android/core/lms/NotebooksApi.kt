package com.lextures.android.core.lms

import com.lextures.android.core.network.ApiClient
import com.lextures.android.core.notebook.CourseNotebook
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import java.net.URLEncoder

/** One server notebook row; `data` is null when the stored document fails to decode. */
data class ServerNotebookEntry(
    val courseCode: String,
    val updatedAt: String,
    val data: CourseNotebook?,
)

/** Server-side notebook documents (sync, parity with web `student-notebook-sync`). */
object NotebooksApi {
    private val client = ApiClient()
    private val json = Json { ignoreUnknownKeys = true }

    private fun encodeQuery(value: String): String = URLEncoder.encode(value, "UTF-8")

    suspend fun fetchAll(accessToken: String): List<ServerNotebookEntry> = withContext(Dispatchers.IO) {
        val (body, _) = client.request("/api/v1/me/notebooks", accessToken = accessToken)
        val root = json.parseToJsonElement(body).jsonObject
        (root["notebooks"]?.jsonArray ?: return@withContext emptyList()).mapNotNull { el ->
            val obj = el as? JsonObject ?: return@mapNotNull null
            val courseCode = obj["courseCode"]?.jsonPrimitive?.content ?: return@mapNotNull null
            val updatedAt = obj["updatedAt"]?.jsonPrimitive?.content ?: return@mapNotNull null
            // Lenient: one malformed document must not break the whole list.
            val data = obj["data"]?.let { runCatching { json.decodeFromJsonElement(CourseNotebook.serializer(), it) }.getOrNull() }
            ServerNotebookEntry(courseCode, updatedAt, data)
        }
    }

    suspend fun put(courseCode: String, notebook: CourseNotebook, accessToken: String): Unit =
        withContext(Dispatchers.IO) {
            client.request(
                "/api/v1/me/notebooks?courseCode=${encodeQuery(courseCode)}",
                method = "PUT",
                body = json.encodeToString(CourseNotebook.serializer(), notebook),
                accessToken = accessToken,
            )
        }
}
