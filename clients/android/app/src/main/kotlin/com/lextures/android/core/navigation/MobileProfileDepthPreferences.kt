package com.lextures.android.core.navigation

import android.content.Context

/** Client rollout flag for M1.5 profile depth sections. */
object MobileProfileDepthPreferences {
    private const val KEY = "mobile_profile_depth"

    fun isEnabled(context: Context): Boolean {
        val prefs = context.getSharedPreferences("lextures_mobile_ia", Context.MODE_PRIVATE)
        return if (!prefs.contains(KEY)) true else prefs.getBoolean(KEY, false)
    }

    fun setEnabled(context: Context, enabled: Boolean) {
        context.getSharedPreferences("lextures_mobile_ia", Context.MODE_PRIVATE)
            .edit()
            .putBoolean(KEY, enabled)
            .apply()
    }
}