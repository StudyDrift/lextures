import Foundation

/// Course accessibility / alt-text API (M13.8).
extension LMSAPI {
    static func fetchCourseAccessibility(
        courseCode: String,
        accessToken: String
    ) async throws -> CourseAccessibilityInfo {
        let (data, response) = try await client.requestRaw(
            path: "/api/v1/courses/\(encodePath(courseCode))/accessibility",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(CourseAccessibilityInfo.self, from: data)
    }

    static func suggestAltText(
        courseCode: String,
        imageUrl: String,
        language: String,
        accessToken: String
    ) async throws -> AltTextSuggestion {
        let body = SuggestAltTextBody(imageUrl: imageUrl, language: language)
        let (data, response) = try await client.requestRaw(
            path: "/api/v1/courses/\(encodePath(courseCode))/alt-text/suggest",
            method: "POST",
            bodyData: try JSONEncoder().encode(body),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(AltTextSuggestion.self, from: data)
    }

    static func patchItemMarkdown(
        courseCode: String,
        itemId: String,
        kind: String,
        markdown: String,
        accessToken: String
    ) async throws -> ModuleItemDetail {
        guard let path = CourseAccessibilityReviewLogic.markdownPatchPath(
            courseCode: courseCode,
            itemId: itemId,
            kind: kind
        ) else {
            throw APIError.httpStatus(400, message: "Unsupported item kind")
        }
        let (data, response) = try await client.requestRaw(
            path: path,
            method: "PATCH",
            bodyData: try JSONEncoder().encode(PatchItemMarkdownBody(markdown: markdown)),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(ModuleItemDetail.self, from: data)
    }
}
