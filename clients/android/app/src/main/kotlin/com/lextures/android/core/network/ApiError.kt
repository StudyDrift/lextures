package com.lextures.android.core.network

import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.jsonPrimitive

sealed class ApiError : Exception() {
    data object InvalidResponse : ApiError() {
        private fun readResolve(): Any = InvalidResponse
        override val message: String = "Unexpected response from the server."
    }

    class HttpStatus(val code: Int, val apiMessage: String?) : ApiError() {
        override val message: String =
            if (!apiMessage.isNullOrBlank()) apiMessage else "Request failed (HTTP $code)."
    }

    class Transport(cause: Throwable) : ApiError() {
        override val message: String = cause.localizedMessage ?: cause.toString()
    }

    class Decoding(cause: Throwable) : ApiError() {
        override val message: String = cause.localizedMessage ?: cause.toString()
    }
}

/** Mirrors web `readApiErrorMessage` for common API error shapes. */
fun parseApiErrorMessage(body: String): String? {
    if (body.isBlank()) return null
    return runCatching {
        val json = Json.parseToJsonElement(body)
        if (json !is JsonObject) return@runCatching null
        listOf("message", "error", "detail").firstNotNullOfOrNull { key ->
            json[key]?.jsonPrimitive?.content?.takeIf { it.isNotBlank() }
        }
    }.getOrNull()
}
