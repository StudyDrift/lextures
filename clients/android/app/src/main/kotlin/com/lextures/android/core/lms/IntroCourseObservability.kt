package com.lextures.android.core.lms

import android.content.Context

/** Lightweight client-only intro course funnel counters (IC07; no PII). */
object IntroCourseObservability {
    private const val PREFS = "intro_course_observability"

    fun recordCardView(context: Context) = bump(context, "intro_course.card_view")

    fun recordCtaClick(context: Context) = bump(context, "intro_course.cta_click")

    fun recordCelebrationView(context: Context) = bump(context, "intro_course.completed_celebration_view")

    private fun bump(context: Context, key: String) {
        val prefs = context.applicationContext.getSharedPreferences(PREFS, Context.MODE_PRIVATE)
        prefs.edit().putInt(key, prefs.getInt(key, 0) + 1).apply()
    }
}