package com.lextures.android.core.lms

import com.lextures.android.core.network.ApiClient
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json

/** Notebook-task body sent to the dashboard sync endpoint (`dueAt: null` clears, web parity). */
@Serializable
data class NotebookTaskUpsert(
    val id: String,
    val courseCode: String,
    val notebookPageId: String,
    val taskText: String,
    val completed: Boolean,
    val dueAt: String?,
)

/** Server-side notebook tasks (dashboard sync, parity with web `notebook-tasks-api`). */
object NotebookTasksApi {
    private val client = ApiClient()
    private val json = Json { ignoreUnknownKeys = true }

    suspend fun upsert(task: NotebookTaskUpsert, accessToken: String): Unit = withContext(Dispatchers.IO) {
        client.request(
            "/api/v1/me/notebook-tasks",
            method = "POST",
            body = json.encodeToString(NotebookTaskUpsert.serializer(), task),
            accessToken = accessToken,
        )
    }
}
