package com.lextures.android.core.navigation

import android.content.Context

/** Client-side rollout flag for the M0.5 information-architecture redesign. */
object MobileIaPreferences {
    private const val PREFS = "mobile_ia_prefs"
    private const val KEY_REDESIGN = "mobile_ia_redesign"
    private const val KEY_UNIVERSAL_SEARCH = "mobile_universal_search"
    private const val KEY_ROLE_CONTEXT = "mobile_ia_role_context"

    fun isRedesignEnabled(context: Context): Boolean =
        context.applicationContext
            .getSharedPreferences(PREFS, Context.MODE_PRIVATE)
            .getBoolean(KEY_REDESIGN, false)

    fun setRedesignEnabled(context: Context, enabled: Boolean) {
        context.applicationContext
            .getSharedPreferences(PREFS, Context.MODE_PRIVATE)
            .edit()
            .putBoolean(KEY_REDESIGN, enabled)
            .apply()
    }

    fun isUniversalSearchEnabled(context: Context): Boolean =
        context.applicationContext
            .getSharedPreferences(PREFS, Context.MODE_PRIVATE)
            .getBoolean(KEY_UNIVERSAL_SEARCH, false)

    fun setUniversalSearchEnabled(context: Context, enabled: Boolean) {
        context.applicationContext
            .getSharedPreferences(PREFS, Context.MODE_PRIVATE)
            .edit()
            .putBoolean(KEY_UNIVERSAL_SEARCH, enabled)
            .apply()
    }

    fun loadRoleContext(context: Context): MobileRoleContext? {
        val raw = context.applicationContext
            .getSharedPreferences(PREFS, Context.MODE_PRIVATE)
            .getString(KEY_ROLE_CONTEXT, null)
            ?: return null
        return runCatching { MobileRoleContext.valueOf(raw) }.getOrNull()
    }

    fun saveRoleContext(context: Context, roleContext: MobileRoleContext) {
        context.applicationContext
            .getSharedPreferences(PREFS, Context.MODE_PRIVATE)
            .edit()
            .putString(KEY_ROLE_CONTEXT, roleContext.name)
            .apply()
    }

    private const val KEY_SELECTED_CHILD = "mobile_parent_selected_child"

    fun loadSelectedChildId(context: Context): String? =
        context.applicationContext
            .getSharedPreferences(PREFS, Context.MODE_PRIVATE)
            .getString(KEY_SELECTED_CHILD, null)

    fun saveSelectedChildId(context: Context, studentId: String) {
        context.applicationContext
            .getSharedPreferences(PREFS, Context.MODE_PRIVATE)
            .edit()
            .putString(KEY_SELECTED_CHILD, studentId)
            .apply()
    }
}