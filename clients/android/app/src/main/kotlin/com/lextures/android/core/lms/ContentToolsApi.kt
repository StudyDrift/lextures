package com.lextures.android.core.lms

import com.lextures.android.core.network.ApiClient
import com.lextures.android.core.network.ApiError
import com.lextures.android.core.network.parseApiErrorMessage
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.decodeFromJsonElement
import kotlinx.serialization.json.jsonObject
import java.net.URLEncoder

/** CT.M3 — Content Tools HTTP client (consumes shipped web APIs). */
object ContentToolsApi {
    private val client = ApiClient()
    private val json = Json { ignoreUnknownKeys = true; coerceInputValues = true }

    class RevisionConflictException(val current: ToolStateEnvelope) : Exception("revision_conflict")
    class StateTooLargeException(val maxBytes: Long) : Exception("state_too_large")
    class SchemaInvalidException(message: String) : Exception(message)

    private fun encodePath(value: String): String =
        URLEncoder.encode(value, "UTF-8").replace("+", "%20")

    private fun base(courseCode: String): String =
        "/api/v1/courses/${encodePath(courseCode)}/content-tools"

    suspend fun fetchInstances(
        courseCode: String,
        accessToken: String,
        itemId: String? = null,
        hostKind: String? = null,
        withState: Boolean = true,
    ): List<ToolInstance> = withContext(Dispatchers.IO) {
        val qs = buildString {
            val parts = mutableListOf<String>()
            if (!itemId.isNullOrBlank()) parts += "itemId=${encodePath(itemId)}"
            if (!hostKind.isNullOrBlank()) parts += "hostKind=${encodePath(hostKind)}"
            if (withState) parts += "withState=1"
            if (parts.isNotEmpty()) append('?').append(parts.joinToString("&"))
        }
        val (body, status) = client.requestRaw(
            path = "${base(courseCode)}/instances$qs",
            method = "GET",
            accessToken = accessToken,
        )
        if (status !in 200..299) {
            throw ApiError.HttpStatus(status, parseApiErrorMessage(body))
        }
        json.decodeFromString(ToolInstancesListResponse.serializer(), body).instances
    }

    suspend fun fetchSettings(
        courseCode: String,
        accessToken: String,
    ): ContentToolSettings = withContext(Dispatchers.IO) {
        val (body, status) = client.requestRaw(
            path = "${base(courseCode)}/settings",
            method = "GET",
            accessToken = accessToken,
        )
        if (status !in 200..299) {
            throw ApiError.HttpStatus(status, parseApiErrorMessage(body))
        }
        json.decodeFromString(ContentToolSettings.serializer(), body)
    }

    suspend fun getState(
        courseCode: String,
        instanceId: String,
        accessToken: String,
        scope: String? = null,
    ): ToolStateEnvelope = withContext(Dispatchers.IO) {
        val qs = if (!scope.isNullOrBlank()) "?scope=${encodePath(scope)}" else ""
        val (body, status) = client.requestRaw(
            path = "${base(courseCode)}/instances/${encodePath(instanceId)}/state$qs",
            method = "GET",
            accessToken = accessToken,
        )
        if (status !in 200..299) {
            throw ApiError.HttpStatus(status, parseApiErrorMessage(body))
        }
        json.decodeFromString(ToolStateEnvelope.serializer(), body)
    }

    suspend fun putState(
        courseCode: String,
        instanceId: String,
        revision: Long,
        state: JsonElement,
        accessToken: String,
        scope: String? = null,
        idempotencyKey: String? = null,
    ): ToolStateEnvelope = withContext(Dispatchers.IO) {
        val qs = if (!scope.isNullOrBlank()) "?scope=${encodePath(scope)}" else ""
        val payload = SaveToolStateBody(revision = revision, state = state, stateJson = state)
        try {
            val (body, _) = client.requestRaw(
                path = "${base(courseCode)}/instances/${encodePath(instanceId)}/state$qs",
                method = "PUT",
                body = json.encodeToString(SaveToolStateBody.serializer(), payload),
                accessToken = accessToken,
                idempotencyKey = idempotencyKey,
            )
            json.decodeFromString(ToolStateEnvelope.serializer(), body)
        } catch (e: ApiError.HttpStatus) {
            throw remapStateWriteError(e, courseCode, instanceId, accessToken, scope)
        }
    }

