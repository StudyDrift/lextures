import Foundation

// MARK: - Learner profile (LP10)

typealias LearnerProfileFacetKey = String

struct LearnerProfile: Codable, Equatable {
    var status: String
    var lastComputedAt: String?
    var facets: [LearnerProfileFacetSummary]
}

struct LearnerProfileFacetSummary: Codable, Equatable, Identifiable {
    var facetKey: LearnerProfileFacetKey
    var state: String
    var summary: [String: JSONValue]
    var confidence: Double
    var computedVersion: Int
    var updatedAt: String

    var id: String { facetKey }
}

struct LearnerProfileInsight: Codable, Equatable, Identifiable {
    var insightKey: String
    var label: String
    var value: [String: JSONValue]
    var confidence: Double
    var salience: Double
    var evidence: [LearnerProfileEvidenceRow]?

    var id: String { insightKey }
}

struct LearnerProfileEvidenceRow: Codable, Equatable, Identifiable {
    var sourceKind: String
    var sourceTable: String
    var observationCount: Int
    var courseId: String?
    var windowStart: String?
    var windowEnd: String?
    var contribution: Double?

    var id: String { "\(sourceKind)-\(sourceTable)-\(observationCount)-\(courseId ?? "")" }
}

struct LearnerProfileResponse: Codable {
    var profile: LearnerProfile?
}

struct LearnerProfileFacetDetailResponse: Codable {
    var facet: LearnerProfileFacetSummary?
    var insights: [LearnerProfileInsight]?
}

struct LearnerProfileControlResponse: Codable {
    var status: String?
}

typealias LearnerProfileEvidenceMap = [String: [LearnerProfileEvidenceRow]]