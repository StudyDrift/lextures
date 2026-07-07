import Foundation

/// Course JSON backup/restore API (M13.10).
extension LMSAPI {
    static func fetchCourseExport(
        courseCode: String,
        accessToken: String
    ) async throws -> [String: JSONValue] {
        let (data, _) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/export",
            authorized: true,
            accessToken: accessToken
        )
        return try decode([String: JSONValue].self, from: data)
    }

    static func postCourseImport(
        courseCode: String,
        mode: CourseImportExportLogic.ImportMode,
        export: [String: JSONValue],
        accessToken: String
    ) async throws {
        let body = CourseImportRequestBody(mode: mode.rawValue, export: export)
        _ = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/import",
            method: "POST",
            body: body,
            authorized: true,
            accessToken: accessToken
        )
    }
}

private struct CourseImportRequestBody: Encodable {
    let mode: String
    let export: [String: JSONValue]
}
