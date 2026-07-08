package com.lextures.android.core.lms

import android.content.Context
import com.lextures.android.R
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalePreferences
import com.lextures.android.core.navigation.MobilePlatformFeatures
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.doubleOrNull
import kotlinx.serialization.json.intOrNull
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive

/** Learner profile helpers (LP10). */
object LearnerProfileLogic {
    val facetPriority = listOf(
        "study_rhythm",
        "content_modality",
        "strengths_growth",
        "interests",
        "learning_approach",
    )

    fun learnerProfileEnabled(features: MobilePlatformFeatures): Boolean =
        features.learnerProfileEnabled && features.ffMobileLearnerProfile

    fun cacheKeyProfile(): String = "learner-profile:summary"

    fun cacheKeyFacetEvidence(facetKey: String): String = "learner-profile:evidence:$facetKey"

    fun sortFacets(facets: List<LearnerProfileFacetSummary>): List<LearnerProfileFacetSummary> {
        val order = facetPriority.withIndex().associate { it.value to it.index }
        return facets.sortedBy { order[it.facetKey] ?: 999 }
    }

    fun isPaused(profile: LearnerProfile): Boolean = profile.status == "paused"

    fun showEmptyState(profile: LearnerProfile): Boolean {
        if (profile.status == "insufficient_data") return true
        return profile.facets.isNotEmpty() && profile.facets.all { it.state == "insufficient_data" }
    }

    enum class ConfidenceLevel { High, Medium, Low }

    fun confidenceLevel(score: Double): ConfidenceLevel = when {
        score >= 0.75 -> ConfidenceLevel.High
        score >= 0.45 -> ConfidenceLevel.Medium
        else -> ConfidenceLevel.Low
    }

    fun confidenceLabelRes(score: Double): Int = when (confidenceLevel(score)) {
        ConfidenceLevel.High -> R.string.mobile_learnerProfile_confidence_high
        ConfidenceLevel.Medium -> R.string.mobile_learnerProfile_confidence_medium
        ConfidenceLevel.Low -> R.string.mobile_learnerProfile_confidence_low
    }

    fun facetTitleRes(facetKey: String): Int = when (facetKey) {
        "study_rhythm" -> R.string.mobile_learnerProfile_facet_studyRhythm_title
        "content_modality" -> R.string.mobile_learnerProfile_facet_contentModality_title
        "strengths_growth" -> R.string.mobile_learnerProfile_facet_strengthsGrowth_title
        "interests" -> R.string.mobile_learnerProfile_facet_interests_title
        "learning_approach" -> R.string.mobile_learnerProfile_facet_learningApproach_title
        else -> R.string.mobile_learnerProfile_facet_generic_title
    }

    fun facetDescriptionRes(facetKey: String): Int = when (facetKey) {
        "study_rhythm" -> R.string.mobile_learnerProfile_facet_studyRhythm_description
        "content_modality" -> R.string.mobile_learnerProfile_facet_contentModality_description
        "strengths_growth" -> R.string.mobile_learnerProfile_facet_strengthsGrowth_description
        "interests" -> R.string.mobile_learnerProfile_facet_interests_description
        "learning_approach" -> R.string.mobile_learnerProfile_facet_learningApproach_description
        else -> R.string.mobile_learnerProfile_facet_generic_description
    }

    fun insightLabelRes(insightKey: String): Int = when (insightKey) {
        "peak_study_window" -> R.string.mobile_learnerProfile_insight_peakStudyWindow
        "study_consistency" -> R.string.mobile_learnerProfile_insight_studyConsistency
        "study_streak" -> R.string.mobile_learnerProfile_insight_studyStreak
        "session_shape" -> R.string.mobile_learnerProfile_insight_sessionShape
        "modality_affinity" -> R.string.mobile_learnerProfile_insight_modalityAffinity
        "complexity_comfort" -> R.string.mobile_learnerProfile_insight_complexityComfort
        "content_pacing" -> R.string.mobile_learnerProfile_insight_contentPacing
        "top_strengths" -> R.string.mobile_learnerProfile_insight_topStrengths
        "growth_areas" -> R.string.mobile_learnerProfile_insight_growthAreas
        "needs_review" -> R.string.mobile_learnerProfile_insight_needsReview
        "persistence" -> R.string.mobile_learnerProfile_insight_persistenceLabel
        "help_seeking" -> R.string.mobile_learnerProfile_insight_helpSeekingLabel
        "consolidation" -> R.string.mobile_learnerProfile_insight_consolidationLabel
        else -> if (insightKey.startsWith("topic_")) {
            R.string.mobile_learnerProfile_insight_topic
        } else {
            R.string.mobile_learnerProfile_insight_generic
        }
    }

