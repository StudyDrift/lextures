import XCTest
@testable import Lextures

final class PathsLogicTests: XCTestCase {
    func testNextCourseReturnsFirstIncomplete() {
        let progress = PathProgress(
            pathId: "p1",
            pathTitle: "Path",
            slug: "path",
            totalCourses: 3,
            completedCourses: 1,
            percent: 33,
            progressLabel: "33%",
            completedAt: nil,
            justCompleted: false,
            courses: [
                PathCourseProgress(
                    courseId: "c1",
                    position: 1,
                    courseCode: "A",
                    title: "A",
                    completed: true,
                    recommended: true
                ),
                PathCourseProgress(
                    courseId: "c2",
                    position: 2,
                    courseCode: "B",
                    title: "B",
                    completed: false,
                    recommended: true
                ),
                PathCourseProgress(
                    courseId: "c3",
                    position: 3,
                    courseCode: "C",
                    title: "C",
                    completed: false,
                    recommended: false
                ),
            ]
        )
        XCTAssertEqual(PathsLogic.nextCourse(in: progress)?.courseId, "c2")
    }

    func testIsLockedWhenNotRecommendedAndIncomplete() {
        let course = PathCourseProgress(
            courseId: "c3",
            position: 3,
            courseCode: "C",
            title: "C",
            completed: false,
            recommended: false
        )
        XCTAssertTrue(PathsLogic.isLocked(course))
    }

    func testMergeRecommendationsSortsByScore() {
        let responses = [
            LearnerRecommendationsResponse(
                recommendations: [
                    LearnerRecommendationItem(
                        itemId: "1",
                        itemType: "quiz",
                        title: "Low",
                        surface: "continue",
                        reason: "r",
                        score: 1
                    )
                ],
                degraded: false
            ),
            LearnerRecommendationsResponse(
                recommendations: [
                    LearnerRecommendationItem(
                        itemId: "2",
                        itemType: "assignment",
                        title: "High",
                        surface: "strengthen",
                        reason: "r",
                        score: 9
                    )
                ],
                degraded: true
            ),
        ]
        let merged = PathsLogic.mergeRecommendations(responses)
        XCTAssertEqual(merged.primary?.itemId, "2")
        XCTAssertEqual(merged.chips.count, 1)
        XCTAssertTrue(merged.degraded)
    }

    func testStructureItemKindMapping() {
        XCTAssertEqual(PathsLogic.structureItemKind(for: "quiz"), "quiz")
        XCTAssertNil(PathsLogic.structureItemKind(for: "unknown"))
    }
}