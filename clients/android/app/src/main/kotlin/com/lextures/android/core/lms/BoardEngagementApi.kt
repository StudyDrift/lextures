package com.lextures.android.core.lms

import com.lextures.android.core.network.ApiClient
import com.lextures.android.core.network.ApiError
import com.lextures.android.core.network.parseApiErrorMessage
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.json.Json
import java.net.URLEncoder

/** Board reactions, comments, and grade-sync REST (VC.M5). */
object BoardEngagementApi {
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

    suspend fun putReaction(
        courseCode: String,
        boardId: String,
        postId: String,
        kind: String? = null,
        value: Double? = null,
        accessToken: String,
    ): BoardReactionResult = withContext(Dispatchers.IO) {
        val (resp, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/boards/${encodePath(boardId)}/posts/${encodePath(postId)}/reaction",
            method = "PUT",
            body = client.encodeBody(
                PutBoardReactionBody(kind = kind, value = value),
                PutBoardReactionBody.serializer(),
            ),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(resp))
        decode(resp)
    }

    suspend fun deleteReaction(
        courseCode: String,
        boardId: String,
        postId: String,
        accessToken: String,
    ) = withContext(Dispatchers.IO) {
        val (resp, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/boards/${encodePath(boardId)}/posts/${encodePath(postId)}/reaction",
            method = "DELETE",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(resp))
    }

    suspend fun listComments(
        courseCode: String,
        boardId: String,
        postId: String,
        accessToken: String,
    ): List<BoardComment> = withContext(Dispatchers.IO) {
        val (resp, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/boards/${encodePath(boardId)}/posts/${encodePath(postId)}/comments",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(resp))
        decode<BoardCommentsListResponse>(resp).comments
    }

    suspend fun createComment(
        courseCode: String,
        boardId: String,
        postId: String,
        body: BoardPostBody,
        parentId: String? = null,
        accessToken: String,
    ): BoardComment = withContext(Dispatchers.IO) {
        val (resp, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/boards/${encodePath(boardId)}/posts/${encodePath(postId)}/comments",
            method = "POST",
            body = client.encodeBody(
                CreateBoardCommentBody(body = body, parentId = parentId),
                CreateBoardCommentBody.serializer(),
            ),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(resp))
        decode(resp)
    }

    suspend fun patchComment(
        courseCode: String,
        boardId: String,
        postId: String,
        commentId: String,
        body: BoardPostBody? = null,
        hidden: Boolean? = null,
        accessToken: String,
    ): BoardComment = withContext(Dispatchers.IO) {
        val (resp, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/boards/${encodePath(boardId)}/posts/${encodePath(postId)}/comments/${encodePath(commentId)}",
            method = "PATCH",
            body = client.encodeBody(
                PatchBoardCommentBody(body = body, hidden = hidden),
                PatchBoardCommentBody.serializer(),
            ),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(resp))
        decode(resp)
    }

    suspend fun deleteComment(
        courseCode: String,
        boardId: String,
        postId: String,
        commentId: String,
        accessToken: String,
    ) = withContext(Dispatchers.IO) {
        val (resp, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/boards/${encodePath(boardId)}/posts/${encodePath(postId)}/comments/${encodePath(commentId)}",
            method = "DELETE",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(resp))
    }

    suspend fun syncGrade(
        courseCode: String,
        boardId: String,
        postId: String,
        accessToken: String,
    ): BoardGradeSyncResult = withContext(Dispatchers.IO) {
        val (resp, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/boards/${encodePath(boardId)}/posts/${encodePath(postId)}/grade-sync",
            method = "POST",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(resp))
        decode(resp)
    }
}
