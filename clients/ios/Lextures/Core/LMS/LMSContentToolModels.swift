import Foundation

// MARK: - CT.M3 Content Tool models (mirror server contenttools types)

struct ToolScore: Codable, Equatable, Hashable {
    var raw: Double
    var max: Double
}

struct ToolStateEnvelope: Codable, Equatable, Hashable {
    var instanceId: String
    var revision: Int64
    var status: String
    var state: JSONValue
    var stateJson: JSONValue?
    var score: ToolScore?
    var updatedAt: String?
    var resetCount: Int
    var lastResetAt: String?
    var scope: String?
    var stateSchemaVersion: Int
    var quarantined: Bool

    init(
        instanceId: String,
        revision: Int64 = 0,
        status: String = "not_started",
        state: JSONValue = .object([:]),
        stateJson: JSONValue? = nil,
        score: ToolScore? = nil,
        updatedAt: String? = nil,
        resetCount: Int = 0,
        lastResetAt: String? = nil,
        scope: String? = nil,
        stateSchemaVersion: Int = 0,
        quarantined: Bool = false
    ) {
        self.instanceId = instanceId
        self.revision = revision
        self.status = status
        self.state = state
        self.stateJson = stateJson
        self.score = score
        self.updatedAt = updatedAt
        self.resetCount = resetCount
        self.lastResetAt = lastResetAt
        self.scope = scope
        self.stateSchemaVersion = stateSchemaVersion
        self.quarantined = quarantined
    }

    enum CodingKeys: String, CodingKey {
        case instanceId, revision, status, state, stateJson, score, updatedAt
        case resetCount, lastResetAt, scope, stateSchemaVersion, quarantined
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        instanceId = try c.decode(String.self, forKey: .instanceId)
        revision = try c.decodeIfPresent(Int64.self, forKey: .revision) ?? 0
        status = try c.decodeIfPresent(String.self, forKey: .status) ?? "not_started"
        state = try c.decodeIfPresent(JSONValue.self, forKey: .state) ?? .object([:])
        stateJson = try c.decodeIfPresent(JSONValue.self, forKey: .stateJson)
        score = try c.decodeIfPresent(ToolScore.self, forKey: .score)
        updatedAt = try c.decodeIfPresent(String.self, forKey: .updatedAt)
        resetCount = try c.decodeIfPresent(Int.self, forKey: .resetCount) ?? 0
        lastResetAt = try c.decodeIfPresent(String.self, forKey: .lastResetAt)
        scope = try c.decodeIfPresent(String.self, forKey: .scope)
        stateSchemaVersion = try c.decodeIfPresent(Int.self, forKey: .stateSchemaVersion) ?? 0
        quarantined = try c.decodeIfPresent(Bool.self, forKey: .quarantined) ?? false
    }

    var document: JSONValue { stateJson ?? state }

    static func empty(instanceId: String) -> ToolStateEnvelope {
        ToolStateEnvelope(instanceId: instanceId)
    }
}

struct ToolInstance: Codable, Equatable, Identifiable, Hashable {
    var id: String
    var toolId: String
    var toolVersion: String
    var hostKind: String
    var structureItemId: String?
    var sectionKey: String?
    var title: String?
    var config: JSONValue
    var status: String
    var updatedAt: String
    var state: ToolStateEnvelope?
    var sandboxMode: String?
    var contract: Int
    var breakerOpen: Bool
    var deprecated: Bool
    var sunsetAt: String?
    var capabilities: [String]
    var tombstone: Bool

    init(
        id: String,
        toolId: String,
        toolVersion: String = "1.0.0",
        hostKind: String = "content_page",
        structureItemId: String? = nil,
        sectionKey: String? = nil,
        title: String? = nil,
        config: JSONValue = .object([:]),
        status: String = "published",
        updatedAt: String = "",
        state: ToolStateEnvelope? = nil,
        sandboxMode: String? = nil,
        contract: Int = 0,
        breakerOpen: Bool = false,
        deprecated: Bool = false,
        sunsetAt: String? = nil,
        capabilities: [String] = [],
        tombstone: Bool = false
    ) {
        self.id = id
        self.toolId = toolId
        self.toolVersion = toolVersion
        self.hostKind = hostKind
        self.structureItemId = structureItemId
        self.sectionKey = sectionKey
        self.title = title
        self.config = config
        self.status = status
        self.updatedAt = updatedAt
        self.state = state
        self.sandboxMode = sandboxMode
        self.contract = contract
        self.breakerOpen = breakerOpen
        self.deprecated = deprecated
        self.sunsetAt = sunsetAt
        self.capabilities = capabilities
        self.tombstone = tombstone
    }

