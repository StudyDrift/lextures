package com.lextures.android.core.lms

/** In-memory-only checklist summary memo (FR-8, FR-11). Never writes to disk. */
object CourseChecklistSummaryStore {
    private val entries = mutableMapOf<String, CourseChecklistLogic.SummaryCacheEntry>()
    private val hiddenBy403 = mutableSetOf<String>()

    @Synchronized
    fun cached(courseCode: String): CourseChecklistSummary? {
        val entry = entries[courseCode] ?: return null
        if (!CourseChecklistLogic.isSummaryFresh(entry)) return null
        return entry.summary
    }

    @Synchronized
    fun outstandingEssential(courseCode: String): Int =
        cached(courseCode)?.outstandingEssential ?: 0

    @Synchronized
    fun isHiddenBy403(courseCode: String): Boolean = courseCode in hiddenBy403

    @Synchronized
    fun put(
        courseCode: String,
        summary: CourseChecklistSummary,
        catalogVersion: String? = null,
    ) {
        hiddenBy403.remove(courseCode)
        entries[courseCode] = CourseChecklistLogic.SummaryCacheEntry(
            summary = summary,
            catalogVersion = catalogVersion,
            fetchedAtMs = System.currentTimeMillis(),
        )
    }

    @Synchronized
    fun markForbidden(courseCode: String) {
        hiddenBy403.add(courseCode)
        entries.remove(courseCode)
    }

    @Synchronized
    fun invalidate(courseCode: String) {
        entries.remove(courseCode)
    }

    @Synchronized
    fun clearAll() {
        entries.clear()
        hiddenBy403.clear()
    }

    @Synchronized
    fun applyChecklist(checklist: CourseChecklist) {
        put(
            courseCode = checklist.courseCode,
            summary = checklist.summary,
            catalogVersion = checklist.catalogVersion,
        )
    }
}
