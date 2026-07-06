import XCTest
@testable import Lextures

final class InstructorInsightsLogicTests: XCTestCase {
    func testEnabledRequiresRolloutFlag() {
        var features = MobilePlatformFeatures()
        features.ffMobileInstructorInsights = false
        features.atRiskAlertsEnabled = true
        XCTAssertFalse(InstructorInsightsLogic.enabled(features))
    }

    func testEnabledWithAnyAnalyticsFlag() {
        var atRisk = MobilePlatformFeatures()
        atRisk.atRiskAlertsEnabled = true
        XCTAssertTrue(InstructorInsightsLogic.enabled(atRisk))

        var insights = MobilePlatformFeatures()
        insights.instructorInsightsEnabled = true
        XCTAssertTrue(InstructorInsightsLogic.enabled(insights))

        var progress = MobilePlatformFeatures()
        progress.studentProgressEnabled = true
        XCTAssertTrue(InstructorInsightsLogic.enabled(progress))
    }

    func testShouldShowWorkspaceSectionStaffOnly() {
        let staff = CourseSummary(
            id: "1", courseCode: "demo", title: "Demo", description: "",
            viewerEnrollmentRoles: ["teacher"]
        )
        let student = CourseSummary(
            id: "2", courseCode: "demo", title: "Demo", description: "",
            viewerEnrollmentRoles: ["student"]
        )
        var features = MobilePlatformFeatures(atRiskAlertsEnabled: true)
        XCTAssertTrue(InstructorInsightsLogic.shouldShowWorkspaceSection(course: staff, features: features))
        XCTAssertFalse(InstructorInsightsLogic.shouldShowWorkspaceSection(course: student, features: features))
    }

    func testSortAlertsByScoreDescending() {
        let alerts = [
            AtRiskAlert(
                id: "1", enrollmentId: "e1", userId: "u1", displayName: "Zoe",
                score: 40, status: "active", topFactor: "missing", topFactorLabel: "Missing",
                triggeredDate: "2026-01-01"
            ),
            AtRiskAlert(
                id: "2", enrollmentId: "e2", userId: "u2", displayName: "Alex",
                score: 90, status: "active", topFactor: "inactive", topFactorLabel: "Inactive",
                triggeredDate: "2026-01-01"
            ),
        ]
        let sorted = InstructorInsightsLogic.sortAlerts(alerts)
        XCTAssertEqual(sorted.first?.id, "2")
        XCTAssertEqual(sorted.last?.id, "1")
    }

    func testSeverityThreshold() {
        XCTAssertEqual(InstructorInsightsLogic.severity(score: 79), .moderate)
        XCTAssertEqual(InstructorInsightsLogic.severity(score: 80), .high)
    }

    func testSnapshotAggregation() {
        let snapshot = InstructorInsightsLogic.snapshot(
            atRiskCount: 2,
            ungradedCount: 5,
            workingWell: [InstructorSignalItem(
                itemId: "a", title: "A", kind: "assignment", completionRate: 0.8,
                avgScore: 90, engagement: 10, difficulty: nil, compositeScore: 1, narrative: "Good"
            )],
            needsAttention: []
        )
        XCTAssertEqual(snapshot.atRiskCount, 2)
        XCTAssertEqual(snapshot.ungradedCount, 5)
        XCTAssertEqual(snapshot.engagementHighlightCount, 1)
    }
}
