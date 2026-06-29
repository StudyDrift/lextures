package com.lextures.android.core.auth

import java.net.URI
import java.net.URLDecoder

sealed class SsoProvider {
    data class Saml(val idpId: String) : SsoProvider()
    data class Oidc(val path: String, val label: String) : SsoProvider()
}

data class AuthCallbackPayload(
    val accessToken: String? = null,
    val refreshToken: String? = null,
    val expiresIn: Int? = null,
    val requiresMfa: Boolean = false,
    val mfaPendingToken: String? = null,
    val mfaSetupRequired: Boolean = false,
    val magicLinkToken: String? = null,
) {
    fun asTokenResponse(): AuthTokenResponse = AuthTokenResponse(
        accessToken = accessToken,
        refreshToken = refreshToken,
        expiresIn = expiresIn,
        requiresMfa = requiresMfa.takeIf { it },
        mfaPendingToken = mfaPendingToken,
        mfaSetupRequired = mfaSetupRequired.takeIf { it },
        user = null,
    )
}

object AuthCallbackParser {
    fun parse(raw: String?): AuthCallbackPayload? {
        val trimmed = raw?.trim().orEmpty()
        if (trimmed.isEmpty()) return null
        return parseMagicLink(trimmed) ?: parseAuthCallback(trimmed)
    }

    fun parseMagicLink(value: String): AuthCallbackPayload? {
        val path = extractPath(value) ?: return null
        val segments = path.trim('/').split('/').filter { it.isNotEmpty() }
        if (segments.size < 2 || segments[0].lowercase() != "login" || segments[1].lowercase() != "magic-link") {
            return null
        }
        val token = queryValue("token", value)?.trim().orEmpty()
        if (token.isEmpty()) return null
        return AuthCallbackPayload(magicLinkToken = token)
    }

    fun parseAuthCallback(value: String): AuthCallbackPayload? {
        val uri = runCatching { URI(value) }.getOrNull() ?: return null
        val isCallback = uri.scheme.equals(AuthConstants.CALLBACK_SCHEME, ignoreCase = true) &&
            uri.host.equals(AuthConstants.CALLBACK_HOST, ignoreCase = true) &&
            normalizePath(uri.path) == AuthConstants.CALLBACK_PATH
        if (!isCallback) return null
        return payloadFromUri(uri)
    }

    fun payloadFromUri(uri: URI): AuthCallbackPayload {
        val params = mutableMapOf<String, String>()
        parseQuery(uri.rawQuery).forEach { (key, value) -> params[key] = value }
        if (!uri.rawFragment.isNullOrBlank()) {
            parseQuery(uri.rawFragment).forEach { (key, value) -> params[key] = value }
        }
        return AuthCallbackPayload(
            accessToken = params["access_token"],
            refreshToken = params["refresh_token"],
            expiresIn = params["expires_in"]?.toIntOrNull(),
            requiresMfa = params["requires_mfa"] == "1" || params["requires_mfa"] == "true",
            mfaPendingToken = params["mfa_pending_token"],
            mfaSetupRequired = params["mfa_setup_required"] == "1" || params["mfa_setup_required"] == "true",
        )
    }

    private fun normalizePath(path: String?): String {
        if (path.isNullOrBlank()) return ""
        return if (path.startsWith("/")) path else "/$path"
    }

    private fun parseQuery(raw: String?): List<Pair<String, String>> {
        if (raw.isNullOrBlank()) return emptyList()
        return raw.split('&').mapNotNull { pair ->
            val parts = pair.split('=', limit = 2)
            if (parts.isEmpty()) return@mapNotNull null
            val key = parts[0]
            val value = if (parts.size == 2) URLDecoder.decode(parts[1], "UTF-8") else ""
            key to value
        }
    }

    private fun extractPath(value: String): String? {
        if (value.startsWith("lextures://")) {
            val pathPart = value.removePrefix("lextures://").substringBefore('?').substringBefore('#')
            return normalizePath(pathPart)
        }
        if (value.startsWith("/")) return value.substringBefore('?').substringBefore('#')
        if (value.startsWith("http://") || value.startsWith("https://")) {
            val uri = runCatching { URI(value) }.getOrNull() ?: return null
            val host = uri.host?.lowercase().orEmpty()
            if (host == "lextures.com" || host.endsWith(".lextures.com") || host == "localhost") {
                return normalizePath(uri.path)
            }
        }
        return null
    }

    private fun queryValue(name: String, value: String): String? {
        val uri = runCatching { URI(value) }.getOrNull() ?: return null
        return parseQuery(uri.rawQuery).firstOrNull { it.first == name }?.second
    }
}
