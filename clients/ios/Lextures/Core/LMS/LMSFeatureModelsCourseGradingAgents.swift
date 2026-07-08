import Foundation

/// Grading agent settings models (M13.6).
struct CourseGradingAgentSummary: Codable, Identifiable, Hashable {
    var id: String
    var itemId: String
    var itemKind: String?
    var assignmentTitle: String
    var assignmentArchived: Bool
    var status: String
    var autoGradeNew: Bool
    var hasWorkflowGraph: Bool
    var updatedAt: String
    var reviewCount: Int?
}

struct CourseGradingAgentsListResponse: Codable {
    var agents: [CourseGradingAgentSummary]
}

struct GraderAgentTemplateSummary: Codable, Identifiable, Hashable {
    var id: String
    var name: String
    var isBuiltin: Bool?
    var updatedAt: String
}

struct GraderAgentTemplatesListResponse: Codable {
    var templates: [GraderAgentTemplateSummary]
}

struct GraderAgentConfig: Codable, Hashable {
    var id: String?
    var prompt: String
    var includeAssignmentContent: Bool
    var includeRubric: Bool
    var status: String
    var autoGradeNew: Bool?
    var postPolicy: String?
    var confidenceFloor: Double?
    var modelId: String?
    var updatedAt: String?
    var workflowGraph: JSONValue?
}

struct GraderAgentConfigEnvelope: Decodable {
    var config: GraderAgentConfig?
}

struct GraderAgentTemplateDetail: Codable {
    var id: String
    var name: String
    var prompt: String
    var includeAssignmentContent: Bool
    var includeRubric: Bool
    var workflowGraph: JSONValue?
    var createdAt: String
    var updatedAt: String
}

struct GraderAgentTemplateDetailEnvelope: Decodable {
    var template: GraderAgentTemplateDetail
}

struct PutGraderAgentConfigBody: Encodable {
    var prompt: String
    var includeAssignmentContent: Bool
    var includeRubric: Bool
    var status: String
    var autoGradeNew: Bool
    var postPolicy: String
    var confidenceFloor: Double?
    var workflowGraph: JSONValue
}

struct PutGraderAgentConfigResponse: Decodable {
    var config: GraderAgentConfig
}
