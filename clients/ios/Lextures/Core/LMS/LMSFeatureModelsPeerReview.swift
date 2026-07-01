import Foundation

enum PeerReviewAnonymity: String, Codable, Hashable {
    case doubleBlind = "double_blind"
    case reviewerAnon = "reviewer_anon"
    case named
}

enum PeerReviewAllocationStatus: String, Codable, Hashable {
    case assigned
    case inProgress = "in_progress"
    case submitted
    case expired
}

struct PeerReviewAllocation: Codable, Identifiable, Hashable {
    var id: String
    var configId: String
    var assignmentId: String
    var courseId: String
    var courseCode: String
    var targetSubmissionId: String
    var status: PeerReviewAllocationStatus
    var assignedAt: String
    var anonymity: PeerReviewAnonymity
    var targetLabel: String?
    var targetUserId: String?
    var closesAt: String?
    var opensAt: String?
}

struct PeerReviewAssignedResponse: Decodable {
    var allocations: [PeerReviewAllocation]
}

struct PeerReviewReceivedItem: Codable, Identifiable, Hashable {
    var id: String
    var score: Double?
    var comments: String?
    var submittedAt: String
    var reviewerLabel: String?
}

struct PeerReviewReceivedResponse: Decodable {
    var reviews: [PeerReviewReceivedItem]
}

struct PeerReviewExistingReview: Codable, Hashable {
    var id: String
    var allocationId: String
    var score: Double?
    var rubricScores: [String: Double]?
    var comments: String?
    var submittedAt: String
}

struct PeerReviewAllocationDetail: Decodable {
    var allocation: PeerReviewAllocation
    var assignmentTitle: String?
    var submission: AssignmentSubmission
    var rubric: RubricDefinition?
    var review: PeerReviewExistingReview?
}

struct PeerReviewSubmitRequest: Encodable {
    var score: Double?
    var rubricScores: [String: Double]?
    var comments: String?
}

struct PeerReviewSubmitResponse: Decodable {
    var id: String
    var allocationId: String
    var score: Double?
    var submittedAt: String
}

struct PeerReviewListRoute: Hashable {}

struct PeerReviewDetailRoute: Hashable {
    var allocationId: String
}

struct PeerReviewsReceivedRoute: Hashable {
    var courseCode: String
    var assignmentId: String
    var assignmentTitle: String
}