    suspend fun submit(
        courseCode: String,
        instanceId: String,
        revision: Long,
        state: JsonElement,
        accessToken: String,
    ): ToolStateEnvelope = withContext(Dispatchers.IO) {
        val payload = SaveToolStateBody(revision = revision, state = state, stateJson = state)
        try {
            val (body, _) = client.requestRaw(
                path = "${base(courseCode)}/instances/${encodePath(instanceId)}/submit",
                method = "POST",
                body = json.encodeToString(SaveToolStateBody.serializer(), payload),
                accessToken = accessToken,
            )
            json.decodeFromString(ToolStateEnvelope.serializer(), body)
        } catch (e: ApiError.HttpStatus) {
            throw remapStateWriteError(e, courseCode, instanceId, accessToken, scope = null)
        }
    }

    suspend fun runAction(
        courseCode: String,
        instanceId: String,
        action: String,
        input: JsonElement = JsonObject(emptyMap()),
        accessToken: String,
        idempotencyKey: String,
    ): RunToolActionResponse = withContext(Dispatchers.IO) {
        val payload = RunToolActionBody(input = input, idempotencyKey = idempotencyKey)
        val (body, status) = client.requestRaw(
            path = "${base(courseCode)}/instances/${encodePath(instanceId)}/actions/${encodePath(action)}",
            method = "POST",
            body = json.encodeToString(RunToolActionBody.serializer(), payload),
            accessToken = accessToken,
            idempotencyKey = idempotencyKey,
        )
        if (status !in 200..299) {
            throw ApiError.HttpStatus(status, parseApiErrorMessage(body))
        }
        json.decodeFromString(RunToolActionResponse.serializer(), body)
    }

    suspend fun fetchAIConsent(
        courseCode: String,
        toolId: String,
        accessToken: String,
    ): ContentToolAIConsent = withContext(Dispatchers.IO) {
        val qs = "?toolId=${encodePath(toolId)}"
        val (body, status) = client.requestRaw(
            path = "${base(courseCode)}/ai-consent$qs",
            method = "GET",
            accessToken = accessToken,
        )
        if (status !in 200..299) {
            throw ApiError.HttpStatus(status, parseApiErrorMessage(body))
        }
        json.decodeFromString(ContentToolAIConsent.serializer(), body)
    }

    suspend fun postAIConsent(
        courseCode: String,
        toolId: String?,
        decision: String,
        accessToken: String,
    ): ContentToolAIConsent = withContext(Dispatchers.IO) {
        val payload = ContentToolAIConsentBody(toolId = toolId, decision = decision)
        val (body, status) = client.requestRaw(
            path = "${base(courseCode)}/ai-consent",
            method = "POST",
            body = json.encodeToString(ContentToolAIConsentBody.serializer(), payload),
            accessToken = accessToken,
        )
        if (status !in 200..299) {
            throw ApiError.HttpStatus(status, parseApiErrorMessage(body))
        }
        json.decodeFromString(ContentToolAIConsent.serializer(), body)
    }

    suspend fun selfReset(
        courseCode: String,
        instanceId: String,
        accessToken: String,
    ): ToolStateEnvelope = withContext(Dispatchers.IO) {
        val (body, status) = client.requestRaw(
            path = "${base(courseCode)}/instances/${encodePath(instanceId)}/self-reset",
            method = "POST",
            accessToken = accessToken,
        )
        if (status !in 200..299) {
            throw ApiError.HttpStatus(status, parseApiErrorMessage(body))
        }
        json.decodeFromString(ToolStateEnvelope.serializer(), body)
    }

    suspend fun reportContent(
        courseCode: String,
        instanceId: String,
        category: String?,
        reason: String?,
        contentPath: String?,
        accessToken: String,
    ): ContentToolModerationAction = withContext(Dispatchers.IO) {
        val payload = ContentToolReportBody(category = category, reason = reason, contentPath = contentPath)
        val (body, status) = client.requestRaw(
            path = "${base(courseCode)}/instances/${encodePath(instanceId)}/report",
            method = "POST",
            body = json.encodeToString(ContentToolReportBody.serializer(), payload),
            accessToken = accessToken,
        )
        if (status !in 200..299) {
            throw ApiError.HttpStatus(status, parseApiErrorMessage(body))
        }
        json.decodeFromString(ContentToolModerationAction.serializer(), body)
    }

