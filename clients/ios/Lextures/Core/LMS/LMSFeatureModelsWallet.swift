import Foundation

// MARK: - Credentials wallet (M12.2)

struct CCRAchievement: Codable, Identifiable, Hashable {
    var id: String
    var type: String
    var title: String
    var description: String?
    var issuedAt: String
    var evidenceUrl: String?
    var outcomeTags: [String]?
}

struct CCRDocument: Codable, Identifiable, Hashable {
    var id: String
    var generatedAt: String
    var shareable: Bool
    var verificationUrl: String?
}

struct CCRSummaryResponse: Codable {
    var achievements: [CCRAchievement]?
    var documents: [CCRDocument]?
}

struct CCRGenerateRequest: Encodable {
    var sharePublicly: Bool
}

struct CCRGenerateResponse: Decodable {
    var document: CCRDocument
    var achievements: [CCRAchievement]?
    var verificationUrl: String?
}

struct CETranscriptAward: Codable, Identifiable, Hashable {
    var courseTitle: String
    var ceuCredit: Double
    var contactHours: Double
    var completedAt: String

    var id: String { "\(courseTitle)-\(completedAt)" }
}

struct CETranscriptResponse: Codable {
    var awards: [CETranscriptAward]?
}

struct TranscriptRequestSummary: Codable, Identifiable, Hashable {
    var id: String
    var status: String
    var deliveryType: String
    var deliveryEmail: String?
    var deliveryAddress: String?
    var urgencyDays: Int?
    var urgencyDaysMin: Int?
    var urgencyUnit: String?
    var requestedAt: String
    var submittedAt: String?
    var errorMessage: String?
    var webhookResponseCode: Int?
}

struct TranscriptRequestsResponse: Codable {
    var requests: [TranscriptRequestSummary]?
}

struct TranscriptsStudentConfig: Codable {
    var pickupInstructions: String?
    var pickupAvailable: Bool
}

struct TranscriptRequestCreateBody: Encodable {
    var deliveryType: String
    var deliveryEmail: String?
    var deliveryAddress: String?
    var mailUrgency: String?
    var urgencyDays: Int?
}

struct TranscriptRequestCreateResponse: Decodable {
    var request: TranscriptRequestSummary?
}
