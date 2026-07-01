import Foundation

enum PeerReviewLogic {
    static func targetLabel(_ allocation: PeerReviewAllocation) -> String {
        allocation.targetLabel?.trimmingCharacters(in: .whitespacesAndNewlines).nonEmpty
            ?? L.text("mobile.peerReview.anonymousPeer")
    }

    static func isComplete(_ allocation: PeerReviewAllocation) -> Bool {
        allocation.status == .submitted
    }

    static func pending(_ allocations: [PeerReviewAllocation]) -> [PeerReviewAllocation] {
        allocations.filter { !isComplete($0) }
    }

    static func completedCount(_ allocations: [PeerReviewAllocation]) -> Int {
        allocations.filter(isComplete).count
    }

    static func rubricTotal(_ rubric: RubricDefinition, scores: [String: Double]) -> Double {
        rubric.criteria.reduce(0) { partial, criterion in
            partial + (scores[criterion.id] ?? 0)
        }
    }

    static func rubricGradedCount(_ rubric: RubricDefinition, scores: [String: Double]) -> Int {
        rubric.criteria.filter { scores[$0.id] != nil }.count
    }

    static func rubricScoresComplete(_ rubric: RubricDefinition, scores: [String: Double]) -> Bool {
        guard !rubric.criteria.isEmpty else { return true }
        return rubricGradedCount(rubric, scores: scores) == rubric.criteria.count
    }

    static func statusLabelKey(_ status: PeerReviewAllocationStatus) -> String {
        switch status {
        case .assigned: return "mobile.peerReview.statusAssigned"
        case .inProgress: return "mobile.peerReview.statusInProgress"
        case .submitted: return "mobile.peerReview.statusSubmitted"
        case .expired: return "mobile.peerReview.statusExpired"
        }
    }

    static func cacheKeyAssigned() -> String {
        "peer-review:assigned"
    }

    static func cacheKeyAllocation(_ id: String) -> String {
        "peer-review:allocation:\(id)"
    }

    static func cacheKeyReceived(courseCode: String, assignmentId: String) -> String {
        "peer-review:received:\(courseCode):\(assignmentId)"
    }
}

private extension String {
    var nonEmpty: String? {
        isEmpty ? nil : self
    }
}
