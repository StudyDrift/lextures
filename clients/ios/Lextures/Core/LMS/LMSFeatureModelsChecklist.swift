import Foundation

// MARK: - Course checklist (CC.9)

enum ChecklistStatus: String, Codable, Equatable {
    case done
    case todo
    case inProgress = "in_progress"
    case unknown
    case notApplicable = "not_applicable"
}

enum ChecklistTier: String, Codable, Equatable {
    case essential
    case recommended
}

enum ChecklistDismissReason: String, Codable, Equatable, CaseIterable, Identifiable {
    case notApplicable = "not_applicable"
    case doneElsewhere = "done_elsewhere"
    case disagree
    case later
    case other

    var id: String { rawValue }

    var labelKey: String {
        switch self {
        case .notApplicable: return "mobile.checklist.dismissReason.not_applicable"
        case .doneElsewhere: return "mobile.checklist.dismissReason.done_elsewhere"
        case .disagree: return "mobile.checklist.dismissReason.disagree"
        case .later: return "mobile.checklist.dismissReason.later"
        case .other: return "mobile.checklist.dismissReason.other"
        }
    }
}

struct ChecklistNavTarget: Codable, Equatable {
    var route: String
    var anchor: String?
    var entityKey: String?
}

struct ChecklistEvidenceRow: Codable, Equatable, Identifiable {
    var id: String { "\(label)|\(sublabel ?? "")|\(status)" }
    var label: String
    var sublabel: String?
    var status: String
    var target: ChecklistNavTarget?
}

struct ChecklistEvidence: Codable, Equatable {
    var columns: [String]
    var rows: [ChecklistEvidenceRow]
    var truncatedAt: Int?
}

struct ChecklistDismissal: Codable, Equatable {
    var dismissedAt: String
    var byUserId: String
    var byDisplayName: String
    var reason: String
    var note: String
}

struct ChecklistProgress: Codable, Equatable {
    var done: Int
    var total: Int
}

/// Optional assisted-fix primary action (CC.10 FR-5). Unknown kinds render nothing.
struct ChecklistAction: Codable, Equatable {
    var kind: String
    var labelKey: String
    var label: String
    var endpoint: String
    var requiresAi: Bool?
}

struct ChecklistItem: Codable, Equatable, Identifiable {
    var id: String
    var titleKey: String
    var title: String
    var whyKey: String
    var why: String
    var tier: ChecklistTier
    var status: String
    var detail: String?
    var progress: ChecklistProgress?
    var sources: [String]
    var helpRef: String?
    var target: ChecklistNavTarget?
    var evidence: ChecklistEvidence?
    var action: ChecklistAction?
    var dismissal: ChecklistDismissal?
}

struct ChecklistCategory: Codable, Equatable, Identifiable {
    var id: String
    var titleKey: String
    var title: String
    var items: [ChecklistItem]
}

struct CourseChecklistSummary: Codable, Equatable {
    var outstandingEssential: Int
    var outstandingTotal: Int
    var done: Int
    var total: Int
    var dismissed: Int
    var computedAt: String
    var stale: Bool
}

struct CourseChecklist: Codable, Equatable {
    var courseCode: String
    var engineVersion: Int
    var catalogVersion: String
    var computedAt: String
    var stale: Bool
    var evidenceTruncated: Bool
    var summary: CourseChecklistSummary
    var categories: [ChecklistCategory]
    var dismissed: [ChecklistItem]
}

struct ChecklistDismissBody: Encodable {
    var reason: String
    var note: String
}
