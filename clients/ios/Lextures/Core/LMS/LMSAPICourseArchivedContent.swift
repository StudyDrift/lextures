import Foundation

/// Archived module item list + restore API (M13.12).
extension LMSAPI {
    static func fetchCourseArchivedStructure(
        courseCode: String,
        accessToken: String
    ) async throws -> [CourseStructureItem] {
        let (data, _) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/structure/archived",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(CourseStructureResponse.self, from: data).items
    }

    static func unarchiveCourseStructureItem(
        courseCode: String,
        itemId: String,
        accessToken: String
    ) async throws -> CourseStructureItem {
        let (data, _) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/structure/items/\(encodePath(itemId))",
            method: "PATCH",
            body: CourseStructureItemPatch(archived: false),
            authorized: true,
            accessToken: accessToken
        )
        return try decode(CourseStructureItem.self, from: data)
    }
}
