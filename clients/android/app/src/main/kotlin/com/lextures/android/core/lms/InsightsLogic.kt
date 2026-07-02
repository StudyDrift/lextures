package com.lextures.android.core.lms

object InsightsLogic {
    const val JOURNAL_MAX_LENGTH = 280

    fun formatHours(hours: Double): String = when {
        hours < 0.1 -> "0"
        hours >= 10 -> "%.0f".format(hours)
        else -> "%.1f".format(hours)
    }

    fun hoursFromSeconds(seconds: Int): Double = seconds / 3600.0

    fun goalProgressPercent(progressHours: Float, goalHours: Float?): Int? {
        if (goalHours == null || goalHours <= 0f) return null
        val pct = ((progressHours / goalHours) * 100f).toInt()
        return pct.coerceIn(0, 100)
    }

    fun maxAllocationMinutes(rows: List<StudyTimeAllocationRow>): Double =
        maxOf(1.0, rows.maxOfOrNull { it.minutes } ?: 1.0)

    fun barWidthPercent(minutes: Double, maxMinutes: Double): Double {
        if (maxMinutes <= 0) return 0.0
        return ((minutes / maxMinutes) * 100.0).coerceIn(0.0, 100.0)
    }

    fun moduleCompletionPercent(snapshot: ModulesProgressSnapshot): Int {
        var total = 0
        var complete = 0
        for (module in snapshot.modules) {
            val items = module.items
            if (!items.isNullOrEmpty()) {
                total += items.size
                complete += items.count { it.complete }
            } else {
                total += 1
                if (module.complete) complete += 1
            }
        }
        if (total == 0) return 0
        return ((complete * 100) / total).coerceIn(0, 100)
    }

    fun journalEntryValid(text: String): Boolean {
        val trimmed = text.trim()
        return trimmed.isNotEmpty() && trimmed.length <= JOURNAL_MAX_LENGTH
    }

    fun latestCoachingTip(response: CoachingTipsResponse): CoachingTip? =
        response.latest ?: response.history.firstOrNull()
}