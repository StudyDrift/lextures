package com.lextures.android.core.lms

import android.content.Context
import java.io.File

/** Bundled CC.8 native target table (anchor → Android destination). */
object CourseChecklistTargetTable {
    @Volatile
    private var cached: Map<String, String>? = null

    fun shared(context: Context? = null): Map<String, String> {
        cached?.let { return it }
        val map = load(context)
        cached = map
        return map
    }

    fun load(context: Context? = null): Map<String, String> {
        if (context != null) {
            runCatching {
                context.assets.open("checklist-targets.json").bufferedReader().use { it.readText() }
            }.getOrNull()?.let { return CourseChecklistLogic.loadTargetTable(it) }
        }
        // Tests / fallback: walk up to monorepo packages path.
        var dir = File(System.getProperty("user.dir") ?: ".")
        repeat(10) {
            val candidates = listOf(
                File(dir, "clients/packages/checklist-targets.json"),
                File(dir, "packages/checklist-targets.json"),
                File(dir, "app/src/main/assets/checklist-targets.json"),
            )
            for (f in candidates) {
                if (f.isFile) {
                    return CourseChecklistLogic.loadTargetTable(f.readText())
                }
            }
            dir = dir.parentFile ?: return emptyMap()
        }
        return emptyMap()
    }

    fun resetCacheForTests() {
        cached = null
    }
}
