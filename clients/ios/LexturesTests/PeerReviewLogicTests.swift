import XCTest
@testable import Lextures

final class PeerReviewLogicTests: XCTestCase {
    func testPendingExcludesSubmitted() {
        let allocations = [
            makeAllocation(id: "a", status: .assigned),
            makeAllocation(id: "b", status: .submitted),
        ]
        XCTAssertEqual(PeerReviewLogic.pending(allocations).map(\.id), ["a"])
        XCTAssertEqual(PeerReviewLogic.completedCount(allocations), 1)
    }

    func testRubricTotalAndCompletion() {
        let rubric = RubricDefinition(
            title: "Essay",
            criteria: [
                RubricCriterion(
                    id: "c1",
                    title: "Thesis",
                    description: nil,
                    levels: [RubricLevel(label: "Good", points: 4, description: nil)]
                ),
                RubricCriterion(
                    id: "c2",
                    title: "Evidence",
                    description: nil,
                    levels: [RubricLevel(label: "Good", points: 6, description: nil)]
                ),
            ]
        )
        let partial: [String: Double] = ["c1": 4]
        XCTAssertEqual(PeerReviewLogic.rubricTotal(rubric, scores: partial), 4)
        XCTAssertFalse(PeerReviewLogic.rubricScoresComplete(rubric, scores: partial))
        let complete: [String: Double] = ["c1": 4, "c2": 6]
        XCTAssertEqual(PeerReviewLogic.rubricTotal(rubric, scores: complete), 10)
        XCTAssertTrue(PeerReviewLogic.rubricScoresComplete(rubric, scores: complete))
    }

    func testTargetLabelUsesServerLabel() {
        let allocation = makeAllocation(id: "a", status: .assigned, targetLabel: "Student 42")
        XCTAssertEqual(PeerReviewLogic.targetLabel(allocation), "Student 42")
    }

    private func makeAllocation(
        id: String,
        status: PeerReviewAllocationStatus,
        targetLabel: String? = nil
    ) -> PeerReviewAllocation {
        PeerReviewAllocation(
            id: id,
            configId: "cfg",
            assignmentId: "assign",
            courseId: "course",
            courseCode: "C-101",
            targetSubmissionId: "sub",
            status: status,
            assignedAt: "2024-01-01T00:00:00Z",
            anonymity: .doubleBlind,
            targetLabel: targetLabel
        )
    }
}
