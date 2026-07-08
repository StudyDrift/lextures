import Foundation

/// Learner profile read + LP08 control endpoints (LP10).
extension LMSAPI {
    static func fetchLearnerProfile(accessToken: String) async throws -> LearnerProfile {
        let (data, response) = try await client.request(
            path: "/api/v1/me/learner-profile",
            authorized: true,
            accessToken: accessToken
        )
        if response.statusCode == 404 {
            throw APIError.httpStatus(404, message: L.text("mobile.learnerProfile.unavailable"))
        }
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        let envelope = try decode(LearnerProfileResponse.self, from: data)
        return envelope.profile ?? LearnerProfile(status: "insufficient_data", facets: [])
    }

    static func fetchLearnerProfileFacet(
        facetKey: LearnerProfileFacetKey,
        accessToken: String
    ) async throws -> LearnerProfileFacetDetailResponse? {
        let (data, response) = try await client.request(
            path: "/api/v1/me/learner-profile/facets/\(encodePath(facetKey))",
            authorized: true,
            accessToken: accessToken
        )
        if response.statusCode == 404 { return nil }
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(LearnerProfileFacetDetailResponse.self, from: data)
    }

    static func fetchLearnerProfileFacetEvidence(
        facetKey: LearnerProfileFacetKey,
        accessToken: String
    ) async throws -> LearnerProfileEvidenceMap {
        let (data, response) = try await client.request(
            path: "/api/v1/me/learner-profile/facets/\(encodePath(facetKey))/evidence",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(LearnerProfileEvidenceMap.self, from: data)
    }

    static func pauseLearnerProfile(accessToken: String) async throws -> String {
        try await postLearnerProfileControl(path: "/api/v1/me/learner-profile/pause", accessToken: accessToken)
    }

    static func resumeLearnerProfile(accessToken: String) async throws -> String {
        try await postLearnerProfileControl(path: "/api/v1/me/learner-profile/resume", accessToken: accessToken)
    }

    static func resetLearnerProfile(accessToken: String) async throws -> String {
        try await postLearnerProfileControl(path: "/api/v1/me/learner-profile/reset", accessToken: accessToken)
    }

    static func exportLearnerProfile(accessToken: String) async throws -> Data {
        let (data, response) = try await client.request(
            path: "/api/v1/me/learner-profile/export",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return data
    }

    private struct EmptyControlBody: Encodable {}

    private static func postLearnerProfileControl(path: String, accessToken: String) async throws -> String {
        let (data, response) = try await client.request(
            path: path,
            method: "POST",
            body: EmptyControlBody(),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return (try? decode(LearnerProfileControlResponse.self, from: data)).flatMap(\.status) ?? "ok"
    }
}