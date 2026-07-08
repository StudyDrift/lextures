import Foundation

/// Course grading agents list/edit helpers (M13.6).
enum CourseGradingAgentsLogic {
    struct GradableOption: Identifiable, Hashable {
        var id: String
        var label: String
        var kind: String
    }

    struct AgentDraft: Equatable, Hashable {
        var prompt: String
        var includeAssignmentContent: Bool
        var includeRubric: Bool
        var status: String
        var autoGradeNew: Bool
        var postPolicy: String
        var confidenceFloor: Double?
        var workflowGraph: JSONValue?
    }

    enum AgentStatus: String, CaseIterable, Identifiable {
        case draft
        case accepted
        case archived

        var id: String { rawValue }

        var labelKey: String {
            "mobile.courseSettings.gradingAgents.status.\(rawValue)"
        }
    }

    enum ValidationError: Equatable {
        case promptRequired
    }

    private static let workflowVersion = 1

    static func cacheKeyAgents(courseCode: String) -> String {
        "course:\(courseCode):grading-agents"
    }

    static func saveIdempotencyKey(courseCode: String, itemId: String, itemKind: String) -> String {
        "course-grading-agents:\(courseCode):\(itemKind):\(itemId):save"
    }

    static func deleteIdempotencyKey(courseCode: String, itemId: String, itemKind: String) -> String {
        "course-grading-agents:\(courseCode):\(itemKind):\(itemId):delete"
    }

    static func graderAgentPath(courseCode: String, itemId: String, itemKind: String) -> String {
        let collection = itemKind == "quiz" ? "quizzes" : "assignments"
        return "/api/v1/courses/\(courseCode)/\(collection)/\(itemId)/grader-agent"
    }

    static func normalizedItemKind(_ itemKind: String?) -> String {
        itemKind == "quiz" ? "quiz" : "assignment"
    }

    static func gradableOptions(
        from structure: [CourseStructureItem],
        excluding existingItemIds: Set<String>
    ) -> [GradableOption] {
        let byId = Dictionary(uniqueKeysWithValues: structure.map { ($0.id, $0) })
        let rows = structure.filter { item in
            (item.kind == "assignment" || item.kind == "quiz")
                && item.archived != true
                && !existingItemIds.contains(item.id)
        }
        let withLabels: [GradableOption] = rows.map { item in
            var moduleTitle = ""
            var parent = item.parentId.flatMap { byId[$0] }
            var guardIds = Set<String>()
            while let currentParent = parent, !guardIds.contains(currentParent.id) {
                guardIds.insert(currentParent.id)
                if currentParent.kind == "module" {
                    moduleTitle = currentParent.title
                    break
                }
                parent = currentParent.parentId.flatMap { byId[$0] }
            }
            let label = moduleTitle.isEmpty ? item.title : "\(moduleTitle) — \(item.title)"
            return GradableOption(id: item.id, label: label, kind: item.kind)
        }
        return withLabels.sorted { $0.label.localizedCaseInsensitiveCompare($1.label) == .orderedAscending }
    }

    static func draft(from config: GraderAgentConfig?) -> AgentDraft {
        AgentDraft(
            prompt: config?.prompt ?? "",
            includeAssignmentContent: config?.includeAssignmentContent ?? false,
            includeRubric: config?.includeRubric ?? false,
            status: config?.status ?? AgentStatus.draft.rawValue,
            autoGradeNew: config?.autoGradeNew ?? false,
            postPolicy: config?.postPolicy ?? "draft",
            confidenceFloor: config?.confidenceFloor,
            workflowGraph: config?.workflowGraph
        )
    }

    static func draft(from template: GraderAgentTemplateDetail) -> AgentDraft {
        AgentDraft(
            prompt: template.prompt,
            includeAssignmentContent: template.includeAssignmentContent,
            includeRubric: template.includeRubric,
            status: AgentStatus.draft.rawValue,
            autoGradeNew: false,
            postPolicy: "draft",
            confidenceFloor: nil,
            workflowGraph: template.workflowGraph
        )
    }

    static func isDirty(current: AgentDraft, baseline: AgentDraft) -> Bool {
        current != baseline
    }

