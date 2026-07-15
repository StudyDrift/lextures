import Foundation

// MARK: - AI models, system prompts & reports (M14.7)

struct AiSettingsResponse: Codable, Equatable {
    var imageModelId: String
    var courseSetupModelId: String
    var notebookFlashcardsModelId: String
    var vibeActivityModelId: String
    var graderAgentModelId: String
    var openRouterApiKey: String?

    enum CodingKeys: String, CodingKey {
        case imageModelId
        case courseSetupModelId
        case notebookFlashcardsModelId
        case vibeActivityModelId
        case graderAgentModelId
        case openRouterApiKey
    }

    init(
        imageModelId: String = "",
        courseSetupModelId: String = "",
        notebookFlashcardsModelId: String = "",
        vibeActivityModelId: String = "",
        graderAgentModelId: String = "",
        openRouterApiKey: String? = nil
    ) {
        self.imageModelId = imageModelId
        self.courseSetupModelId = courseSetupModelId
        self.notebookFlashcardsModelId = notebookFlashcardsModelId
        self.vibeActivityModelId = vibeActivityModelId
        self.graderAgentModelId = graderAgentModelId
        self.openRouterApiKey = openRouterApiKey
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        imageModelId = try container.decodeIfPresent(String.self, forKey: .imageModelId) ?? ""
        courseSetupModelId = try container.decodeIfPresent(String.self, forKey: .courseSetupModelId) ?? ""
        notebookFlashcardsModelId = try container.decodeIfPresent(String.self, forKey: .notebookFlashcardsModelId) ?? ""
        vibeActivityModelId = try container.decodeIfPresent(String.self, forKey: .vibeActivityModelId) ?? ""
        graderAgentModelId = try container.decodeIfPresent(String.self, forKey: .graderAgentModelId) ?? ""
        openRouterApiKey = try container.decodeIfPresent(String.self, forKey: .openRouterApiKey)
    }
}

struct PutAiSettingsRequest: Encodable {
    var imageModelId: String
    var courseSetupModelId: String
    var notebookFlashcardsModelId: String
    var vibeActivityModelId: String
    var graderAgentModelId: String
    var openRouterApiKey: String?
    var clearOpenRouterApiKey: Bool?

    func encode(to encoder: Encoder) throws {
        var container = encoder.container(keyedBy: CodingKeys.self)
        try container.encode(imageModelId, forKey: .imageModelId)
        try container.encode(courseSetupModelId, forKey: .courseSetupModelId)
        try container.encode(notebookFlashcardsModelId, forKey: .notebookFlashcardsModelId)
        try container.encode(vibeActivityModelId, forKey: .vibeActivityModelId)
        try container.encode(graderAgentModelId, forKey: .graderAgentModelId)
        if let openRouterApiKey {
            try container.encode(openRouterApiKey, forKey: .openRouterApiKey)
        }
        if let clearOpenRouterApiKey, clearOpenRouterApiKey {
            try container.encode(true, forKey: .clearOpenRouterApiKey)
        }
    }

    private enum CodingKeys: String, CodingKey {
        case imageModelId
        case courseSetupModelId
        case notebookFlashcardsModelId
        case vibeActivityModelId
        case graderAgentModelId
        case openRouterApiKey
        case clearOpenRouterApiKey
    }
}

struct AiModelOption: Codable, Equatable, Identifiable, Hashable {
    var id: String
    var name: String?
    var contextLength: UInt64?
    var inputPricePerMillionUsd: Double?
    var outputPricePerMillionUsd: Double?
    var modalitiesSummary: String?

    init(
        id: String,
        name: String? = nil,
        contextLength: UInt64? = nil,
        inputPricePerMillionUsd: Double? = nil,
        outputPricePerMillionUsd: Double? = nil,
        modalitiesSummary: String? = nil
    ) {
        self.id = id
        self.name = name
        self.contextLength = contextLength
        self.inputPricePerMillionUsd = inputPricePerMillionUsd
        self.outputPricePerMillionUsd = outputPricePerMillionUsd
        self.modalitiesSummary = modalitiesSummary
    }
}

