package com.lextures.android.core.lms

import android.content.Context

/** Lightweight client-only enrollment management counters (MOB.4; no PII). */
object CoursePeopleObservability {
    private const val PREFS = "course_people_observability"

    fun recordAdded(
        context: Context,
        role: String,
        addedCount: Int,
        alreadyCount: Int,
        notFoundCount: Int,
    ) {
        if (addedCount > 0) {
            bump(context, "enrollment_added")
            prefs(context).edit()
                .putString("enrollment_added.last_role", role)
                .putInt("enrollment_added.last_count", addedCount)
                .apply()
        }
        if (alreadyCount > 0) bump(context, "enrollment_add_already")
        if (notFoundCount > 0) bump(context, "enrollment_add_not_found")
    }

    fun recordStateChanged(context: Context, role: String, state: String) {
        bump(context, "enrollment_state_changed")
        prefs(context).edit()
            .putString("enrollment_state_changed.last_role", role)
            .putString("enrollment_state_changed.last_state", state)
            .apply()
    }

    fun recordRemoved(context: Context, role: String) {
        bump(context, "enrollment_removed")
        prefs(context).edit().putString("enrollment_removed.last_role", role).apply()
    }

    private fun prefs(context: Context) =
        context.applicationContext.getSharedPreferences(PREFS, Context.MODE_PRIVATE)

    private fun bump(context: Context, key: String) {
        val p = prefs(context)
        p.edit().putInt(key, p.getInt(key, 0) + 1).apply()
    }
}
