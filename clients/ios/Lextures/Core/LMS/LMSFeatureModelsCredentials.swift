import Foundation

// MARK: - Completion credentials (M9.3)

struct IssuedCredentialSummary: Codable, Identifiable, Hashable {
    var id: String
    var title: String
    var sourceType: String
    var sourceId: String
    var issuedAt: String
    var verificationUrl: String
    var revoked: Bool
}

struct CredentialsListResponse: Codable {
    var credentials: [IssuedCredentialSummary]?
}

struct CredentialLinkedInParams: Codable {
    var name: String
    var organizationName: String
    var issueYear: Int
    var issueMonth: Int
    var certUrl: String
    var certId: String
    var url: String
}

struct CredentialBadgeExportResponse: Codable {
    var downloadUrl: String
    var expiresAt: String
}

struct CredentialShareRequest: Encodable {
    var channel: String
}