    fun totalObservationCount(evidence: List<LearnerProfileEvidenceRow>): Int =
        evidence.sumOf { it.observationCount }

    fun uniqueCourseCount(evidence: List<LearnerProfileEvidenceRow>): Int =
        evidence.mapNotNull { it.courseId?.takeIf { id -> id.isNotBlank() } }.toSet().size

    fun formatInsightValue(
        context: Context,
        prefs: LocalePreferences,
        insight: LearnerProfileInsight,
        facetKey: String,
    ): String {
        val value = insight.value
        return when (insight.insightKey) {
            "peak_study_window" -> {
                val top = value["peakWindows"]?.jsonArray?.firstOrNull()?.jsonObject
                val dow = top?.get("dow")?.jsonPrimitive?.content
                val hour = top?.get("hourBucket")?.jsonPrimitive?.content
                val share = top?.get("share")?.jsonPrimitive?.doubleOrNull
                if (dow.isNullOrBlank() || hour.isNullOrBlank() || share == null) {
                    L.text(context, prefs, R.string.mobile_learnerProfile_insight_peakUnknown)
                } else {
                    L.format(context, prefs, R.string.mobile_learnerProfile_insight_peak, dow, hour, (share * 100).toInt())
                }
            }
            "study_consistency" -> L.format(
                context,
                prefs,
                R.string.mobile_learnerProfile_insight_consistency,
                jsonInt(value["consistencyScore"], percent = true),
                jsonDoubleString(value["activeDaysPerWeek"], 1) ?: "0",
            )
            "study_streak" -> L.format(
                context,
                prefs,
                R.string.mobile_learnerProfile_insight_streak,
                jsonInt(value["currentStreakDays"]),
                jsonInt(value["longestStreakDays"]),
            )
            "session_shape" -> L.format(
                context,
                prefs,
                R.string.mobile_learnerProfile_insight_session,
                jsonInt(value["medianSessionMin"]),
                jsonDoubleString(value["sessionsPerActiveWeek"], 1) ?: "0",
            )
            "modality_affinity" -> {
                val affinity = value["modalityAffinity"]?.jsonObject
                    ?: return L.text(context, prefs, R.string.mobile_learnerProfile_insight_genericUnknown)
                val top = affinity.entries.maxByOrNull { jsonDouble(it.value) }
                    ?: return L.text(context, prefs, R.string.mobile_learnerProfile_insight_genericUnknown)
                L.format(
                    context,
                    prefs,
                    R.string.mobile_learnerProfile_insight_modalityTop,
                    top.key,
                    (jsonDouble(top.value) * 100).toInt(),
                )
            }
            "complexity_comfort" -> {
                val band = value["complexityComfort"]?.jsonObject
                val low = band?.get("low")?.jsonPrimitive?.content
                val high = band?.get("high")?.jsonPrimitive?.content
                if (low.isNullOrBlank() || high.isNullOrBlank()) {
                    L.text(context, prefs, R.string.mobile_learnerProfile_insight_genericUnknown)
                } else {
                    L.format(context, prefs, R.string.mobile_learnerProfile_insight_comfort, low, high)
                }
            }
            "content_pacing" -> L.format(
                context,
                prefs,
                R.string.mobile_learnerProfile_insight_pacing,
                value["pacing"]?.jsonPrimitive?.content ?: "unknown",
            )
            "top_strengths" -> conceptList(context, prefs, value["strengths"], listOf("concept"), R.string.mobile_learnerProfile_insight_strengthsList)
            "growth_areas" -> conceptList(context, prefs, value["growth"], listOf("concept", "misconception"), R.string.mobile_learnerProfile_insight_growthList)
            "needs_review" -> conceptList(context, prefs, value["needsReview"], listOf("concept"), R.string.mobile_learnerProfile_insight_reviewList)
            "persistence" -> {
                val productive = value["productive"]?.jsonPrimitive?.content == "true"
                L.format(
                    context,
                    prefs,
                    R.string.mobile_learnerProfile_insight_persistence,
                    value["level"]?.jsonPrimitive?.content ?: "unknown",
                    if (productive) L.text(context, prefs, R.string.mobile_learnerProfile_insight_productiveRetakes) else "",
                )
            }
            "help_seeking" -> L.format(
                context,
                prefs,
                R.string.mobile_learnerProfile_insight_helpSeeking,
                value["style"]?.jsonPrimitive?.content ?: "unknown",
                jsonDoubleString(value["hintsPerAttempt"], 1) ?: "0",
            )
            "consolidation" -> L.format(
                context,
                prefs,
                R.string.mobile_learnerProfile_insight_consolidation,
                value["level"]?.jsonPrimitive?.content ?: "unknown",
                jsonInt(value["notebookActions"]),
            )
            else -> if (insight.insightKey.startsWith("topic_")) {
                val topic = value["topic"]?.jsonPrimitive?.content
                if (topic.isNullOrBlank()) {
                    L.text(context, prefs, R.string.mobile_learnerProfile_insight_genericUnknown)
                } else {
                    L.format(context, prefs, R.string.mobile_learnerProfile_insight_interestTop, topic, jsonInt(value["affinity"], percent = true))
                }
            } else if (insight.label.isNotBlank()) {
                insight.label
            } else {
                L.text(context, prefs, R.string.mobile_learnerProfile_insight_genericUnknown)
            }
        }
    }

