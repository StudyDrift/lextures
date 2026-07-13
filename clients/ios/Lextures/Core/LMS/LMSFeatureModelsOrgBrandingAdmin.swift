import Foundation

// MARK: - Org branding / AI governance / AI provider admin (M14.5)

struct OrgBrandingResponse: Decodable, Hashable {
    var logoUrl: String?
    var faviconUrl: String?
    var primaryColor: String
    var secondaryColor: String
    var customDomain: String?
    var customEmailDisplayName: String?
    var contrastWarningPrimary: Bool?
    var contrastRatioPrimary: Double?

    init(
        logoUrl: String? = nil,
        faviconUrl: String? = nil,
        primaryColor: String = OrgBrandingAdminLogic.defaultPrimaryColor,
        secondaryColor: String = OrgBrandingAdminLogic.defaultSecondaryColor,
        customDomain: String? = nil,
        customEmailDisplayName: String? = nil,
        contrastWarningPrimary: Bool? = nil,
        contrastRatioPrimary: Double? = nil
    ) {
        self.logoUrl = logoUrl
        self.faviconUrl = faviconUrl
        self.primaryColor = primaryColor
        self.secondaryColor = secondaryColor
        self.customDomain = customDomain
        self.customEmailDisplayName = customEmailDisplayName
        self.contrastWarningPrimary = contrastWarningPrimary
        self.contrastRatioPrimary = contrastRatioPrimary
    }

    enum CodingKeys: String, CodingKey {
        case logoUrl, faviconUrl, primaryColor, secondaryColor
        case customDomain, customEmailDisplayName
        case contrastWarningPrimary, contrastRatioPrimary
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        logoUrl = try c.decodeIfPresent(String.self, forKey: .logoUrl)
        faviconUrl = try c.decodeIfPresent(String.self, forKey: .faviconUrl)
        primaryColor = try c.decodeIfPresent(String.self, forKey: .primaryColor)
            ?? OrgBrandingAdminLogic.defaultPrimaryColor
        secondaryColor = try c.decodeIfPresent(String.self, forKey: .secondaryColor)
            ?? OrgBrandingAdminLogic.defaultSecondaryColor
        customDomain = try c.decodeIfPresent(String.self, forKey: .customDomain)
        customEmailDisplayName = try c.decodeIfPresent(String.self, forKey: .customEmailDisplayName)
        contrastWarningPrimary = try c.decodeIfPresent(Bool.self, forKey: .contrastWarningPrimary)
        contrastRatioPrimary = try c.decodeIfPresent(Double.self, forKey: .contrastRatioPrimary)
    }
}

struct OrgBrandingPutRequest: Encodable {
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

struct AIGovernanceConfig: Decodable, Hashable {
    var orgId: String?
    var featuresEnabled: [String: Bool]?
    var allowedModels: [String]?
    var updatedAt: String?
    var updatedBy: String?
}

struct AIGovernancePutRequest: Encodable {
    var featuresEnabled: [String: Bool]
    var allowedModels: [String]?
}

struct AIProviderSettings: Decodable, Hashable {
    var orgId: String?
    var provider: String?
    var modelAlias: String?
    var fallbackProvider: String?
    var byokConfigured: Bool?
    var providers: [String]?
    var modelAliases: [String]?
    var updatedAt: String?
    var updatedBy: String?
}

struct AIProviderSettingsPutRequest: Encodable {
    var provider: String
    var modelAlias: String
    var fallbackProvider: String?
    var byokApiKey: String?

    enum CodingKeys: String, CodingKey {
        case provider, modelAlias, fallbackProvider, byokApiKey
    }

    func encode(to encoder: Encoder) throws {
        var c = encoder.container(keyedBy: CodingKeys.self)
        try c.encode(provider, forKey: .provider)
        try c.encode(modelAlias, forKey: .modelAlias)
        try c.encodeIfPresent(fallbackProvider, forKey: .fallbackProvider)
        try c.encodeIfPresent(byokApiKey, forKey: .byokApiKey)
    }
}

struct AIProviderTestResponse: Decodable {
    var provider: String?
    var latencyMs: Double?
    var responsePreview: String?
}