    suspend fun moderateContent(
        courseCode: String,
        instanceId: String,
        action: String,
        category: String?,
        reason: String?,
        contentPath: String?,
        accessToken: String,
    ): ContentToolModerationAction = withContext(Dispatchers.IO) {
        val payload = ContentToolModerateBody(
            action = action,
            category = category,
            reason = reason,
            contentPath = contentPath,
        )
        val (body, status) = client.requestRaw(
            path = "${base(courseCode)}/instances/${encodePath(instanceId)}/moderate",
            method = "POST",
            body = json.encodeToString(ContentToolModerateBody.serializer(), payload),
            accessToken = accessToken,
        )
        if (status !in 200..299) {
            throw ApiError.HttpStatus(status, parseApiErrorMessage(body))
        }
        runCatching {
            val obj = json.parseToJsonElement(body).jsonObject
            obj["action"]?.let {
                json.decodeFromJsonElement(ContentToolModerationAction.serializer(), it)
            }
        }.getOrNull() ?: json.decodeFromString(ContentToolModerationAction.serializer(), body)
    }

    suspend fun fetchModeration(
        courseCode: String,
        instanceId: String,
        accessToken: String,
    ): List<ContentToolModerationAction> = withContext(Dispatchers.IO) {
        val (body, status) = client.requestRaw(
            path = "${base(courseCode)}/instances/${encodePath(instanceId)}/moderation",
            method = "GET",
            accessToken = accessToken,
        )
        if (status !in 200..299) {
            throw ApiError.HttpStatus(status, parseApiErrorMessage(body))
        }
        json.decodeFromString(ContentToolModerationListResponse.serializer(), body).items
    }

    suspend fun fetchFilterFlags(
        courseCode: String,
        instanceId: String,
        accessToken: String,
    ): List<JsonElement> = withContext(Dispatchers.IO) {
        val (body, status) = client.requestRaw(
            path = "${base(courseCode)}/instances/${encodePath(instanceId)}/filter-flags",
            method = "GET",
            accessToken = accessToken,
        )
        if (status !in 200..299) {
            throw ApiError.HttpStatus(status, parseApiErrorMessage(body))
        }
        json.decodeFromString(ContentToolFilterFlagsResponse.serializer(), body).items
    }

    suspend fun fetchConformance(accessToken: String): ContentToolConformanceResponse =
        withContext(Dispatchers.IO) {
            val (body, status) = client.requestRaw(
                path = "/api/v1/content-tools/conformance",
                method = "GET",
                accessToken = accessToken,
            )
            if (status !in 200..299) {
                throw ApiError.HttpStatus(status, parseApiErrorMessage(body))
            }
            json.decodeFromString(ContentToolConformanceResponse.serializer(), body)
        }

    fun statePutPath(courseCode: String, instanceId: String): String =
        "${base(courseCode)}/instances/${encodePath(instanceId)}/state"

    fun encodeStateBody(revision: Long, state: JsonElement): String =
        json.encodeToString(
            SaveToolStateBody.serializer(),
            SaveToolStateBody(revision = revision, state = state, stateJson = state),
        )

    /**
     * ApiClient discards error bodies on throw; for 409 we re-fetch the current envelope
     * so conflict policy can resolve without silently discarding learner input.
     */
    private suspend fun remapStateWriteError(
        error: ApiError.HttpStatus,
        courseCode: String,
        instanceId: String,
        accessToken: String,
        scope: String?,
    ): Nothing {
        when (error.code) {
            409 -> {
                val current = runCatching {
                    getState(courseCode, instanceId, accessToken, scope)
                }.getOrNull()
                if (current != null) throw RevisionConflictException(current)
                throw error
            }
            413 -> throw StateTooLargeException(0)
            422 -> throw SchemaInvalidException(error.apiMessage ?: "schema_invalid")
            else -> throw error
        }
    }
}
