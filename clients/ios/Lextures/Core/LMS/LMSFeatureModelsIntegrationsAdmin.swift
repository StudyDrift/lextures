import Foundation

// MARK: - Integrations & provisioning admin models (M14.8)

// Status-only models: secrets (cloud keys, SCIM bearer tokens, LRS passwords) are never decoded.

struct LtiRegistrationsResponse: Decodable {
    var parentPlatforms: [LtiParentPlatform]
    var externalTools: [LtiExternalTool]
}

struct LtiParentPlatform: Decodable, Identifiable, Hashable {
    var id: String
    var name: String
    var clientId: String
    var platformIss: String
    var active: Bool
}

struct LtiExternalTool: Decodable, Identifiable, Hashable {
    var id: String
    var name: String
    var clientId: String
    var toolIssuer: String
    var active: Bool
}

struct LtiActiveBody: Encodable {
    var active: Bool
}

struct ScimTokensResponse: Decodable {
    var tokens: [ScimTokenRow]?
}

struct ScimTokenRow: Decodable, Identifiable, Hashable {
    var id: String
    var institutionId: String
    var label: String
    var createdAt: String
    var revokedAt: String?
}

struct ScimEventsResponse: Decodable {
    var events: [ScimEventRow]?
}

struct ScimEventRow: Decodable, Identifiable, Hashable {
    var id: String
    var operation: String
    var scimResource: String
    var userEmail: String?
    var createdAt: String
}

/// Platform settings subset used only for SCIM feature gating.
struct PlatformScimFlag: Decodable {
    var scimEnabled: Bool?
}

/// Secret-free cloud provider status for mobile admin.
struct CloudProviderStatus: Decodable, Identifiable, Hashable {
    var provider: String
    var enabled: Bool
    var updatedAt: String?

    var id: String { provider }
}

struct CloudProviderEnabledBody: Encodable {
    var enabled: Bool
}

struct LrsEndpointStatus: Decodable, Identifiable, Hashable {
    var id: String
    var label: String
    var endpointUrl: String
    var authType: String
    var username: String?
    var enabled: Bool
    var hasPassword: Bool?
    var hasOauthSecret: Bool?
    var updatedAt: String?
}

struct LrsEnabledBody: Encodable {
    var enabled: Bool
}

struct OerProviderStatus: Decodable, Identifiable, Hashable {
    var provider: String
    var enabled: Bool
    var updatedAt: String?

    var id: String { provider }
}

struct OerProviderEnabledBody: Encodable {
    var enabled: Bool
}
