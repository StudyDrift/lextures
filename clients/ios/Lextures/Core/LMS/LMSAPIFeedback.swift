import Foundation

/// Product feedback submit endpoint (FB3).
extension LMSAPI {
    static func submitFeedback(body: SubmitFeedbackRequest, accessToken: String) async throws -> SubmitFeedbackResponse {
        let (data, response) = try await client.request(
            path: "/api/v1/feedback",
            method: "POST",
            body: body,
            authorized: true,
            accessToken: accessToken
        )
        guard response.statusCode == 201 else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(SubmitFeedbackResponse.self, from: data)
    }
}
