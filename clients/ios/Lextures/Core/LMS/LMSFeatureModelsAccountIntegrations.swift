import Foundation

// MARK: - Account integrations (M14.1)

struct AccessKeyScopeDef: Decodable, Identifiable, Hashable {
    var id: String
    var label: String
    var description: String
    var group: String
}

struct AccessKeyScopesResponse: Decodable {
    var scopes: [AccessKeyScopeDef]
}

struct AccessKeyCourseSummary: Decodable, Hashable {
    var id: String
    var courseCode: String
    var title: String
}

struct AccessKeySummary: Decodable, Identifiable, Hashable {
    var id: String
    var label: String
    var tokenMask: String
    var scopes: [String]
    var courseIds: [String]?
    var courses: [AccessKeyCourseSummary]?
    var allCourses: Bool?
    var isServiceToken: Bool?
    var serviceAccountName: String?
    var expiresAt: String?
    var lastUsedAt: String?
    var revokedAt: String?
    var createdAt: String
    var unusedDays: Int?
}

struct AccessKeysListResponse: Decodable {
    var tokens: [AccessKeySummary]
}

struct CreateAccessKeyRequest: Encodable {
    var label: String
    var scopes: [String]
    var courseIds: [String] = []
}

struct CreateAccessKeyResponse: Decodable {
    var token: String?
    var label: String?
}

struct RotateAccessKeyRequest: Encodable {
    var overlapHours: Int = 24
}

struct RotateAccessKeyResponse: Decodable {
    var token: String?
    var label: String?
}

struct MCPConfigResponse: Decodable {
    var apiBaseUrl: String
    var cursorConfig: [String: JSONValue]
    var claudeDesktopConfig: [String: JSONValue]
    var instructions: [String]
}

struct CreateServiceTokenRequest: Encodable {
    var serviceAccountName: String
    var label: String
    var scopes: [String]
}

struct CreateServiceTokenResponse: Decodable {
    var token: String?
    var label: String?
}

struct OneTimeSecretReveal: Equatable {
    var token: String
    var label: String
}