struct AiModelsListResponse: Decodable {
    var configured: Bool
    var models: [AiModelOption]

    init(configured: Bool = false, models: [AiModelOption] = []) {
        self.configured = configured
        self.models = models
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        configured = try container.decodeIfPresent(Bool.self, forKey: .configured) ?? false
        models = try container.decodeIfPresent([AiModelOption].self, forKey: .models) ?? []
    }

    private enum CodingKeys: String, CodingKey {
        case configured, models
    }
}

struct SystemPromptItem: Codable, Equatable, Identifiable {
    var key: String
    var label: String
    var content: String
    var updatedAt: String?

    var id: String { key }

    init(key: String, label: String, content: String, updatedAt: String? = nil) {
        self.key = key
        self.label = label
        self.content = content
        self.updatedAt = updatedAt
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        key = try container.decodeIfPresent(String.self, forKey: .key) ?? ""
        label = try container.decodeIfPresent(String.self, forKey: .label) ?? key
        content = try container.decodeIfPresent(String.self, forKey: .content) ?? ""
        updatedAt = try container.decodeIfPresent(String.self, forKey: .updatedAt)
    }
}

struct SystemPromptsListResponse: Decodable {
    var prompts: [SystemPromptItem]

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        prompts = try container.decodeIfPresent([SystemPromptItem].self, forKey: .prompts) ?? []
    }

    private enum CodingKeys: String, CodingKey {
        case prompts
    }
}

struct PutSystemPromptRequest: Encodable {
    var content: String
}

struct AiReportsPayload: Decodable, Equatable {
    var range: AiReportsDateRange
    var cost: AiCostReport
    var byUser: [AiUserUsageRow]
    var byCourse: [AiCourseUsageRow]

    init(
        range: AiReportsDateRange = AiReportsDateRange(),
        cost: AiCostReport = AiCostReport(),
        byUser: [AiUserUsageRow] = [],
        byCourse: [AiCourseUsageRow] = []
    ) {
        self.range = range
        self.cost = cost
        self.byUser = byUser
        self.byCourse = byCourse
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        range = try container.decodeIfPresent(AiReportsDateRange.self, forKey: .range) ?? AiReportsDateRange()
        cost = try container.decodeIfPresent(AiCostReport.self, forKey: .cost) ?? AiCostReport()
        byUser = try container.decodeIfPresent([AiUserUsageRow].self, forKey: .byUser) ?? []
        byCourse = try container.decodeIfPresent([AiCourseUsageRow].self, forKey: .byCourse) ?? []
    }

    private enum CodingKeys: String, CodingKey {
        case range, cost, byUser, byCourse
    }
}

struct AiReportsDateRange: Decodable, Equatable {
    var from: String
    var to: String

    init(from: String = "", to: String = "") {
        self.from = from
        self.to = to
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        self.from = try container.decodeIfPresent(String.self, forKey: .from) ?? ""
        self.to = try container.decodeIfPresent(String.self, forKey: .to) ?? ""
    }

    private enum CodingKeys: String, CodingKey {
        case from, to
    }
}

struct AiCostReport: Decodable, Equatable {
    var summary: AiCostSummary
    var byDay: [AiDayCostBucket]
    var byFeature: [AiFeatureCostRow]

    init(
        summary: AiCostSummary = AiCostSummary(),
        byDay: [AiDayCostBucket] = [],
        byFeature: [AiFeatureCostRow] = []
    ) {
        self.summary = summary
        self.byDay = byDay
        self.byFeature = byFeature
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        summary = try container.decodeIfPresent(AiCostSummary.self, forKey: .summary) ?? AiCostSummary()
        byDay = try container.decodeIfPresent([AiDayCostBucket].self, forKey: .byDay) ?? []
        byFeature = try container.decodeIfPresent([AiFeatureCostRow].self, forKey: .byFeature) ?? []
    }

    private enum CodingKeys: String, CodingKey {
        case summary, byDay, byFeature
    }
}

struct AiCostSummary: Decodable, Equatable {
    var totalCostUsd: Double
    var totalCalls: Int64
    var totalTokens: Int64

