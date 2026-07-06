import Foundation

// MARK: - Behavior / PBIS (M10.3)

struct BehaviorCategory: Codable, Identifiable, Hashable {
    var id: String
    var orgId: String
    var name: String
    var type: String
    var color: String?
    var active: Bool

    var isPositive: Bool { type.lowercased() == "positive" }
    var isNegative: Bool { type.lowercased() == "negative" }
}

struct BehaviorCategoriesResponse: Decodable {
    var categories: [BehaviorCategory]?
}

struct PBISAwardInput: Encodable {
    var studentId: String
    var categoryId: String
    var points: Int
    var note: String?
}

struct PBISAwardsBody: Encodable {
    var awards: [PBISAwardInput]
}

struct PBISAward: Codable, Identifiable, Hashable {
    var id: String
    var studentId: String
    var awardedBy: String?
    var categoryId: String
    var categoryName: String?
    var orgId: String?
    var points: Int
    var note: String?
    var awardedAt: String
}

struct PBISAwardsResponse: Decodable {
    var saved: Int?
    var awards: [PBISAward]?
    var message: String?
}

struct BehaviorReferralBody: Encodable {
    var studentId: String
    var categoryId: String
    var schoolId: String?
    var incidentAt: String?
    var location: String?
    var description: String
    var response: String?
}

struct BehaviorReferral: Codable, Identifiable, Hashable {
    var id: String
    var studentId: String
    var filedBy: String?
    var orgId: String?
    var schoolId: String?
    var categoryId: String
    var categoryName: String?
    var incidentAt: String
    var location: String?
    var description: String?
    var response: String?
    var createdAt: String
}

struct StudentBehaviorResponse: Decodable {
    var studentId: String
    var totalPoints: Int
    var awards: [PBISAward]?
    var referrals: [BehaviorReferral]?
}

// MARK: - Hall pass (M10.3)

struct HallPass: Codable, Identifiable, Hashable {
    var id: String
    var sectionId: String
    var studentId: String?
    var destination: String
    var status: String
    var estimatedMins: Int?
    var requestedAt: String
    var approvedAt: String?
    var returnedAt: String?
    var approvedBy: String?
    var overdue: Bool?
}

struct HallPassResponse: Decodable {
    var pass: HallPass?
}

struct ActiveHallPassesResponse: Decodable {
    var passes: [HallPass]?
}

struct RequestHallPassBody: Encodable {
    var destination: String
    var estimatedMins: Int
}

struct UpdateHallPassBody: Encodable {
    var status: String
}
