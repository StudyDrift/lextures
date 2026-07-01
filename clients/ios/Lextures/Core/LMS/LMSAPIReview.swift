import Foundation

extension LMSAPI {
    // MARK: - Spaced repetition / review (M8.1)

    static func fetchLearnerReviewQueue(
        userId: String,
        accessToken: String,
        limit: Int = ReviewLogic.prefetchLimit,
        offset: Int = 0
    ) async throws -> ReviewQueueResponse {
        var components = URLComponents()
        components.queryItems = [
            URLQueryItem(name: "limit", value: String(limit)),
            URLQueryItem(name: "offset", value: String(offset)),
        ]
        let queryString = components.percentEncodedQuery ?? ""
        let (data, response) = try await client.request(
            path: "/api/v1/learners/\(encodePath(userId))/review-queue?\(queryString)",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(ReviewQueueResponse.self, from: data)
    }

    static func fetchLearnerReviewStats(userId: String, accessToken: String) async throws -> ReviewStats {
        let (data, response) = try await client.request(
            path: "/api/v1/learners/\(encodePath(userId))/review-stats",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(ReviewStats.self, from: data)
    }

    static func postLearnerSrsReview(
        userId: String,
        body: SrsReviewSubmitBody,
        accessToken: String,
        idempotencyKey: String? = nil
    ) async throws -> SrsReviewSubmitResponse {
        let (data, response) = try await client.request(
            path: "/api/v1/learners/\(encodePath(userId))/review",
            method: "POST",
            body: body,
            authorized: true,
            accessToken: accessToken,
            idempotencyKey: idempotencyKey
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(SrsReviewSubmitResponse.self, from: data)
    }

    static func fetchLearnerRecommendations(
        userId: String,
        courseId: String,
        surface: String,
        accessToken: String,
        limit: Int = 5
    ) async throws -> LearnerRecommendationsResponse {
        var components = URLComponents()
        components.queryItems = [
            URLQueryItem(name: "courseId", value: courseId),
            URLQueryItem(name: "surface", value: surface),
            URLQueryItem(name: "limit", value: String(limit)),
        ]
        let queryString = components.percentEncodedQuery ?? ""
        let (data, response) = try await client.request(
            path: "/api/v1/learners/\(encodePath(userId))/recommendations?\(queryString)",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(LearnerRecommendationsResponse.self, from: data)
    }
}
