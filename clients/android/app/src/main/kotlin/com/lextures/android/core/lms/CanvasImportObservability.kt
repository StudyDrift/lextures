package com.lextures.android.core.lms

import android.content.Context

/** Client-only Canvas import funnel counters (MOB.2). Never records the Canvas token. */
object CanvasImportObservability {
    private const val PREFS = "canvas_import_observability"

    fun recordListed(context: Context, courseCount: Int) {
        bump(context, "canvas_import_listed")
        prefs(context).edit().putInt("canvas_import_listed.last_count", courseCount).apply()
    }

    fun recordStarted(context: Context, include: CanvasImportLogic.Include) {
        bump(context, "canvas_import_started")
        prefs(context).edit()
            .putString("canvas_import_started.categories", include.enabledCategoryCounts().toString())
            .apply()
    }

    fun recordProgress(context: Context) {
        bump(context, "canvas_import_progress")
    }

    fun recordSucceeded(context: Context, include: CanvasImportLogic.Include) {
        bump(context, "canvas_import_succeeded")
        prefs(context).edit()
            .putString("canvas_import_succeeded.categories", include.enabledCategoryCounts().toString())
            .apply()
    }

    fun recordFailed(context: Context) {
        bump(context, "canvas_import_failed")
    }

    fun recordCancelled(context: Context) {
        bump(context, "canvas_import_cancelled")
    }

    private fun prefs(context: Context) =
        context.applicationContext.getSharedPreferences(PREFS, Context.MODE_PRIVATE)

    private fun bump(context: Context, key: String) {
        val p = prefs(context)
        p.edit().putInt(key, p.getInt(key, 0) + 1).apply()
    }
}
