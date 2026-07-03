import Foundation

/// Public course catalog browse, landing, reviews, and self-enroll (M9.1).
extension LMSAPI {
    static func fetchPublicCatalogCourses(
        query: String = "",
        category: String = "",
        level: String = "",
        sort: String = "popular",
        priceMax: Int? = nil,
        cursor: String = "",
        accessToken: String? = nil
    ) async throws -> PublicCatalogSearchResponse {
        var components = URLComponents()
        var items: [URLQueryItem] = []
        let trimmed = query.trimmingCharacters(in: .whitespacesAndNewlines)
        if !trimmed.isEmpty { items.append(URLQueryItem(name: "q", value: trimmed)) }
        let categoryTrimmed = category.trimmingCharacters(in: .whitespacesAndNewlines)
        if !categoryTrimmed.isEmpty { items.append(URLQueryItem(name: "category", value: categoryTrimmed)) }
        let levelTrimmed = level.trimmingCharacters(in: .whitespacesAndNewlines)
        if !levelTrimmed.isEmpty { items.append(URLQueryItem(name: "level", value: levelTrimmed)) }
        let sortTrimmed = sort.trimmingCharacters(in: .whitespacesAndNewlines)
        if !sortTrimmed.isEmpty { items.append(URLQueryItem(name: "sort", value: sortTrimmed)) }
        if let priceMax { items.append(URLQueryItem(name: "price_max", value: String(priceMax))) }
        let cursorTrimmed = cursor.trimmingCharacters(in: .whitespacesAndNewlines)
        if !cursorTrimmed.isEmpty { items.append(URLQueryItem(name: "cursor", value: cursorTrimmed)) }
        components.queryItems = items.isEmpty ? nil : items
        let queryString = components.percentEncodedQuery.map { "?\($0)" } ?? ""
        let (data, response) = try await client.request(
            path: "/api/v1/public/catalog/courses\(queryString)",
            authorized: accessToken != nil,
            accessToken: accessToken
        )
        if response.statusCode == 404 {
            throw APIError.httpStatus(404, message: L.text("mobile.catalog.unavailable"))
        }
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(PublicCatalogSearchResponse.self, from: data)
    }

    static func fetchPublicCatalogCategories(accessToken: String? = nil) async throws -> [CatalogCategory] {
        let (data, response) = try await client.request(
            path: "/api/v1/public/catalog/categories",
            authorized: accessToken != nil,
            accessToken: accessToken
        )
        if response.statusCode == 404 { return [] }
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(CatalogCategoriesResponse.self, from: data).categories ?? []
    }

    static func fetchPublicCatalogCourseDetail(
        slug: String,
        accessToken: String? = nil
    ) async throws -> PublicCatalogCourse? {
        let (data, response) = try await client.request(
            path: "/api/v1/public/catalog/courses/\(encodePath(slug))",
            authorized: accessToken != nil,
            accessToken: accessToken
        )
        if response.statusCode == 404 { return nil }
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(PublicCatalogCourseDetailResponse.self, from: data).course
    }

    static func fetchPublicCatalogCourseReviews(
        slug: String,
        cursor: String = "",
        accessToken: String? = nil
    ) async throws -> CourseReviewsListResponse? {
        var components = URLComponents()
        let cursorTrimmed = cursor.trimmingCharacters(in: .whitespacesAndNewlines)
        if !cursorTrimmed.isEmpty {
            components.queryItems = [URLQueryItem(name: "cursor", value: cursorTrimmed)]
        }
        let queryString = components.percentEncodedQuery.map { "?\($0)" } ?? ""
        let (data, response) = try await client.request(
            path: "/api/v1/public/catalog/courses/\(encodePath(slug))/reviews\(queryString)",
            authorized: accessToken != nil,
            accessToken: accessToken
        )
        if response.statusCode == 404 { return nil }
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(CourseReviewsListResponse.self, from: data)
    }

    static func selfEnrollInCourse(courseCode: String, accessToken: String) async throws -> CourseSelfEnrollResponse {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/self-enroll",
            method: "POST",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(CourseSelfEnrollResponse.self, from: data)
    }
}