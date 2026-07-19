import Foundation

/// Authenticated course marketplace browse, claim, and catalog-listing (MKT6).
extension LMSAPI {
    static func fetchMarketplaceCourses(
        query: String = "",
        category: String = "",
        level: String = "",
        sort: String = "popular",
        priceMax: Int? = nil,
        freeOnly: Bool = false,
        cursor: String = "",
        accessToken: String
    ) async throws -> MarketplaceSearchResponse {
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
        if freeOnly { items.append(URLQueryItem(name: "free_only", value: "true")) }
        let cursorTrimmed = cursor.trimmingCharacters(in: .whitespacesAndNewlines)
        if !cursorTrimmed.isEmpty { items.append(URLQueryItem(name: "cursor", value: cursorTrimmed)) }
        components.queryItems = items.isEmpty ? nil : items
        let queryString = components.percentEncodedQuery.map { "?\($0)" } ?? ""
        let (data, response) = try await client.request(
            path: "/api/v1/marketplace/courses\(queryString)",
            authorized: true,
            accessToken: accessToken
        )
        if response.statusCode == 404 {
            throw APIError.httpStatus(404, message: L.text("mobile.marketplace.unavailable"))
        }
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(MarketplaceSearchResponse.self, from: data)
    }

    static func fetchMarketplaceCategories(accessToken: String) async throws -> [MarketplaceCategory] {
        let (data, response) = try await client.request(
            path: "/api/v1/marketplace/categories",
            authorized: true,
            accessToken: accessToken
        )
        if response.statusCode == 404 { return [] }
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(MarketplaceCategoriesResponse.self, from: data).categories ?? []
    }

    static func fetchMarketplaceCourseDetail(
        slug: String,
        accessToken: String
    ) async throws -> MarketplaceCourseDetail? {
        let (data, response) = try await client.request(
            path: "/api/v1/marketplace/courses/\(encodePath(slug))",
            authorized: true,
            accessToken: accessToken
        )
        if response.statusCode == 404 { return nil }
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(MarketplaceCourseDetail.self, from: data)
    }

    static func claimMarketplaceCourse(slug: String, accessToken: String) async throws -> MarketplaceClaimResult {
        let (data, response) = try await client.requestRaw(
            path: "/api/v1/marketplace/courses/\(encodePath(slug))/claim",
            method: "POST",
            bodyData: Data("{}".utf8),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        let result = try decode(MarketplaceClaimResult.self, from: data)
        MarketplaceObservability.record(
            "marketplace_claim",
            attributes: ["already_owned": result.alreadyOwned == true ? "1" : "0"]
        )
        return result
    }

    /// Paid marketplace checkout (MOB.7) — Stripe session URL or already-owned.
    static func checkoutMarketplaceCourse(
        slug: String,
        accessToken: String
    ) async throws -> MarketplaceCheckoutResult {
        let (data, response) = try await client.requestRaw(
            path: "/api/v1/marketplace/courses/\(encodePath(slug))/checkout",
            method: "POST",
            bodyData: Data("{}".utf8),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        let result = try decode(MarketplaceCheckoutResult.self, from: data)
        if result.alreadyOwned == true {
            MarketplaceObservability.record("marketplace_checkout_started", attributes: ["already_owned": "1"])
        } else {
            MarketplaceObservability.record("marketplace_checkout_started")
        }
        return result
    }

    static func fetchMyPurchases(accessToken: String) async throws -> [CoursePurchase] {
        let (data, response) = try await client.request(
            path: "/api/v1/me/purchases",
            authorized: true,
            accessToken: accessToken
        )
        if response.statusCode == 404 { return [] }
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        MarketplaceObservability.record("purchases_list_viewed")
        return try decode(CoursePurchasesResponse.self, from: data).purchases ?? []
    }

    static func fetchCourseCatalogListing(
        courseCode: String,
        accessToken: String
    ) async throws -> CourseCatalogListing {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/catalog-listing",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(CourseCatalogListingResponse.self, from: data).listing
    }

    static func putCourseCatalogListing(
        courseCode: String,
        body: CourseCatalogListingPutBody,
        accessToken: String
    ) async throws -> CourseCatalogListing {
        let (data, response) = try await client.request(
            path: "/api/v1/courses/\(encodePath(courseCode))/catalog-listing",
            method: "PUT",
            body: body,
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(CourseCatalogListingResponse.self, from: data).listing
    }
}
