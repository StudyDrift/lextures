package com.lextures.android.core.lms

import com.lextures.android.R
import com.lextures.android.core.navigation.MobilePlatformFeatures

/** Staff course health, at-risk, and engagement helpers (M11.3). */
object InstructorInsightsLogic {
    enum class AtRiskSeverity {
        High,
        Moderate,
        ;

        val labelRes: Int
            get() = when (this) {
                High -> R.string.mobile_instructorInsights_severity_high
                Moderate -> R.string.mobile_instructorInsights_severity_moderate
            }
    }

    fun enabled(features: MobilePlatformFeatures): Boolean {
        if (!features.ffMobileInstructorInsights) return false
        return features.atRiskAlertsEnabled ||
            features.instructorInsightsEnabled ||
            features.studentProgressEnabled
    }

    fun shouldShowWorkspaceSection(course: CourseSummary, features: MobilePlatformFeatures): Boolean =
        course.viewerIsStaff && enabled(features)

    fun severity(score: Float): AtRiskSeverity = if (score >= 80f) AtRiskSeverity.High else AtRiskSeverity.Moderate

    fun sortAlerts(alerts: List<AtRiskAlert>): List<AtRiskAlert> =
        alerts.sortedWith(compareByDescending<AtRiskAlert> { it.score }.thenBy { it.displayName.lowercase() })

    fun snapshot(
        atRiskCount: Int,
        ungradedCount: Int,
        workingWell: List<InstructorSignalItem>,
        needsAttention: List<InstructorSignalItem>,
    ): CourseHealthSnapshot = CourseHealthSnapshot(
        atRiskCount = atRiskCount,
        ungradedCount = ungradedCount,
        engagementHighlightCount = workingWell.size + needsAttention.size,
    )

    fun webReportsPath(courseCode: String): String = "/courses/$courseCode/at-risk"

    fun webWhatsWorkingPath(courseCode: String): String = "/courses/$courseCode/whats-working"

    fun completionPercentText(rate: Double): String = "${rate.times(100).toInt()}%"

    fun optionalPercentText(value: Double?): String? = value?.let { String.format("%.1f%%", it) }
}
