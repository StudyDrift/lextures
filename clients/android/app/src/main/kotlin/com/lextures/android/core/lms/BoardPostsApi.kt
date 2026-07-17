package com.lextures.android.core.lms

import com.lextures.android.core.network.ApiClient
import com.lextures.android.core.network.ApiError
import com.lextures.android.core.network.parseApiErrorMessage
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.json.Json
import java.net.URLEncoder

/** Board posts, attachments, and link preview REST (VC.M2). */
object BoardPostsApi {
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

    suspend fun listPosts(
        courseCode: String,
        boardId: String,
        accessToken: String,
    ): List<BoardPost> = withContext(Dispatchers.IO) {
        val (body, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/boards/${encodePath(boardId)}/posts",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode<BoardPostsListResponse>(body).posts.map(::normalizePost)
    }

    suspend fun createPost(
        courseCode: String,
        boardId: String,
        contentType: String,
        title: String? = null,
        body: BoardPostBody? = null,
        linkUrl: String? = null,
        attachmentId: String? = null,
        accessToken: String,
    ): BoardPost = withContext(Dispatchers.IO) {
        val (resp, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/boards/${encodePath(boardId)}/posts",
            method = "POST",
            body = client.encodeBody(
                CreateBoardPostBody(
                    contentType = contentType,
                    title = title,
                    body = body,
                    linkUrl = linkUrl,
                    attachmentId = attachmentId,
                ),
                CreateBoardPostBody.serializer(),
            ),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(resp))
        normalizePost(decode(resp))
    }

    suspend fun patchPost(
        courseCode: String,
        boardId: String,
        postId: String,
        title: String? = null,
        body: BoardPostBody? = null,
        linkUrl: String? = null,
        accessToken: String,
    ): BoardPost = withContext(Dispatchers.IO) {
        val (resp, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/boards/${encodePath(boardId)}/posts/${encodePath(postId)}",
            method = "PATCH",
            body = client.encodeBody(
                PatchBoardPostBody(title = title, body = body, linkUrl = linkUrl),
                PatchBoardPostBody.serializer(),
            ),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(resp))
        normalizePost(decode(resp))
    }

    suspend fun deletePost(
        courseCode: String,
        boardId: String,
        postId: String,
        accessToken: String,
    ) = withContext(Dispatchers.IO) {
        val (resp, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/boards/${encodePath(boardId)}/posts/${encodePath(postId)}",
            method = "DELETE",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(resp))
    }

    suspend fun uploadAttachment(
        courseCode: String,
        boardId: String,
        fileName: String,
        mimeType: String,
        fileBytes: ByteArray,
        altText: String? = null,
        contentType: String? = null,
        accessToken: String,
    ): BoardAttachment = withContext(Dispatchers.IO) {
        val extra = buildMap {
            if (!altText.isNullOrBlank()) put("altText", altText)
            if (!contentType.isNullOrBlank()) put("contentType", contentType)
        }
        val resp = client.uploadMultipart(
            path = "/api/v1/courses/${encodePath(courseCode)}/boards/${encodePath(boardId)}/attachments",
            fieldName = "file",
            fileName = fileName,
            mimeType = mimeType,
            fileBytes = fileBytes,
            accessToken = accessToken,
            extraFields = extra,
        )
        normalizeAttachment(decode(resp))
    }

    suspend fun fetchLinkPreview(
        courseCode: String,
        boardId: String,
        url: String,
        accessToken: String,
    ): BoardLinkPreview = withContext(Dispatchers.IO) {
        val (resp, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/boards/${encodePath(boardId)}/link-preview",
            method = "POST",
            body = client.encodeBody(BoardLinkPreviewBody(url), BoardLinkPreviewBody.serializer()),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(resp))
        decode(resp)
    }

    private fun normalizePost(post: BoardPost): BoardPost {
        val att = post.attachment ?: return post
        return post.copy(attachment = normalizeAttachment(att))
    }

    private fun normalizeAttachment(att: BoardAttachment): BoardAttachment {
        val absolute = att.url?.let { BoardsLogic.absoluteUrl(it) }
        return if (absolute != null) att.copy(url = absolute) else att
    }
}
