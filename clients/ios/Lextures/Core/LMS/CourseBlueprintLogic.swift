import Foundation

/// District blueprint course helpers (M13.11).
enum CourseBlueprintLogic {
    static let globalAdminPermission = "global:app:rbac:manage"
    static let orgUnitsAdminPermission = "tenant:org:units:admin"

    enum BlueprintRole: Equatable {
        case master
        case child(parentCode: String)
        case none
    }

    static func canManageBlueprint(course: CourseSummary, permissions: [String]) -> Bool {
        guard course.orgId?.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty == false else {
            return false
        }
        return permissions.contains(globalAdminPermission)
            || permissions.contains(orgUnitsAdminPermission)
    }

    static func blueprintRole(for course: CourseSummary) -> BlueprintRole {
        if course.isBlueprint == true { return .master }
        let parent = course.blueprintParentCourseCode?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        if !parent.isEmpty { return .child(parentCode: parent) }
        return .none
    }

    static func shouldLoadBlueprintDetails(course: CourseSummary, canManage: Bool) -> Bool {
        canManage && course.isBlueprint == true
    }

    static func cacheKeyBlueprintData(courseCode: String) -> String {
        "course:\(courseCode):blueprint"
    }

    static func formatSyncAt(_ iso: String?) -> String {
        let formatted = DateFormatting.formatDateTime(iso)
        return formatted.isEmpty ? L.text("mobile.emDash") : formatted
    }

    static func pushDisabledReason(isOnline: Bool, childCount: Int) -> String? {
        if !isOnline {
            return L.text("mobile.courseSettings.blueprint.offlinePushDisabled")
        }
        if childCount == 0 {
            return L.text("mobile.courseSettings.blueprint.noChildrenPushDisabled")
        }
        return nil
    }

    static func mutationsDisabledReason(isOnline: Bool) -> String? {
        guard !isOnline else { return nil }
        return L.text("mobile.courseSettings.blueprint.offlineMutationsDisabled")
    }

    static func syncHistorySummary(success: Int, total: Int, errors: Int) -> String {
        L.format(
            "mobile.courseSettings.blueprint.syncHistoryRow",
            "\(success)",
            "\(total)",
            "\(errors)"
        )
    }

    static func pushResultSummary(success: Int, total: Int, errors: Int) -> String {
        L.format(
            "mobile.courseSettings.blueprint.pushResult",
            "\(success)",
            "\(total)",
            "\(errors)"
        )
    }

    static func userFacingError(_ error: Error) -> String {
        if let apiError = error as? APIError {
            return apiError.errorDescription ?? L.text("mobile.courseSettings.blueprint.genericError")
        }
        return error.localizedDescription
    }
}

struct BlueprintChildRow: Codable, Identifiable, Hashable {
    var courseCode: String
    var title: String
    var lastSyncAt: String?

    var id: String { courseCode }
}

struct BlueprintPushDetailRow: Codable, Hashable {
    var courseCode: String?
    var ok: Bool?
    var error: String?
}

struct BlueprintPushResult: Codable, Hashable {
    var childrenTotal: Int
    var childrenSuccess: Int
    var childrenError: Int
    var detail: [BlueprintPushDetailRow]
}

struct BlueprintSyncLogRow: Codable, Identifiable, Hashable {
    var id: String
    var triggeredBy: String
    var triggeredAt: String
    var childrenTotal: Int
    var childrenSuccess: Int
    var childrenError: Int
}

struct BlueprintCachedPayload: Codable, Hashable {
    var children: [BlueprintChildRow]
    var syncLogs: [BlueprintSyncLogRow]
}

struct BlueprintChildrenResponse: Decodable {
    var children: [BlueprintChildRow]?
}

struct BlueprintSyncLogsResponse: Decodable {
    var logs: [BlueprintSyncLogRow]?
}

struct BlueprintPatchRequest: Encodable {
    var isBlueprint: Bool
}

struct BlueprintLinkChildRequest: Encodable {
    var childCourseCode: String
}
