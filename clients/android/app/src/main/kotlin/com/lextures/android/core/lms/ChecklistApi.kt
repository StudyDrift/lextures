package com.lextures.android.core.lms

import com.lextures.android.core.network.ApiClient
import com.lextures.android.core.network.ApiError
import com.lextures.android.core.network.parseApiErrorMessage
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.json.Json
import java.net.URLEncoder

/** Course checklist endpoints (CC.2 / CC.9). */
object ChecklistApi {
    private val client = ApiClient()
    private val json = Json { ignoreUnknownKeys = true; coerceInputValues = true }

    private fun encodePath(value: String): String =
        URLEncoder.encode(value, "UTF-8").replace("+", "%20")

    private fun base(courseCode: String): String =
        "/api/v1/courses/${encodePath(courseCode)}/checklist"

    private inline fun <reified T> decode(body: String): T =
        try {
            json.decodeFromString(body)
        } catch (e: Exception) {
            throw ApiError.Decoding(e)
        }

    suspend fun fetchChecklist(courseCode: String, accessToken: String): CourseChecklist =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(path = base(courseCode), accessToken = accessToken)
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode(body)
        }

    suspend fun fetchChecklistSummary(courseCode: String, accessToken: String): CourseChecklistSummary =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(
                path = "${base(courseCode)}/summary",
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode(body)
        }

    suspend fun refreshChecklist(courseCode: String, accessToken: String): CourseChecklist =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(
                path = "${base(courseCode)}/refresh",
                method = "POST",
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode(body)
        }

    suspend fun dismissItem(
        courseCode: String,
        itemId: String,
        reason: ChecklistDismissReason,
        note: String?,
        accessToken: String,
    ): ChecklistItem = withContext(Dispatchers.IO) {
        val payload = json.encodeToString(
            ChecklistDismissBody.serializer(),
            ChecklistDismissBody(
                reason = reason.wireValue,
                note = CourseChecklistLogic.clampedNote(note),
            ),
        )
        val (body, code) = client.request(
            path = "${base(courseCode)}/items/${encodePath(itemId)}/dismiss",
            method = "POST",
            body = payload,
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun restoreItem(
        courseCode: String,
        itemId: String,
        accessToken: String,
    ): ChecklistItem = withContext(Dispatchers.IO) {
        val (body, code) = client.request(
            path = "${base(courseCode)}/items/${encodePath(itemId)}/restore",
            method = "POST",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun recheckItem(
        courseCode: String,
        itemId: String,
        accessToken: String,
    ): ChecklistItem = withContext(Dispatchers.IO) {
        val (body, code) = client.request(
            path = "${base(courseCode)}/items/${encodePath(itemId)}/recheck",
            method = "POST",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }
}
