import Foundation

/// Course create API (M11.5).
extension LMSAPI {
    static func createCourse(
        body: CreateCourseRequest,
        accessToken: String
    ) async throws -> CourseSummary {
        let (data, _) = try await client.request(
            path: "/api/v1/courses",
            method: "POST",
            body: body,
            authorized: true,
            accessToken: accessToken
        )
        return try decode(CourseSummary.self, from: data)
    }

    static func patchCourseSyllabus(
        courseCode: String,
        body: PatchCourseSyllabusRequest,
        accessToken: String
    ) async throws -> SyllabusPayload {
        let (data, _) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/syllabus",
            method: "PATCH",
            body: body,
            authorized: true,
            accessToken: accessToken
        )
        return try decode(SyllabusPayload.self, from: data)
    }

    static func createCourseModule(
        courseCode: String,
        title: String,
        accessToken: String
    ) async throws -> CourseStructureItem {
        let (data, _) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/structure/modules",
            method: "POST",
            body: CreateCourseModuleRequest(title: title),
            authorized: true,
            accessToken: accessToken
        )
        return try decode(CourseStructureItem.self, from: data)
    }

    static func fetchOrgTerms(orgId: String, accessToken: String) async throws -> [OrgTerm] {
        let (data, _) = try await client.request(
            path: "/api/v1/orgs/\(encodePath(orgId))/terms",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(OrgTermsResponse.self, from: data).terms ?? []
    }

    static func createModuleAssignment(
        courseCode: String,
        moduleId: String,
        title: String,
        accessToken: String
    ) async throws -> CourseStructureItem {
        let (data, _) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/structure/modules/\(encodePath(moduleId))/assignments",
            method: "POST",
            body: CreateModuleItemRequest(title: title),
            authorized: true,
            accessToken: accessToken
        )
        return try decode(CourseStructureItem.self, from: data)
    }

    static func createModuleQuiz(
        courseCode: String,
        moduleId: String,
        title: String,
        accessToken: String
    ) async throws -> CourseStructureItem {
        let (data, _) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/structure/modules/\(encodePath(moduleId))/quizzes",
            method: "POST",
            body: CreateModuleItemRequest(title: title),
            authorized: true,
            accessToken: accessToken
        )
        return try decode(CourseStructureItem.self, from: data)
    }
}
