package com.lextures.android.core.lms

import com.lextures.android.core.network.ApiClient
import com.lextures.android.core.network.ApiError
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.json.Json
import java.net.URLEncoder

/** LMS endpoints used by the post-auth tabs (parity with web `courses-api` / `communication-api`). */
object LmsApi {
    private val client = ApiClient()
    private val json = Json { ignoreUnknownKeys = true }

    private inline fun <reified T> decode(body: String): T =
        try {
            json.decodeFromString<T>(body)
        } catch (e: Exception) {
            throw ApiError.Decoding(e)
        }

    private fun encodeQuery(value: String): String = URLEncoder.encode(value, "UTF-8")

    private fun encodePath(value: String): String = encodeQuery(value).replace("+", "%20")

    // Courses

    suspend fun fetchCourses(accessToken: String): List<CourseSummary> = withContext(Dispatchers.IO) {
        val (body, _) = client.request("/api/v1/courses", accessToken = accessToken)
        decode<CoursesResponse>(body).courses
    }

    /** Single-course GET includes `viewerEnrollmentRoles` (list GET does not). */
    suspend fun fetchCourse(courseCode: String, accessToken: String): CourseSummary = withContext(Dispatchers.IO) {
        val (body, _) = client.request("/api/v1/courses/${encodePath(courseCode)}", accessToken = accessToken)
        decode<CourseSummary>(body)
    }

    suspend fun fetchCourseStructure(courseCode: String, accessToken: String): List<CourseStructureItem> =
        withContext(Dispatchers.IO) {
            val (body, _) = client.request(
                "/api/v1/courses/${encodePath(courseCode)}/structure",
                accessToken = accessToken,
            )
            decode<CourseStructureResponse>(body).items
        }

    // Inbox (communication)

    suspend fun fetchMailboxMessages(
        folder: MailboxFolder,
        query: String,
        accessToken: String,
    ): List<MailboxMessage> = withContext(Dispatchers.IO) {
        val params = "folder=${folder.wire}&q=${encodeQuery(query.trim())}"
        val (body, _) = client.request("/api/v1/communication/messages?$params", accessToken = accessToken)
        decode<MailboxMessagesResponse>(body).messages
    }

    suspend fun fetchUnreadInboxCount(accessToken: String): Int = withContext(Dispatchers.IO) {
        val (body, _) = client.request("/api/v1/communication/unread-count", accessToken = accessToken)
        decode<UnreadInboxResponse>(body).unreadInbox ?: 0
    }

    suspend fun patchMailbox(messageId: String, patch: MailboxPatchRequest, accessToken: String) {
        withContext(Dispatchers.IO) {
            client.request(
                path = "/api/v1/communication/messages/${encodePath(messageId)}",
                method = "PATCH",
                body = client.encodeBody(patch, MailboxPatchRequest.serializer()),
                accessToken = accessToken,
            )
        }
    }

    suspend fun sendMessage(request: SendMessageRequest, accessToken: String) {
        withContext(Dispatchers.IO) {
            client.request(
                path = "/api/v1/communication/messages",
                method = "POST",
                body = client.encodeBody(request, SendMessageRequest.serializer()),
                accessToken = accessToken,
            )
        }
    }
}
