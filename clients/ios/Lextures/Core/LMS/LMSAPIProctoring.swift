import Foundation

extension LMSAPI {
    static func fetchQuizProctoringConfig(
        courseCode: String,
        itemId: String,
        accessToken: String
    ) async -> QuizProctoringConfig? {
        do {
            let (data, response) = try await client.request(
                path: "/api/v1/courses/\(encodePath(courseCode))/quizzes/\(encodePath(itemId))/proctoring-config",
                authorized: true,
                accessToken: accessToken
            )
            if response.statusCode == 204 { return nil }
            guard (200 ... 299).contains(response.statusCode) else { return nil }
            return try decode(QuizProctoringConfig.self, from: data)
        } catch {
            return nil
        }
    }
}