    enum CodingKeys: String, CodingKey {
        case id, toolId, toolVersion, hostKind, structureItemId, sectionKey, title
        case config, status, updatedAt, state, sandboxMode, contract, breakerOpen
        case deprecated, sunsetAt, capabilities, tombstone
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        id = try c.decode(String.self, forKey: .id)
        toolId = try c.decode(String.self, forKey: .toolId)
        toolVersion = try c.decodeIfPresent(String.self, forKey: .toolVersion) ?? "1.0.0"
        hostKind = try c.decodeIfPresent(String.self, forKey: .hostKind) ?? "content_page"
        structureItemId = try c.decodeIfPresent(String.self, forKey: .structureItemId)
        sectionKey = try c.decodeIfPresent(String.self, forKey: .sectionKey)
        title = try c.decodeIfPresent(String.self, forKey: .title)
        config = try c.decodeIfPresent(JSONValue.self, forKey: .config) ?? .object([:])
        status = try c.decodeIfPresent(String.self, forKey: .status) ?? "published"
        updatedAt = try c.decodeIfPresent(String.self, forKey: .updatedAt) ?? ""
        state = try c.decodeIfPresent(ToolStateEnvelope.self, forKey: .state)
        sandboxMode = try c.decodeIfPresent(String.self, forKey: .sandboxMode)
        contract = try c.decodeIfPresent(Int.self, forKey: .contract) ?? 0
        breakerOpen = try c.decodeIfPresent(Bool.self, forKey: .breakerOpen) ?? false
        deprecated = try c.decodeIfPresent(Bool.self, forKey: .deprecated) ?? false
        sunsetAt = try c.decodeIfPresent(String.self, forKey: .sunsetAt)
        capabilities = try c.decodeIfPresent([String].self, forKey: .capabilities) ?? []
        tombstone = try c.decodeIfPresent(Bool.self, forKey: .tombstone) ?? false
    }
}

struct ToolInstancesListResponse: Codable, Equatable {
    var instances: [ToolInstance]
}

/// CT.M9 — org governance policy snapshot (mirrors server OrgPolicy).
struct ToolGovernancePolicy: Codable, Equatable {
    var deniedCapabilities: [String]
    var deniedToolIds: [String]
    var allowedToolIds: [String]
    var aiDisclosureMode: String
    var freeTextFilterAction: String
    var crisisEscalationEnabled: Bool?
    var aiLogRetentionDays: Int
    var updatedAt: String?

    enum CodingKeys: String, CodingKey {
        case deniedCapabilities, deniedToolIds, allowedToolIds
        case aiDisclosureMode, freeTextFilterAction, crisisEscalationEnabled
        case aiLogRetentionDays, updatedAt
    }

    init(
        deniedCapabilities: [String] = [],
        deniedToolIds: [String] = [],
        allowedToolIds: [String] = [],
        aiDisclosureMode: String = "",
        freeTextFilterAction: String = "",
        crisisEscalationEnabled: Bool? = nil,
        aiLogRetentionDays: Int = 0,
        updatedAt: String? = nil
    ) {
        self.deniedCapabilities = deniedCapabilities
        self.deniedToolIds = deniedToolIds
        self.allowedToolIds = allowedToolIds
        self.aiDisclosureMode = aiDisclosureMode
        self.freeTextFilterAction = freeTextFilterAction
        self.crisisEscalationEnabled = crisisEscalationEnabled
        self.aiLogRetentionDays = aiLogRetentionDays
        self.updatedAt = updatedAt
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        deniedCapabilities = try c.decodeIfPresent([String].self, forKey: .deniedCapabilities) ?? []
        deniedToolIds = try c.decodeIfPresent([String].self, forKey: .deniedToolIds) ?? []
        allowedToolIds = try c.decodeIfPresent([String].self, forKey: .allowedToolIds) ?? []
        aiDisclosureMode = try c.decodeIfPresent(String.self, forKey: .aiDisclosureMode) ?? ""
        freeTextFilterAction = try c.decodeIfPresent(String.self, forKey: .freeTextFilterAction) ?? ""
        crisisEscalationEnabled = try c.decodeIfPresent(Bool.self, forKey: .crisisEscalationEnabled)
        aiLogRetentionDays = try c.decodeIfPresent(Int.self, forKey: .aiLogRetentionDays) ?? 0
        updatedAt = try c.decodeIfPresent(String.self, forKey: .updatedAt)
    }
}

