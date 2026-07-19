package com.lextures.android.core.lms

import android.content.Context

/** Lightweight client-only course-create funnel counters (MOB.1; no PII). */
object CourseCreateObservability {
    private const val PREFS = "course_create_observability"

    fun recordStarted(context: Context, mode: String, templateId: String) {
        bump(context, "course_create_started")
        prefs(context).edit()
            .putString("course_create_started.last_mode", mode)
            .putString("course_create_started.last_template", templateId)
            .apply()
    }

    fun recordStepCompleted(context: Context, step: Int) {
        bump(context, "course_create_step_completed")
        prefs(context).edit().putInt("course_create_step_completed.last_step", step).apply()
    }

    fun recordFinished(context: Context, mode: String, templateId: String) {
        bump(context, "course_create_finished")
        prefs(context).edit()
            .putString("course_create_finished.last_mode", mode)
            .putString("course_create_finished.last_template", templateId)
            .apply()
    }

    private fun prefs(context: Context) =
        context.applicationContext.getSharedPreferences(PREFS, Context.MODE_PRIVATE)

    private fun bump(context: Context, key: String) {
        val p = prefs(context)
        p.edit().putInt(key, p.getInt(key, 0) + 1).apply()
    }
}