    static func validateDraft(_ draft: AgentDraft) -> ValidationError? {
        draft.prompt.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty ? .promptRequired : nil
    }

    static func buildPutBody(current: AgentDraft, itemKind: String) -> PutGraderAgentConfigBody {
        let floor = current.confidenceFloor
        return PutGraderAgentConfigBody(
            prompt: current.prompt.trimmingCharacters(in: .whitespacesAndNewlines),
            includeAssignmentContent: current.includeAssignmentContent,
            includeRubric: current.includeRubric,
            status: current.status,
            autoGradeNew: current.autoGradeNew,
            postPolicy: current.postPolicy,
            confidenceFloor: floor.flatMap { $0 > 0 ? $0 : nil },
            workflowGraph: effectiveWorkflowGraph(
                stored: current.workflowGraph,
                itemKind: itemKind
            )
        )
    }

    static func effectiveWorkflowGraph(stored: JSONValue?, itemKind: String) -> JSONValue {
        if let stored, hasWorkflowNodes(stored) {
            return itemKind == "quiz" ? ensureQuizResponsesNode(stored) : stored
        }
        return defaultWorkflowGraph(itemKind: itemKind)
    }

    static func filteredAgents(
        _ agents: [CourseGradingAgentSummary],
        query: String
    ) -> [CourseGradingAgentSummary] {
        let trimmed = query.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else {
            return agents.sorted { $0.assignmentTitle.localizedCaseInsensitiveCompare($1.assignmentTitle) == .orderedAscending }
        }
        let needle = trimmed.lowercased()
        return agents
            .filter { agent in
                agent.assignmentTitle.lowercased().contains(needle)
                    || statusLabelKey(agent.status).lowercased().contains(needle)
            }
            .sorted { $0.assignmentTitle.localizedCaseInsensitiveCompare($1.assignmentTitle) == .orderedAscending }
    }

    static func statusLabelKey(_ status: String) -> String {
        AgentStatus(rawValue: status)?.labelKey ?? status
    }

    static func kindLabelKey(_ itemKind: String?) -> String? {
        normalizedItemKind(itemKind) == "quiz"
            ? "mobile.courseSettings.gradingAgents.quizBadge"
            : nil
    }

    private static func hasWorkflowNodes(_ graph: JSONValue) -> Bool {
        guard case .object(let object) = graph,
              case .array(let nodes)? = object["nodes"] else {
            return false
        }
        return !nodes.isEmpty
    }

    private static func ensureQuizResponsesNode(_ graph: JSONValue) -> JSONValue {
        guard case .object(var object) = graph,
              case .array(let nodes) = object["nodes"] else {
            return defaultWorkflowGraph(itemKind: "quiz")
        }
        let hasQuizResponses = nodes.contains { node in
            guard case .object(let nodeObject) = node,
                  case .string(let type)? = nodeObject["type"] else {
                return false
            }
            return type == "quizResponses"
        }
        guard !hasQuizResponses else { return graph }
        let quizNode: JSONValue = .object([
            "id": .string("quizResponses"),
            "type": .string("quizResponses"),
            "position": .object(["x": .number(-420), "y": .number(0)]),
            "data": .object([:]),
        ])
        object["nodes"] = .array([quizNode] + nodes)
        return .object(object)
    }

    static func defaultWorkflowGraph(itemKind: String) -> JSONValue {
        let outputNode: JSONValue = .object([
            "id": .string("output"),
            "type": .string("output"),
            "position": .object(["x": .number(0), "y": .number(0)]),
            "data": .object([:]),
        ])
        if itemKind == "quiz" {
            let quizNode: JSONValue = .object([
                "id": .string("quizResponses"),
                "type": .string("quizResponses"),
                "position": .object(["x": .number(-420), "y": .number(0)]),
                "data": .object([:]),
            ])
            return .object([
                "version": .number(Double(workflowVersion)),
                "nodes": .array([quizNode, outputNode]),
                "edges": .array([]),
            ])
        }
        return .object([
            "version": .number(Double(workflowVersion)),
            "nodes": .array([outputNode]),
            "edges": .array([]),
        ])
    }
}