    init(totalCostUsd: Double = 0, totalCalls: Int64 = 0, totalTokens: Int64 = 0) {
        self.totalCostUsd = totalCostUsd
        self.totalCalls = totalCalls
        self.totalTokens = totalTokens
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        totalCostUsd = try container.decodeIfPresent(Double.self, forKey: .totalCostUsd) ?? 0
        totalCalls = try container.decodeIfPresent(Int64.self, forKey: .totalCalls) ?? 0
        totalTokens = try container.decodeIfPresent(Int64.self, forKey: .totalTokens) ?? 0
    }

    private enum CodingKeys: String, CodingKey {
        case totalCostUsd, totalCalls, totalTokens
    }
}

struct AiDayCostBucket: Decodable, Equatable, Identifiable {
    var day: String
    var costUsd: Double
    var calls: Int64
    var tokens: Int64

    var id: String { day }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        day = try container.decodeIfPresent(String.self, forKey: .day) ?? ""
        costUsd = try container.decodeIfPresent(Double.self, forKey: .costUsd) ?? 0
        calls = try container.decodeIfPresent(Int64.self, forKey: .calls) ?? 0
        tokens = try container.decodeIfPresent(Int64.self, forKey: .tokens) ?? 0
    }

    private enum CodingKeys: String, CodingKey {
        case day, costUsd, calls, tokens
    }
}

struct AiFeatureCostRow: Decodable, Equatable, Identifiable {
    var feature: String
    var costUsd: Double
    var calls: Int64
    var tokens: Int64

    var id: String { feature }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        feature = try container.decodeIfPresent(String.self, forKey: .feature) ?? ""
        costUsd = try container.decodeIfPresent(Double.self, forKey: .costUsd) ?? 0
        calls = try container.decodeIfPresent(Int64.self, forKey: .calls) ?? 0
        tokens = try container.decodeIfPresent(Int64.self, forKey: .tokens) ?? 0
    }

    private enum CodingKeys: String, CodingKey {
        case feature, costUsd, calls, tokens
    }
}

struct AiUserUsageRow: Decodable, Equatable, Identifiable {
    var userId: String
    var email: String
    var displayName: String
    var calls: Int64
    var promptTokens: Int64
    var completionTokens: Int64
    var totalTokens: Int64
    var costUsd: Double

    var id: String { userId }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        userId = try container.decodeIfPresent(String.self, forKey: .userId) ?? ""
        email = try container.decodeIfPresent(String.self, forKey: .email) ?? ""
        displayName = try container.decodeIfPresent(String.self, forKey: .displayName) ?? ""
        calls = try container.decodeIfPresent(Int64.self, forKey: .calls) ?? 0
        promptTokens = try container.decodeIfPresent(Int64.self, forKey: .promptTokens) ?? 0
        completionTokens = try container.decodeIfPresent(Int64.self, forKey: .completionTokens) ?? 0
        totalTokens = try container.decodeIfPresent(Int64.self, forKey: .totalTokens) ?? 0
        costUsd = try container.decodeIfPresent(Double.self, forKey: .costUsd) ?? 0
    }

    private enum CodingKeys: String, CodingKey {
        case userId, email, displayName, calls, promptTokens, completionTokens, totalTokens, costUsd
    }
}

struct AiCourseUsageRow: Decodable, Equatable, Identifiable {
    var courseId: String
    var courseCode: String
    var title: String
    var calls: Int64
    var totalTokens: Int64
    var costUsd: Double

    var id: String { courseId }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        courseId = try container.decodeIfPresent(String.self, forKey: .courseId) ?? ""
        courseCode = try container.decodeIfPresent(String.self, forKey: .courseCode) ?? ""
        title = try container.decodeIfPresent(String.self, forKey: .title) ?? ""
        calls = try container.decodeIfPresent(Int64.self, forKey: .calls) ?? 0
        totalTokens = try container.decodeIfPresent(Int64.self, forKey: .totalTokens) ?? 0
        costUsd = try container.decodeIfPresent(Double.self, forKey: .costUsd) ?? 0
    }

    private enum CodingKeys: String, CodingKey {
        case courseId, courseCode, title, calls, totalTokens, costUsd
    }
}
