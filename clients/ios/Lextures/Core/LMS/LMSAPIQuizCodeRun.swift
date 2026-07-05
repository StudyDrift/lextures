import Foundation

extension LMSAPI {
    static func postQuizQuestionRun(
        courseCode: String,
        itemId: String,
        attemptId: String,
        questionId: String,
        code: String,
        languageId: Int?,
        accessToken: String
    ) async throws -> QuizCodeRunResponse {
        let body = QuizCodeRunRequest(code: code, languageId: languageId)
        let path = [
            "/api/v1/courses/\(encodePath(courseCode))/quizzes/\(encodePath(itemId))",
            "attempts/\(encodePath(attemptId))/questions/\(encodePath(questionId))/run",
        ].joined(separator: "/")
        let (data, response) = try await client.request(
            path: path,
            method: "POST",
            body: body,
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(QuizCodeRunResponse.self, from: data)
    }
}
