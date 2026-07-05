import Foundation

/// Course evaluation status, submit, and results (M7.7).
extension LMSAPI {
    static func fetchEvaluationStatus(courseCode: String, accessToken: String) async throws -> EvaluationStatus {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/evaluations/status",
            authorized: true,
            accessToken: accessToken
        )
        if response.statusCode == 404 {
            throw APIError.httpStatus(404, message: L.text("mobile.evaluations.unavailable"))
        }
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(EvaluationStatus.self, from: data)
    }

    static func submitEvaluation(
        courseCode: String,
        windowId: String,
        answers: [String: String],
        accessToken: String
    ) async throws {
        let body = EvaluationSubmitBody(answers: answers)
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/evaluations/\(encodePath(windowId))/submit",
            method: "POST",
            body: body,
            authorized: true,
            accessToken: accessToken
        )
        if response.statusCode == 409 {
            throw APIError.httpStatus(409, message: L.text("mobile.evaluations.alreadySubmitted"))
        }
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
    }

    static func fetchEvaluationResults(courseCode: String, accessToken: String) async throws -> EvaluationResults {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/evaluations/results",
            authorized: true,
            accessToken: accessToken
        )
        if response.statusCode == 404 {
            throw APIError.httpStatus(404, message: L.text("mobile.evaluations.noResults"))
        }
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(EvaluationResults.self, from: data)
    }
}
