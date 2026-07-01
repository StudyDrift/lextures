package com.lextures.android.features.assignments

import android.content.Context

/** Persists unsubmitted assignment text locally (M5.1). */
class AssignmentDraftStore(context: Context) {
    private val prefs = context.applicationContext.getSharedPreferences(PREFS, Context.MODE_PRIVATE)

    fun load(key: String): String = prefs.getString(key, "").orEmpty()

    fun save(key: String, text: String) {
        val trimmed = text.trim()
        if (trimmed.isEmpty()) {
            prefs.edit().remove(key).apply()
        } else {
            prefs.edit().putString(key, text).apply()
        }
    }

    fun clear(key: String) {
        prefs.edit().remove(key).apply()
    }

    companion object {
        private const val PREFS = "assignment_drafts"
    }
}