struct ContentToolSettings: Codable, Equatable {
    var allowedToolIds: [String]
    var studentResetAllowed: Bool
    var maxInstancesPerItem: Int
    var monthlyAiTokenBudget: Int64
    var dailyAiCallsPerUser: Int
    var linkIngestionMode: String
    var linkHostAllowlist: [String]
    var gradeLinksAllowed: Bool
    var updatedAt: String?
    /// CT.M9 — org policy + kill snapshot attached to settings GET.
    var policy: ToolGovernancePolicy?
    var killedToolIds: [String]
    var killedCapabilities: [String]
    var killAllAI: Bool

    enum CodingKeys: String, CodingKey {
        case allowedToolIds, studentResetAllowed, maxInstancesPerItem
        case monthlyAiTokenBudget, dailyAiCallsPerUser, linkIngestionMode
        case linkHostAllowlist, gradeLinksAllowed, updatedAt
        case policy, killedToolIds, killedCapabilities, killAllAI
    }

    init(
        allowedToolIds: [String] = [],
        studentResetAllowed: Bool = false,
        maxInstancesPerItem: Int = 0,
        monthlyAiTokenBudget: Int64 = 0,
        dailyAiCallsPerUser: Int = 0,
        linkIngestionMode: String = "off",
        linkHostAllowlist: [String] = [],
        gradeLinksAllowed: Bool = false,
        updatedAt: String? = nil,
        policy: ToolGovernancePolicy? = nil,
        killedToolIds: [String] = [],
        killedCapabilities: [String] = [],
        killAllAI: Bool = false
    ) {
        self.allowedToolIds = allowedToolIds
        self.studentResetAllowed = studentResetAllowed
        self.maxInstancesPerItem = maxInstancesPerItem
        self.monthlyAiTokenBudget = monthlyAiTokenBudget
        self.dailyAiCallsPerUser = dailyAiCallsPerUser
        self.linkIngestionMode = linkIngestionMode
        self.linkHostAllowlist = linkHostAllowlist
        self.gradeLinksAllowed = gradeLinksAllowed
        self.updatedAt = updatedAt
        self.policy = policy
        self.killedToolIds = killedToolIds
        self.killedCapabilities = killedCapabilities
        self.killAllAI = killAllAI
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        allowedToolIds = try c.decodeIfPresent([String].self, forKey: .allowedToolIds) ?? []
        studentResetAllowed = try c.decodeIfPresent(Bool.self, forKey: .studentResetAllowed) ?? false
        maxInstancesPerItem = try c.decodeIfPresent(Int.self, forKey: .maxInstancesPerItem) ?? 0
        monthlyAiTokenBudget = try c.decodeIfPresent(Int64.self, forKey: .monthlyAiTokenBudget) ?? 0
        dailyAiCallsPerUser = try c.decodeIfPresent(Int.self, forKey: .dailyAiCallsPerUser) ?? 0
        linkIngestionMode = try c.decodeIfPresent(String.self, forKey: .linkIngestionMode) ?? "off"
        linkHostAllowlist = try c.decodeIfPresent([String].self, forKey: .linkHostAllowlist) ?? []
        gradeLinksAllowed = try c.decodeIfPresent(Bool.self, forKey: .gradeLinksAllowed) ?? false
        updatedAt = try c.decodeIfPresent(String.self, forKey: .updatedAt)
        policy = try c.decodeIfPresent(ToolGovernancePolicy.self, forKey: .policy)
        killedToolIds = try c.decodeIfPresent([String].self, forKey: .killedToolIds) ?? []
        killedCapabilities = try c.decodeIfPresent([String].self, forKey: .killedCapabilities) ?? []
        killAllAI = try c.decodeIfPresent(Bool.self, forKey: .killAllAI) ?? false
    }
}

