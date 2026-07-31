package com.lextures.android.features.contenttools

import android.content.Context
import android.content.SharedPreferences

/**
 * Local free-text draft store for CT.M6 composers.
 * Keyed separately from tool state so a draft is never mistaken for saved work (FR-2).
 */
class ContentToolDraftStore private constructor(
    private val prefs: SharedPreferences?,
    private val memory: MutableMap<String, String>?,
) {
    fun load(key: String): String {
        memory?.let { return it[key].orEmpty() }
        return prefs?.getString(key, "").orEmpty()
    }

    fun save(key: String, text: String) {
        if (text.trim().isEmpty()) {
            clear(key)
            return
        }
        memory?.let {
            it[key] = text
            return
        }
        prefs?.edit()?.putString(key, text)?.apply()
    }

    fun clear(key: String) {
        memory?.let {
            it.remove(key)
            return
        }
        prefs?.edit()?.remove(key)?.apply()
    }

    companion object {
        private const val PREFS = "content_tool_drafts"

        fun create(context: Context): ContentToolDraftStore =
            ContentToolDraftStore(
                prefs = context.applicationContext.getSharedPreferences(PREFS, Context.MODE_PRIVATE),
                memory = null,
            )

        /** Unit-test helper (no Android context). */
        fun inMemory(): ContentToolDraftStore =
            ContentToolDraftStore(prefs = null, memory = mutableMapOf())
    }
}
