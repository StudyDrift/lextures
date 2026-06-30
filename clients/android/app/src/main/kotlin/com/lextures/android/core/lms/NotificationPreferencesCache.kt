package com.lextures.android.core.lms

import android.content.Context
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json

/** Local cache of notification preferences for offline reads and client-side push gating. */
object NotificationPreferencesCache {
    private const val storageName = "notification_preferences_cache"
    private val json = Json { ignoreUnknownKeys = true }

    fun save(context: Context, ownerKey: String, preferences: List<NotificationPreference>) {
        val prefs = context.applicationContext.getSharedPreferences(storageName, Context.MODE_PRIVATE)
        prefs.edit().putString(key(ownerKey), json.encodeToString(preferences)).apply()
    }

    fun load(context: Context, ownerKey: String): List<NotificationPreference> {
        val prefs = context.applicationContext.getSharedPreferences(storageName, Context.MODE_PRIVATE)
        val raw = prefs.getString(key(ownerKey), null) ?: return emptyList()
        return runCatching { json.decodeFromString<List<NotificationPreference>>(raw) }.getOrDefault(emptyList())
    }

    fun clear(context: Context, ownerKey: String) {
        val prefs = context.applicationContext.getSharedPreferences(storageName, Context.MODE_PRIVATE)
        prefs.edit().remove(key(ownerKey)).apply()
    }

    private fun key(ownerKey: String): String = "prefs.$ownerKey"
}
