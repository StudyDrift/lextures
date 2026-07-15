import Foundation

// MARK: - Transcripts & Advising admin models (M14.9)

struct AdminTranscriptsConfig: Codable, Equatable {
    var webhookUrl: String
    var webhookSecret: String?
    var hasWebhookSecret: Bool
    var pickupInstructions: String?

    init(
        webhookUrl: String = "",
        webhookSecret: String? = nil,
        hasWebhookSecret: Bool = false,
        pickupInstructions: String? = nil
    ) {
        self.webhookUrl = webhookUrl
        self.webhookSecret = webhookSecret
        self.hasWebhookSecret = hasWebhookSecret
        self.pickupInstructions = pickupInstructions
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        webhookUrl = try container.decodeIfPresent(String.self, forKey: .webhookUrl) ?? ""
        webhookSecret = try container.decodeIfPresent(String.self, forKey: .webhookSecret)
        hasWebhookSecret = try container.decodeIfPresent(Bool.self, forKey: .hasWebhookSecret) ?? false
        pickupInstructions = try container.decodeIfPresent(String.self, forKey: .pickupInstructions)
    }
}

struct PutAdminTranscriptsConfigRequest: Encodable {
    var webhookUrl: String
    var webhookSecret: String?
    var pickupInstructions: String?

    func encode(to encoder: Encoder) throws {
        var container = encoder.container(keyedBy: CodingKeys.self)
        try container.encode(webhookUrl, forKey: .webhookUrl)
        if let webhookSecret {
            try container.encode(webhookSecret, forKey: .webhookSecret)
        }
        try container.encode(pickupInstructions ?? "", forKey: .pickupInstructions)
    }

    private enum CodingKeys: String, CodingKey {
        case webhookUrl
        case webhookSecret
        case pickupInstructions
    }
}

struct AdminTranscriptRequestRow: Decodable, Identifiable, Equatable, Hashable {
    var id: String
    var status: String?
    var deliveryType: String?
    var requestedAt: String
    var submittedAt: String?
    var errorMessage: String?
    var webhookResponseCode: Int?
}

struct AdminTranscriptRequestsResponse: Decodable {
    var requests: [AdminTranscriptRequestRow]?
}

struct AdminAdvisingConfig: Codable, Equatable {
    var appointmentUrl: String
    var degreeAuditProvider: String
    var degreeAuditBaseUrl: String
    var apiCredentialsRef: String
    var atRiskBannerEnabled: Bool

    init(
        appointmentUrl: String = "",
        degreeAuditProvider: String = "none",
        degreeAuditBaseUrl: String = "",
        apiCredentialsRef: String = "",
        atRiskBannerEnabled: Bool = false
    ) {
        self.appointmentUrl = appointmentUrl
        self.degreeAuditProvider = degreeAuditProvider
        self.degreeAuditBaseUrl = degreeAuditBaseUrl
        self.apiCredentialsRef = apiCredentialsRef
        self.atRiskBannerEnabled = atRiskBannerEnabled
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        appointmentUrl = try container.decodeIfPresent(String.self, forKey: .appointmentUrl) ?? ""
        degreeAuditProvider = try container.decodeIfPresent(String.self, forKey: .degreeAuditProvider) ?? "none"
        degreeAuditBaseUrl = try container.decodeIfPresent(String.self, forKey: .degreeAuditBaseUrl) ?? ""
        apiCredentialsRef = try container.decodeIfPresent(String.self, forKey: .apiCredentialsRef) ?? ""
        atRiskBannerEnabled = try container.decodeIfPresent(Bool.self, forKey: .atRiskBannerEnabled) ?? false
    }
}

struct PutAdminAdvisingConfigRequest: Encodable {
    var appointmentUrl: String
    var degreeAuditProvider: String
    var degreeAuditBaseUrl: String
    var apiCredentialsRef: String
    var atRiskBannerEnabled: Bool
}
