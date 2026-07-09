package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.put
import java.net.URLEncoder

/** Account integrations helpers (M14.1). */
object AccountIntegrationsLogic {
    const val RBAC_MANAGE_PERMISSION = "global:app:rbac:manage"
    val defaultCreateScopes = listOf("mcp:connect", "courses:read")

    fun integrationsEnabled(features: MobilePlatformFeatures): Boolean =
        features.ffMobileSettingsIntegrations

    fun accessKeysEnabled(features: MobilePlatformFeatures): Boolean =
        features.ffApiTokens

    fun calendarSubscriptionsEnabled(features: MobilePlatformFeatures): Boolean =
        features.ffCalendarFeeds

    fun canManageServiceTokens(permissions: List<String>): Boolean =
        permissions.contains(RBAC_MANAGE_PERMISSION)

    fun shouldShowServiceTokensSection(
        permissions: List<String>,
        adminApiForbidden: Boolean,
    ): Boolean = !adminApiForbidden && canManageServiceTokens(permissions)

    fun isRevoked(revokedAt: String?): Boolean = !revokedAt.isNullOrEmpty()

    fun activeAccessKeys(tokens: List<AccessKeySummary>): List<AccessKeySummary> =
        tokens.filter { !isRevoked(it.revokedAt) && it.isServiceToken != true }

    fun revokedAccessKeys(tokens: List<AccessKeySummary>): List<AccessKeySummary> =
        tokens.filter { isRevoked(it.revokedAt) && it.isServiceToken != true }

    fun activeServiceTokens(tokens: List<AccessKeySummary>): List<AccessKeySummary> =
        tokens.filter { it.isServiceToken == true && !isRevoked(it.revokedAt) }

    fun resolveCalendarFeedURL(template: String, token: String): String {
        val encoded = URLEncoder.encode(token, "UTF-8")
        return template.replace("<token>", encoded)
    }

    fun resolvedPersonalFeedUrl(
        info: CalendarTokenInfo?,
        created: CalendarTokenCreated?,
    ): String? {
        created?.feedUrl?.takeIf { it.isNotBlank() }?.let { return it }
        val token = created?.token ?: return null
        val template = info?.personalFeedUrl ?: return null
        if (template.isBlank()) return null
        return resolveCalendarFeedURL(template, token)
    }

    fun resolvedCourseFeedUrl(template: String, token: String?): String? {
        if (token.isNullOrBlank()) return null
        return resolveCalendarFeedURL(template, token)
    }

    fun mcpConfigJson(base: JsonObject, token: String): String {
        val trimmed = token.trim()
        if (trimmed.isEmpty()) return base.toString()
        val servers = base["mcpServers"]?.jsonObject ?: return base.toString()
        val lextures = servers["lextures"]?.jsonObject ?: return base.toString()
        val env = lextures["env"]?.jsonObject ?: return base.toString()
        val updatedEnv = buildJsonObject {
            env.forEach { (key, value) -> put(key, value) }
            put("LEXTURES_API_TOKEN", JsonPrimitive(trimmed))
        }
        val updatedLextures = buildJsonObject {
            lextures.forEach { (key, value) ->
                put(key, if (key == "env") updatedEnv else value)
            }
        }
        val updatedServers = buildJsonObject {
            servers.forEach { (key, value) ->
                put(key, if (key == "lextures") updatedLextures else value)
            }
        }
        return buildJsonObject {
            base.forEach { (key, value) ->
                put(key, if (key == "mcpServers") updatedServers else value)
            }
        }.toString()
    }
}