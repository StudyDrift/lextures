import Foundation

/// Grading agent settings API (M13.6).
extension LMSAPI {
    static func fetchCourseGradingAgents(
        courseCode: String,
        accessToken: String
    ) async throws -> CourseGradingAgentsListResponse {
        let (data, response) = try await client.requestRaw(
            path: "/api/v1/courses/\(encodePath(courseCode))/grader-agents",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(CourseGradingAgentsListResponse.self, from: data)
    }

    static func fetchGraderAgentTemplates(
        courseCode: String,
        accessToken: String
    ) async throws -> GraderAgentTemplatesListResponse {
        let (data, response) = try await client.requestRaw(
            path: "/api/v1/courses/\(encodePath(courseCode))/grader-agent-templates",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(GraderAgentTemplatesListResponse.self, from: data)
    }

    static func fetchGraderAgentConfig(
        courseCode: String,
        itemId: String,
        itemKind: String,
        accessToken: String
    ) async throws -> GraderAgentConfig? {
        let path = CourseGradingAgentsLogic.graderAgentPath(
            courseCode: courseCode,
            itemId: itemId,
            itemKind: itemKind
        )
        let (data, response) = try await client.requestRaw(
            path: path,
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(GraderAgentConfigEnvelope.self, from: data).config
    }

    static func fetchGraderAgentTemplate(
        courseCode: String,
        templateId: String,
        accessToken: String
    ) async throws -> GraderAgentTemplateDetail {
        let (data, response) = try await client.requestRaw(
            path: "/api/v1/courses/\(encodePath(courseCode))/grader-agent-templates/\(encodePath(templateId))",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(GraderAgentTemplateDetailEnvelope.self, from: data).template
    }

    static func putGraderAgentConfig(
        courseCode: String,
        itemId: String,
        itemKind: String,
        body: PutGraderAgentConfigBody,
        accessToken: String
    ) async throws -> GraderAgentConfig {
        let path = CourseGradingAgentsLogic.graderAgentPath(
            courseCode: courseCode,
            itemId: itemId,
            itemKind: itemKind
        )
        let (data, response) = try await client.requestRaw(
            path: path,
            method: "PUT",
            bodyData: try JSONEncoder().encode(body),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(PutGraderAgentConfigResponse.self, from: data).config
    }

    static func deleteGraderAgentConfig(
        courseCode: String,
        itemId: String,
        itemKind: String,
        accessToken: String
    ) async throws {
        let path = CourseGradingAgentsLogic.graderAgentPath(
            courseCode: courseCode,
            itemId: itemId,
            itemKind: itemKind
        )
        let (data, response) = try await client.requestRaw(
            path: path,
            method: "DELETE",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
    }
}
