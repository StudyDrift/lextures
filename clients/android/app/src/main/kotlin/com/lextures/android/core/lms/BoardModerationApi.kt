package com.lextures.android.core.lms

import com.lextures.android.core.network.ApiClient
import com.lextures.android.core.network.ApiError
import com.lextures.android.core.network.parseApiErrorMessage
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.json.Json
import java.net.URLEncoder

/** Board moderation, reports, and safety queue REST (VC.M7). Mirrors web `boards-api`. */
object BoardModerationApi {
    private val client = ApiClient()
    private val json = Json { ignoreUnknownKeys = true; coerceInputValues = true }

    private inline fun <reified T> decode(body: String): T =
        try {
            json.decodeFromString<T>(body)
        } catch (e: Exception) {
            throw ApiError.Decoding(e)
        }

    private fun encodePath(value: String): String =
        URLEncoder.encode(value, "UTF-8").replace("+", "%20")

    suspend fun fetchModerationQueue(
        courseCode: String,
        boardId: String,
        accessToken: String,
    ): BoardModerationQueue = withContext(Dispatchers.IO) {
        val (body, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/boards/${encodePath(boardId)}/moderation/queue",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun patchBoardModeration(
        courseCode: String,
        boardId: String,
        moderationMode: String? = null,
        filterAction: String? = null,
        locked: Boolean? = null,
        frozenUntil: String? = null,
        freezeMinutes: Int? = null,
        accessToken: String,
    ): Board = withContext(Dispatchers.IO) {
        val (body, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/boards/${encodePath(boardId)}",
            method = "PATCH",
            body = client.encodeBody(
                PatchBoardBody(
                    moderationMode = moderationMode,
                    filterAction = filterAction,
                    locked = locked,
                    frozenUntil = frozenUntil,
                    freezeMinutes = freezeMinutes,
                ),
                PatchBoardBody.serializer(),
            ),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun approvePost(
        courseCode: String,
        boardId: String,
        postId: String,
        reason: String? = null,
        accessToken: String,
    ): BoardPost = postAction(courseCode, boardId, postId, "approve", reason, accessToken)

    suspend fun rejectPost(
        courseCode: String,
        boardId: String,
        postId: String,
        reason: String? = null,
        accessToken: String,
    ): BoardPost = postAction(courseCode, boardId, postId, "reject", reason, accessToken)

    suspend fun hidePost(
        courseCode: String,
        boardId: String,
        postId: String,
        reason: String? = null,
        accessToken: String,
    ): BoardPost = postAction(courseCode, boardId, postId, "hide", reason, accessToken)

    suspend fun removePost(
        courseCode: String,
        boardId: String,
        postId: String,
        reason: String? = null,
        accessToken: String,
    ): BoardPost = postAction(courseCode, boardId, postId, "remove", reason, accessToken)

    suspend fun hideComment(
        courseCode: String,
        boardId: String,
        postId: String,
        commentId: String,
        reason: String? = null,
        accessToken: String,
    ): BoardComment = commentAction(courseCode, boardId, postId, commentId, "hide", reason, accessToken)

    suspend fun removeComment(
        courseCode: String,
        boardId: String,
        postId: String,
        commentId: String,
        reason: String? = null,
        accessToken: String,
    ): BoardComment = commentAction(courseCode, boardId, postId, commentId, "remove", reason, accessToken)

    suspend fun reportContent(
        courseCode: String,
        boardId: String,
        postId: String? = null,
        commentId: String? = null,
        reason: String? = null,
        accessToken: String,
    ): BoardReport = withContext(Dispatchers.IO) {
        val (body, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/boards/${encodePath(boardId)}/reports",
            method = "POST",
            body = client.encodeBody(
                CreateBoardReportBody(postId = postId, commentId = commentId, reason = reason),
                CreateBoardReportBody.serializer(),
            ),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun resolveReport(
        courseCode: String,
        boardId: String,
        reportId: String,
        action: String,
        reason: String? = null,
        accessToken: String,
    ): BoardReport = withContext(Dispatchers.IO) {
        val (body, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/boards/${encodePath(boardId)}/reports/${encodePath(reportId)}/resolve",
            method = "POST",
            body = client.encodeBody(
                ResolveBoardReportBody(action = action, reason = reason),
                ResolveBoardReportBody.serializer(),
            ),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    private suspend fun postAction(
        courseCode: String,
        boardId: String,
        postId: String,
        action: String,
        reason: String?,
        accessToken: String,
    ): BoardPost = withContext(Dispatchers.IO) {
        val (body, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/boards/${encodePath(boardId)}/posts/${encodePath(postId)}/$action",
            method = "POST",
            body = client.encodeBody(
                BoardModerationActionBody(reason = reason ?: ""),
                BoardModerationActionBody.serializer(),
            ),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    private suspend fun commentAction(
        courseCode: String,
        boardId: String,
        postId: String,
        commentId: String,
        action: String,
        reason: String?,
        accessToken: String,
    ): BoardComment = withContext(Dispatchers.IO) {
        val (body, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/boards/${encodePath(boardId)}/posts/${encodePath(postId)}/comments/${encodePath(commentId)}/$action",
            method = "POST",
            body = client.encodeBody(
                BoardModerationActionBody(reason = reason ?: ""),
                BoardModerationActionBody.serializer(),
            ),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }
}
