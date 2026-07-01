package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive

/** Pure helpers for AI tutor streaming and context (M7.2). */
object TutorLogic {
    const val MAX_MESSAGE_LENGTH = 2000

    fun shouldShowFab(course: CourseSummary): Boolean = course.isAiTutorEnabled

    fun askAiEnabled(platform: MobilePlatformFeatures): Boolean =
        platform.ragNotebookEnabled || platform.aiStudyBuddyEnabled

    fun disclosureStorageKey(courseCode: String?): String =
        if (!courseCode.isNullOrBlank()) "tutor-disclosure-$courseCode" else "tutor-disclosure-ask-ai"

    fun contextPrefix(itemTitle: String?, itemKind: String?): String? {
        val title = itemTitle?.trim().orEmpty()
        if (title.isEmpty()) return null
        val kind = itemKind?.trim().orEmpty()
        return if (kind.isNotEmpty()) {
            "[Context: viewing ${kind.replace('_', ' ')} \"$title\"]"
        } else {
            "[Context: viewing \"$title\"]"
        }
    }

    fun messageWithContext(
        text: String,
        itemTitle: String?,
        itemKind: String?,
        includeContext: Boolean,
    ): String {
        val trimmed = text.trim()
        if (!includeContext) return trimmed
        val prefix = contextPrefix(itemTitle, itemKind) ?: return trimmed
        return "$prefix\n\n$trimmed"
    }

    fun parseStreamEvent(jsonLine: String, json: Json = Json { ignoreUnknownKeys = true }): TutorStreamEvent? {
        return try {
            val root = json.parseToJsonElement(jsonLine).jsonObject
            when (root["type"]?.jsonPrimitive?.contentOrNull) {
                "content" -> {
                    val text = root["text"]?.jsonPrimitive?.contentOrNull ?: return null
                    TutorStreamEvent.Content(text)
                }
                "error" -> {
                    val message = root["message"]?.jsonPrimitive?.contentOrNull ?: "Stream error"
                    TutorStreamEvent.Error(message)
                }
                "done" -> TutorStreamEvent.Done(
                    conversationId = root["conversationId"]?.jsonPrimitive?.contentOrNull,
                    messageId = root["messageId"]?.jsonPrimitive?.contentOrNull,
                    sessionId = root["sessionId"]?.jsonPrimitive?.contentOrNull,
                    citations = decodeCitations(root["citations"]),
                )
                else -> null
            }
        } catch (_: Exception) {
            null
        }
    }

    fun parseSseLine(line: String, json: Json = Json { ignoreUnknownKeys = true }): TutorStreamEvent? {
        val trimmed = line.trim()
        if (!trimmed.startsWith("data: ")) return null
        return parseStreamEvent(trimmed.removePrefix("data: "), json)
    }

    fun gracefulHttpMessage(statusCode: Int, body: String?): String = when {
        statusCode == 402 || body?.contains("BUDGET_EXCEEDED") == true -> "BUDGET_EXCEEDED"
        statusCode == 403 -> body?.trim()?.takeIf { it.isNotEmpty() } ?: "FORBIDDEN"
        statusCode == 503 -> "UNAVAILABLE"
        else -> body?.trim()?.takeIf { it.isNotEmpty() } ?: "SEND_ERROR"
    }

    private fun decodeCitations(value: kotlinx.serialization.json.JsonElement?): List<TutorCitation> {
        val array = value as? JsonArray ?: return emptyList()
        return array.mapNotNull { element ->
            val row = element.jsonObject
            val sourceId = row.string("sourceId") ?: return@mapNotNull null
            val chunkId = row.string("chunkId") ?: return@mapNotNull null
            val excerpt = row.string("excerpt") ?: return@mapNotNull null
            TutorCitation(
                sourceId = sourceId,
                chunkId = chunkId,
                excerpt = excerpt,
                title = row.string("title"),
            )
        }
    }

    private fun JsonObject.string(key: String): String? = this[key]?.jsonPrimitive?.contentOrNull
}

sealed class TutorStreamEvent {
    data class Content(val text: String) : TutorStreamEvent()
    data class Error(val message: String) : TutorStreamEvent()
    data class Done(
        val conversationId: String? = null,
        val messageId: String? = null,
        val sessionId: String? = null,
        val citations: List<TutorCitation> = emptyList(),
    ) : TutorStreamEvent()
}

data class TutorDisplayMessage(
    val id: String = java.util.UUID.randomUUID().toString(),
    val role: String,
    val content: String,
    val citations: List<TutorCitation> = emptyList(),
    val isStreaming: Boolean = false,
)