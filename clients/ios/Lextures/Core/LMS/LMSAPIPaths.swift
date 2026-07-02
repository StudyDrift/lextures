import Foundation

/// Adaptive learning paths and recommendations (M8.2).
extension LMSAPI {
    static func fetchCatalogPaths(
        query: String = "",
        sort: String = "",
        accessToken: String? = nil
    ) async throws -> [CatalogPathSummary] {
        var components = URLComponents()
        var items: [URLQueryItem] = []
        let trimmed = query.trimmingCharacters(in: .whitespacesAndNewlines)
        if !trimmed.isEmpty { items.append(URLQueryItem(name: "q", value: trimmed)) }
        let sortTrimmed = sort.trimmingCharacters(in: .whitespacesAndNewlines)
        if !sortTrimmed.isEmpty { items.append(URLQueryItem(name: "sort", value: sortTrimmed)) }
        components.queryItems = items.isEmpty ? nil : items
        let queryString = components.percentEncodedQuery.map { "?\($0)" } ?? ""
        let (data, response) = try await client.request(
            path: "/api/v1/catalog/paths\(queryString)",
            authorized: accessToken != nil,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(CatalogPathsListResponse.self, from: data).paths ?? []
    }

    static func fetchCatalogPathDetail(slug: String, accessToken: String? = nil) async throws -> LearningPathDetail? {
        let (data, response) = try await client.request(
            path: "/api/v1/catalog/paths/\(encodePath(slug))",
            authorized: accessToken != nil,
            accessToken: accessToken
        )
        if response.statusCode == 404 { return nil }
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(LearningPathDetail.self, from: data)
    }

    static func fetchMyPaths(accessToken: String) async throws -> [PathProgress] {
        let (data, response) = try await client.request(
            path: "/api/v1/me/paths",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(MyPathsListResponse.self, from: data).paths ?? []
    }

    static func fetchPathProgress(pathId: String, accessToken: String) async throws -> PathProgress {
        let (data, response) = try await client.request(
            path: "/api/v1/me/paths/\(encodePath(pathId))/progress",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(PathProgress.self, from: data)
    }

    static func enrollInPath(pathId: String, accessToken: String) async throws -> PathEnrollResponse {
        let (data, response) = try await client.request(
            path: "/api/v1/paths/\(encodePath(pathId))/enroll",
            method: "POST",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(PathEnrollResponse.self, from: data)
    }

    static func postRecommendationEvent(body: RecommendationEventBody, accessToken: String) async throws {
        let (data, response) = try await client.request(
            path: "/api/v1/recommendations/event",
            method: "POST",
            body: body,
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
    }
}