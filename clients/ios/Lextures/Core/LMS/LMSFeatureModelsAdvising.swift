import Foundation

// MARK: - Academic advising (M7.8)

struct AdvisingNote: Codable, Identifiable, Equatable {
    let id: String
    let studentId: String
    let advisorId: String
    let content: String
    let visibleToStudent: Bool
    let createdAt: String
    let advisorEmail: String?
    let advisorDisplayName: String?
}

struct AdvisingNotesResponse: Decodable {
    let notes: [AdvisingNote]?
}

struct AdvisingRequirementGroup: Codable, Equatable {
    let group: String
    let coursesRemaining: Int
}

struct DegreeProgress: Codable, Equatable {
    let configured: Bool
    let completionPercent: Int?
    let remainingRequiredCount: Int?
    let remainingRequirements: [AdvisingRequirementGroup]?
    let atRisk: Bool?
    let lastUpdated: String?
    let stale: Bool?
    let appointmentUrl: String?
    let recentNotesCount: Int?
}

struct MyAdvisingConfig: Decodable, Equatable {
    let appointmentUrl: String?
}

struct AdvisingAdvisorInfo: Equatable {
    let displayName: String
    let email: String?
}
