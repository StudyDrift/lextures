import Foundation

// MARK: - Org branding admin (M14.5)

struct OrgBrandingResponse: Codable, Equatable {
    var logoUrl: String?
    var faviconUrl: String?
    var primaryColor: String
    var secondaryColor: String
    var customDomain: String?
    var customEmailDisplayName: String?
    var contrastWarningPrimary: Bool?
    var contrastRatioPrimary: Double?
}

struct PutOrgBrandingRequest: Encodable {
    var logoUrl: String?
    var faviconUrl: String?
    var primaryColor: String
    var secondaryColor: String
    var customDomain: String?
    var customEmailDisplayName: String?
}

struct OrgBrandingUploadResponse: Decodable {
    var url: String?
}

struct AiConfigResponse: Decodable {
    var orgId: String?
    var featuresEnabled: [String: Bool]?
    var allowedModels: [String]?
}

struct PutAiConfigRequest: Encodable {
    var featuresEnabled: [String: Bool]
    var allowedModels: [String]?
}

struct AiProviderSettingsResponse: Decodable {
    var orgId: String?
    var provider: String?
    var modelAlias: String?
    var fallbackProvider: String?
    var byokConfigured: Bool?
    var settings: [String: JSONValue]?
    var providers: [String]?
    var modelAliases: [String]?
}

struct PutAiProviderSettingsRequest: Encodable {
    var provider: String
    var modelAlias: String
    var fallbackProvider: String?
    var byokApiKey: String?
}

struct AiProviderTestResponse: Decodable {
    var provider: String?
    var latencyMs: Int?
    var responsePreview: String?
}
