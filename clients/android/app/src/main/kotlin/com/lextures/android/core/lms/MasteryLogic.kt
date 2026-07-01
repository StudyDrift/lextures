package com.lextures.android.core.lms

import com.lextures.android.R

enum class MasteryLevel {
    MASTERED,
    DEVELOPING,
    BEGINNING,
    AT_RISK,
    NOT_ASSESSED,
}

data class MasteryConceptRow(
    val id: String,
    val name: String,
    val score: Double?,
    val assessed: Boolean,
    val level: MasteryLevel,
)

object MasteryLogic {
    /** Matches web `masteryLabel()` thresholds (0.8 / 0.6 / 0.4). */
    fun level(score: Double?, assessed: Boolean): MasteryLevel {
        if (!assessed || score == null) return MasteryLevel.NOT_ASSESSED
        return when {
            score >= 0.8 -> MasteryLevel.MASTERED
            score >= 0.6 -> MasteryLevel.DEVELOPING
            score >= 0.4 -> MasteryLevel.BEGINNING
            else -> MasteryLevel.AT_RISK
        }
    }

    fun levelLabelRes(level: MasteryLevel): Int = when (level) {
        MasteryLevel.MASTERED -> R.string.mobile_mastery_levelMastered
        MasteryLevel.DEVELOPING -> R.string.mobile_mastery_levelDeveloping
        MasteryLevel.BEGINNING -> R.string.mobile_mastery_levelBeginning
        MasteryLevel.AT_RISK -> R.string.mobile_mastery_levelAtRisk
        MasteryLevel.NOT_ASSESSED -> R.string.mobile_mastery_levelNotAssessed
    }

    fun rows(row: StudentMasteryRow): List<MasteryConceptRow> {
        val cellsByConcept = row.cells.associateBy { it.conceptId }
        return row.concepts
            .map { concept ->
                val cell = cellsByConcept[concept.id]
                MasteryConceptRow(
                    id = concept.id,
                    name = concept.name,
                    score = cell?.masteryScore,
                    assessed = cell?.assessed ?: false,
                    level = level(cell?.masteryScore, cell?.assessed ?: false),
                )
            }
            .sortedWith(
                compareBy(
                    { it.assessed },
                    { it.score ?: 0.0 },
                ),
            )
    }

    data class Summary(val mastered: Int, val atRisk: Int, val total: Int)

    fun summary(rows: List<MasteryConceptRow>): Summary {
        val assessed = rows.filter { it.assessed }
        return Summary(
            mastered = assessed.count { it.level == MasteryLevel.MASTERED },
            atRisk = assessed.count { it.level == MasteryLevel.AT_RISK },
            total = rows.size,
        )
    }

    fun cacheKeyMastery(courseCode: String, enrollmentId: String): String =
        "mastery:$courseCode:$enrollmentId"

    fun cacheKeyMyReportCards(): String = "mastery:my-report-cards"

    fun releasedReportCards(cards: List<ReportCardSummary>): List<ReportCardSummary> =
        cards.filter { it.status == "released" }.sortedByDescending { it.gradingPeriod }
}
