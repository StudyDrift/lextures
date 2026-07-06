package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class InstructorInsightsLogicTest {
    @Test
    fun enabledRequiresRolloutFlag() {
        val off = MobilePlatformFeatures(ffMobileInstructorInsights = false, atRiskAlertsEnabled = true)
        assertFalse(InstructorInsightsLogic.enabled(off))
    }

    @Test
    fun enabledWithAnyAnalyticsFlag() {
        assertTrue(InstructorInsightsLogic.enabled(MobilePlatformFeatures(atRiskAlertsEnabled = true)))
        assertTrue(InstructorInsightsLogic.enabled(MobilePlatformFeatures(instructorInsightsEnabled = true)))
        assertTrue(InstructorInsightsLogic.enabled(MobilePlatformFeatures(studentProgressEnabled = true)))
    }

    @Test
    fun shouldShowWorkspaceSectionStaffOnly() {
        val staff = CourseSummary(
            id = "1",
            courseCode = "demo",
            title = "Demo",
            viewerEnrollmentRoles = listOf("teacher"),
        )
        val student = CourseSummary(
            id = "2",
            courseCode = "demo",
            title = "Demo",
            viewerEnrollmentRoles = listOf("student"),
        )
        val features = MobilePlatformFeatures(atRiskAlertsEnabled = true)
        assertTrue(InstructorInsightsLogic.shouldShowWorkspaceSection(staff, features))
        assertFalse(InstructorInsightsLogic.shouldShowWorkspaceSection(student, features))
    }

    @Test
    fun sortAlertsByScoreDescending() {
        val alerts = listOf(
            AtRiskAlert(
                id = "1", enrollmentId = "e1", userId = "u1", displayName = "Zoe",
                score = 40f, status = "active", topFactor = "missing", topFactorLabel = "Missing",
                triggeredDate = "2026-01-01",
            ),
            AtRiskAlert(
                id = "2", enrollmentId = "e2", userId = "u2", displayName = "Alex",
                score = 90f, status = "active", topFactor = "inactive", topFactorLabel = "Inactive",
                triggeredDate = "2026-01-01",
            ),
        )
        val sorted = InstructorInsightsLogic.sortAlerts(alerts)
        assertEquals("2", sorted.first().id)
        assertEquals("1", sorted.last().id)
    }

    @Test
    fun severityThreshold() {
        assertEquals(InstructorInsightsLogic.AtRiskSeverity.Moderate, InstructorInsightsLogic.severity(79f))
        assertEquals(InstructorInsightsLogic.AtRiskSeverity.High, InstructorInsightsLogic.severity(80f))
    }

    @Test
    fun snapshotAggregation() {
        val snapshot = InstructorInsightsLogic.snapshot(
            atRiskCount = 2,
            ungradedCount = 5,
            workingWell = listOf(
                InstructorSignalItem(
                    itemId = "a", title = "A", kind = "assignment", completionRate = 0.8,
                    avgScore = 90.0, engagement = 10.0, compositeScore = 1.0, narrative = "Good",
                ),
            ),
            needsAttention = emptyList(),
        )
        assertEquals(2, snapshot.atRiskCount)
        assertEquals(5, snapshot.ungradedCount)
        assertEquals(1, snapshot.engagementHighlightCount)
    }
}