struct ContentToolModerationAction: Codable, Equatable {
    var id: String
    var instanceId: String
    var action: String
    var category: String?
    var reason: String?
    var contentPath: String?
    var createdAt: String

    enum CodingKeys: String, CodingKey {
        case id, instanceId, action, category, reason, contentPath, createdAt
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        id = try c.decodeIfPresent(String.self, forKey: .id) ?? ""
        instanceId = try c.decodeIfPresent(String.self, forKey: .instanceId) ?? ""
        action = try c.decodeIfPresent(String.self, forKey: .action) ?? ""
        category = try c.decodeIfPresent(String.self, forKey: .category)
        reason = try c.decodeIfPresent(String.self, forKey: .reason)
        contentPath = try c.decodeIfPresent(String.self, forKey: .contentPath)
        createdAt = try c.decodeIfPresent(String.self, forKey: .createdAt) ?? ""
    }
}

struct ContentToolReportBody: Encodable {
    var category: String?
    var reason: String?
    var contentPath: String?
}

struct ContentToolModerateBody: Encodable {
    var action: String
    var category: String?
    var reason: String?
    var contentPath: String?
}

struct ContentToolModerationListResponse: Codable, Equatable {
    var items: [ContentToolModerationAction]
}

struct ContentToolConformanceTool: Codable, Equatable {
    var toolId: String
    var ok: Bool
    var wcagLevel: String
    var errors: [String]?

    enum CodingKeys: String, CodingKey {
        case toolId, ok, wcagLevel, errors
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        toolId = try c.decodeIfPresent(String.self, forKey: .toolId) ?? ""
        ok = try c.decodeIfPresent(Bool.self, forKey: .ok) ?? true
        wcagLevel = try c.decodeIfPresent(String.self, forKey: .wcagLevel) ?? ""
        errors = try c.decodeIfPresent([String].self, forKey: .errors)
    }
}

struct ContentToolConformanceResponse: Codable, Equatable {
    var ok: Bool
    var tools: [ContentToolConformanceTool]

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        ok = try c.decodeIfPresent(Bool.self, forKey: .ok) ?? true
        tools = try c.decodeIfPresent([ContentToolConformanceTool].self, forKey: .tools) ?? []
    }

    enum CodingKeys: String, CodingKey { case ok, tools }
}

struct ContentToolFilterFlagsResponse: Codable, Equatable {
    var items: [JSONValue]
}

struct SaveToolStateBody: Encodable {
    var revision: Int64
    var state: JSONValue
    var stateJson: JSONValue?
}

struct RunToolActionBody: Encodable {
    var input: JSONValue
    var idempotencyKey: String?
}

struct RunToolActionResponse: Codable, Equatable {
    var result: JSONValue?
    var state: ToolStateEnvelope?
}

struct RevisionConflictBody: Codable, Equatable {
    var error: String?
    var current: ToolStateEnvelope
}

struct StateTooLargeBody: Codable, Equatable {
    var error: String?
    var maxBytes: Int64
}

/// CT.M6 / CT.8 — AI consent response for content-tool composers.
struct ContentToolAIConsent: Codable, Equatable {
    var aiDisclosureMode: String
    var decision: String?
    var decidedAt: String?

    enum CodingKeys: String, CodingKey {
        case aiDisclosureMode, decision, decidedAt
    }

    init(aiDisclosureMode: String = "acknowledge", decision: String? = nil, decidedAt: String? = nil) {
        self.aiDisclosureMode = aiDisclosureMode
        self.decision = decision
        self.decidedAt = decidedAt
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        aiDisclosureMode = try c.decodeIfPresent(String.self, forKey: .aiDisclosureMode) ?? "acknowledge"
        decision = try c.decodeIfPresent(String.self, forKey: .decision)
        decidedAt = try c.decodeIfPresent(String.self, forKey: .decidedAt)
    }
}

struct ContentToolAIConsentBody: Encodable {
    var toolId: String?
    var decision: String
}
