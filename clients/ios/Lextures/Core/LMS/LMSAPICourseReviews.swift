import Foundation

/// Course review eligibility and submit (M9.3).
extension LMSAPI {
    static func fetchReviewEligibility(
        courseCode: String,
        accessToken: String
    ) async throws -> ReviewEligibility {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/reviews/eligibility",
            authorized: true,
            accessToken: accessToken
        )
        if response.statusCode == 404 {
            throw APIError.httpStatus(404, message: L.text("mobile.reviews.unavailable"))
        }
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(ReviewEligibility.self, from: data)
    }

    static func submitCourseReview(
        courseCode: String,
        rating: Int,
        reviewText: String?,
        accessToken: String
    ) async throws -> SubmittedCourseReview {
        let body = SubmitCourseReviewRequest(
            rating: rating,
            reviewText: reviewText?.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty == true
                ? nil
                : reviewText?.trimmingCharacters(in: .whitespacesAndNewlines)
        )
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/reviews",
            method: "POST",
            body: body,
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(SubmittedCourseReview.self, from: data)
    }
}