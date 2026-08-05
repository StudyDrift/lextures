import Foundation

/// Course checklist endpoints (CC.2 / CC.9).
extension LMSAPI {
    private static func checklistBase(_ courseCode: String) -> String {
        "/api/v1/courses/\(encodePath(courseCode))/checklist"
    }

    static func fetchCourseChecklist(courseCode: String, accessToken: String) async throws -> CourseChecklist {
        let (data, _) = try await client.request(
            path: checklistBase(courseCode),
            authorized: true,
            accessToken: accessToken
        )
        return try decode(CourseChecklist.self, from: data)
    }

    static func fetchCourseChecklistSummary(
        courseCode: String,
        accessToken: String
    ) async throws -> CourseChecklistSummary {
        let (data, _) = try await client.request(
            path: "\(checklistBase(courseCode))/summary",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(CourseChecklistSummary.self, from: data)
    }

    static func refreshCourseChecklist(courseCode: String, accessToken: String) async throws -> CourseChecklist {
        let (data, _) = try await client.request(
            path: "\(checklistBase(courseCode))/refresh",
            method: "POST",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(CourseChecklist.self, from: data)
    }

    static func dismissChecklistItem(
        courseCode: String,
        itemID: String,
        reason: ChecklistDismissReason,
        note: String?,
        accessToken: String
    ) async throws -> ChecklistItem {
        let body = ChecklistDismissBody(
            reason: reason.rawValue,
            note: CourseChecklistLogic.clampedNote(note)
        )
        let (data, _) = try await client.request(
            path: "\(checklistBase(courseCode))/items/\(encodePath(itemID))/dismiss",
            method: "POST",
            body: body,
            authorized: true,
            accessToken: accessToken
        )
        return try decode(ChecklistItem.self, from: data)
    }

    static func restoreChecklistItem(
        courseCode: String,
        itemID: String,
        accessToken: String
    ) async throws -> ChecklistItem {
        let (data, _) = try await client.request(
            path: "\(checklistBase(courseCode))/items/\(encodePath(itemID))/restore",
            method: "POST",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(ChecklistItem.self, from: data)
    }

    static func recheckChecklistItem(
        courseCode: String,
        itemID: String,
        accessToken: String
    ) async throws -> ChecklistItem {
        let (data, _) = try await client.request(
            path: "\(checklistBase(courseCode))/items/\(encodePath(itemID))/recheck",
            method: "POST",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(ChecklistItem.self, from: data)
    }
}
