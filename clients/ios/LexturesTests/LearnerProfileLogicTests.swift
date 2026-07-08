import XCTest
@testable import Lextures

final class LearnerProfileLogicTests: XCTestCase {
    func testLearnerProfileEnabledRequiresBothFlags() {
        var features = MobilePlatformFeatures()
        XCTAssertFalse(LearnerProfileLogic.learnerProfileEnabled(features))
        features.learnerProfileEnabled = true
        XCTAssertTrue(LearnerProfileLogic.learnerProfileEnabled(features))
        features.ffMobileLearnerProfile = false
        XCTAssertFalse(LearnerProfileLogic.learnerProfileEnabled(features))
    }

    func testSortFacetsUsesStablePriority() {
        let facets = [
            LearnerProfileFacetSummary(
                facetKey: "interests",
                state: "ok",
                summary: [:],
                confidence: 0.8,
                computedVersion: 1,
                updatedAt: "2026-01-01T00:00:00Z"
            ),
            LearnerProfileFacetSummary(
                facetKey: "study_rhythm",
                state: "ok",
                summary: [:],
                confidence: 0.7,
                computedVersion: 1,
                updatedAt: "2026-01-01T00:00:00Z"
            ),
            LearnerProfileFacetSummary(
                facetKey: "learning_approach",
                state: "insufficient_data",
                summary: [:],
                confidence: 0,
                computedVersion: 1,
                updatedAt: "2026-01-01T00:00:00Z"
            ),
        ]
        let sorted = LearnerProfileLogic.sortFacets(facets).map(\.facetKey)
        XCTAssertEqual(sorted, ["study_rhythm", "interests", "learning_approach"])
    }

    func testShowEmptyStateWhenInsufficient() {
        let profile = LearnerProfile(
            status: "insufficient_data",
            facets: [
                LearnerProfileFacetSummary(
                    facetKey: "study_rhythm",
                    state: "insufficient_data",
                    summary: [:],
                    confidence: 0,
                    computedVersion: 1,
                    updatedAt: "2026-01-01T00:00:00Z"
                ),
            ]
        )
        XCTAssertTrue(LearnerProfileLogic.showEmptyState(profile))
    }

    func testConfidenceLevels() {
        XCTAssertEqual(LearnerProfileLogic.confidenceLevel(0.9), .high)
        XCTAssertEqual(LearnerProfileLogic.confidenceLevel(0.6), .medium)
        XCTAssertEqual(LearnerProfileLogic.confidenceLevel(0.2), .low)
    }

    func testEvidenceAggregation() {
        let evidence = [
            LearnerProfileEvidenceRow(
                sourceKind: "quiz_attempt",
                sourceTable: "course.quiz_attempts",
                observationCount: 5,
                courseId: "a",
                windowStart: nil,
                windowEnd: nil,
                contribution: nil
            ),
            LearnerProfileEvidenceRow(
                sourceKind: "quiz_attempt",
                sourceTable: "course.quiz_attempts",
                observationCount: 7,
                courseId: "b",
                windowStart: nil,
                windowEnd: nil,
                contribution: nil
            ),
        ]
        XCTAssertEqual(LearnerProfileLogic.totalObservationCount(evidence), 12)
        XCTAssertEqual(LearnerProfileLogic.uniqueCourseCount(evidence), 2)
    }

    func testFormatPeakStudyWindowInsight() {
        let insight = LearnerProfileInsight(
            insightKey: "peak_study_window",
            label: "Peak",
            value: [
                "peakWindows": .array([
                    .object([
                        "dow": .string("Monday"),
                        "hourBucket": .string("19:00"),
                        "share": .number(0.42),
                    ]),
                ]),
            ],
            confidence: 0.42,
            salience: 100,
            evidence: nil
        )
        let formatted = LearnerProfileLogic.formatInsightValue(insight, facetKey: "study_rhythm")
        XCTAssertTrue(formatted.contains("Monday"))
    }
}