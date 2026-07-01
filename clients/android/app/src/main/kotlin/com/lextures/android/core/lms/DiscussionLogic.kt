package com.lextures.android.core.lms

import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonArray
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.jsonPrimitive
import kotlinx.serialization.json.put
import kotlinx.serialization.json.putJsonArray

data class DiscussionNestedPost(
    val post: DiscussionPost,
    val depth: Int,
)

object DiscussionLogic {
    fun nestPosts(posts: List<DiscussionPost>): List<DiscussionNestedPost> {
        val byParent = posts.groupBy { it.parentPostId }.mapValues { (_, list) ->
            list.sortedBy { it.createdAt }
        }
        val output = mutableListOf<DiscussionNestedPost>()
        fun walk(parentId: String?, depth: Int) {
            for (child in byParent[parentId].orEmpty()) {
                output += DiscussionNestedPost(child, depth)
                walk(child.id, depth + 1)
            }
        }
        walk(null, 0)
        return output
    }

    fun sortThreads(threads: List<DiscussionThreadSummary>): List<DiscussionThreadSummary> =
        threads.sortedWith { lhs, rhs ->
            when {
                lhs.isPinned != rhs.isPinned -> if (lhs.isPinned) -1 else 1
                else -> rhs.updatedAt.compareTo(lhs.updatedAt)
            }
        }

    fun authorLabel(authorId: String, viewerId: String?, youLabel: String): String {
        if (authorId == viewerId) return youLabel
        val trimmed = authorId.trim()
        return if (trimmed.length <= 8) trimmed else trimmed.take(8)
    }

    fun canReply(thread: DiscussionThreadDetail, viewerIsStaff: Boolean): Boolean =
        !thread.isLocked || viewerIsStaff

    fun canDeletePost(post: DiscussionPost, viewerId: String?): Boolean =
        viewerId != null && post.authorId == viewerId

    fun isBodyEmpty(text: String): Boolean = text.trim().isEmpty()

    fun plainText(body: JsonElement): String = extractText(body).trim()

    fun encodeBody(text: String): JsonElement {
        val trimmed = text.trim()
        val lines = if (trimmed.isEmpty()) listOf("") else trimmed.split('\n')
        return buildJsonObject {
            put("type", "doc")
            putJsonArray("content") {
                lines.forEach { line ->
                    add(
                        buildJsonObject {
                            put("type", "paragraph")
                            putJsonArray("content") {
                                add(
                                    buildJsonObject {
                                        put("type", "text")
                                        put("text", line)
                                    },
                                )
                            }
                        },
                    )
                }
            }
        }
    }

    private fun extractText(value: JsonElement): String =
        when (value) {
            is JsonPrimitive -> value.contentOrNull.orEmpty()
            is JsonArray -> value.joinToString("") { extractText(it) }
            is JsonObject -> {
                value["text"]?.jsonPrimitive?.contentOrNull
                    ?: value["content"]?.let { content ->
                        val separator =
                            if (value["type"]?.jsonPrimitive?.contentOrNull == "doc") "\n" else ""
                        when (content) {
                            is JsonArray -> content.joinToString(separator) { extractText(it) }
                            else -> extractText(content)
                        }
                    }
                    ?: ""
            }
            else -> ""
        }
}
