import Foundation

/// Course settings API (M13.1).
extension LMSAPI {
    static func updateCourse(
        courseCode: String,
        body: CourseUpdateRequest,
        accessToken: String
    ) async throws -> CourseSummary {
        let (data, _) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))",
            method: "PUT",
            body: body,
            authorized: true,
            accessToken: accessToken
        )
        return try decode(CourseSummary.self, from: data)
    }

    static func patchCourseMarkdownTheme(
        courseCode: String,
        body: CourseMarkdownThemePatch,
        accessToken: String
    ) async throws -> CourseSummary {
        let (data, _) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/markdown-theme",
            method: "PATCH",
            body: body,
            authorized: true,
            accessToken: accessToken
        )
        return try decode(CourseSummary.self, from: data)
    }

    static func saveCourseHeroImage(
        courseCode: String,
        imageUrl: String,
        accessToken: String
    ) async throws -> CourseSummary {
        let (data, _) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/hero-image",
            method: "PUT",
            body: CourseHeroImageURLRequest(imageUrl: imageUrl),
            authorized: true,
            accessToken: accessToken
        )
        return try decode(CourseSummary.self, from: data)
    }

    static func saveCourseHeroPosition(
        courseCode: String,
        objectPosition: String?,
        accessToken: String
    ) async throws -> CourseSummary {
        let (data, _) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/hero-image",
            method: "PUT",
            body: CourseHeroPositionRequest(objectPosition: objectPosition),
            authorized: true,
            accessToken: accessToken
        )
        return try decode(CourseSummary.self, from: data)
    }

    static func generateCourseImage(
        courseCode: String,
        prompt: String,
        accessToken: String
    ) async throws -> CourseGenerateImageResponse {
        let (data, _) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/generate-image",
            method: "POST",
            body: CourseGenerateImageRequest(prompt: prompt),
            authorized: true,
            accessToken: accessToken
        )
        return try decode(CourseGenerateImageResponse.self, from: data)
    }

    static func uploadCourseFile(
        courseCode: String,
        fileName: String,
        mimeType: String,
        fileData: Data,
        accessToken: String
    ) async throws -> CourseFileUploadResponse {
        let (data, _) = try await client.uploadMultipart(
            path: "/api/v1/courses/\(encodePath(courseCode))/course-files",
            fieldName: "file",
            fileName: fileName,
            mimeType: mimeType,
            fileData: fileData,
            accessToken: accessToken
        )
        return try decode(CourseFileUploadResponse.self, from: data)
    }
}