    fun rhythmChartCaption(context: Context, prefs: LocalePreferences, summary: JsonObject): String? {
        val windows = summary["peakWindows"]?.jsonArray ?: return null
        if (windows.isEmpty()) return null
        val lines = mutableListOf(L.text(context, prefs, R.string.mobile_learnerProfile_chart_rhythmCaption))
        windows.take(3).forEach { window ->
            val row = window.jsonObject
            val dow = row["dow"]?.jsonPrimitive?.content ?: return@forEach
            val hour = row["hourBucket"]?.jsonPrimitive?.content ?: return@forEach
            val share = row["share"]?.jsonPrimitive?.doubleOrNull ?: return@forEach
            lines.add(L.format(context, prefs, R.string.mobile_learnerProfile_chart_rhythmRow, dow, hour, (share * 100).toInt()))
        }
        return lines.joinToString("\n")
    }

    fun modalityChartCaption(context: Context, prefs: LocalePreferences, summary: JsonObject): String? {
        val affinity = summary["modalityAffinity"]?.jsonObject ?: return null
        if (affinity.isEmpty()) return null
        val lines = mutableListOf(L.text(context, prefs, R.string.mobile_learnerProfile_chart_modalityCaption))
        affinity.entries
            .sortedByDescending { jsonDouble(it.value) }
            .take(4)
            .forEach { (modality, share) ->
                lines.add(L.format(context, prefs, R.string.mobile_learnerProfile_chart_modalityRow, modality, (jsonDouble(share) * 100).toInt()))
            }
        return lines.joinToString("\n")
    }

    private fun conceptList(
        context: Context,
        prefs: LocalePreferences,
        raw: JsonElement?,
        keys: List<String>,
        formatRes: Int,
    ): String {
        val items = raw?.jsonArray ?: return L.text(context, prefs, R.string.mobile_learnerProfile_insight_genericUnknown)
        val names = items.take(3).mapNotNull { item ->
            val obj = item.jsonObject
            keys.firstNotNullOfOrNull { key -> obj[key]?.jsonPrimitive?.content?.takeIf { it.isNotBlank() } }
        }
        if (names.isEmpty()) return L.text(context, prefs, R.string.mobile_learnerProfile_insight_genericUnknown)
        return L.format(context, prefs, formatRes, names.joinToString(", "))
    }

    private fun jsonDouble(element: JsonElement): Double =
        element.jsonPrimitive.doubleOrNull
            ?: element.jsonPrimitive.content.toDoubleOrNull()
            ?: 0.0

    private fun jsonInt(element: JsonElement?, percent: Boolean = false): Int {
        val number = element?.let(::jsonDouble) ?: 0.0
        return (if (percent) number * 100 else number).toInt()
    }

    private fun jsonDoubleString(element: JsonElement?, digits: Int): String? {
        element ?: return null
        return "%.${digits}f".format(jsonDouble(element))
    }
}