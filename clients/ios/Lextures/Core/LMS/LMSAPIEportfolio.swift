import Foundation

/// ePortfolio list, detail, artifacts, upload, and sharing (M12.1).
extension LMSAPI {
    static func fetchMyPortfolios(accessToken: String) async throws -> [PortfolioSummary] {
        let (data, response) = try await client.request(
            path: "/api/v1/me/portfolios",
            authorized: true,
            accessToken: accessToken
        )
        if response.statusCode == 501 {
            throw APIError.httpStatus(501, message: L.text("mobile.portfolio.unavailable"))
        }
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(PortfoliosListResponse.self, from: data).portfolios ?? []
    }

    static func createPortfolio(title: String, introText: String, accessToken: String) async throws -> PortfolioSummary {
        let (data, response) = try await client.request(
            path: "/api/v1/me/portfolios",
            method: "POST",
            body: CreatePortfolioRequest(title: title, introText: introText),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(PortfolioSummary.self, from: data)
    }

    static func fetchMyPortfolio(portfolioId: String, accessToken: String) async throws -> PortfolioDetailResponse {
        let (data, response) = try await client.request(
            path: "/api/v1/me/portfolios/\(encodePath(portfolioId))",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(PortfolioDetailResponse.self, from: data)
    }

    static func patchPortfolio(
        portfolioId: String,
        payload: PatchPortfolioRequest,
        accessToken: String
    ) async throws -> PortfolioSummary {
        let (data, response) = try await client.request(
            path: "/api/v1/me/portfolios/\(encodePath(portfolioId))",
            method: "PATCH",
            body: payload,
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(PortfolioSummary.self, from: data)
    }

    static func createArtifact(
        portfolioId: String,
        payload: CreateArtifactRequest,
        accessToken: String
    ) async throws -> PortfolioArtifact {
        let (data, response) = try await client.request(
            path: "/api/v1/me/portfolios/\(encodePath(portfolioId))/artifacts",
            method: "POST",
            body: payload,
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(PortfolioArtifact.self, from: data)
    }

    static func patchArtifact(
        portfolioId: String,
        artifactId: String,
        payload: PatchArtifactRequest,
        accessToken: String
    ) async throws -> PortfolioArtifact {
        let (data, response) = try await client.request(
            path: "/api/v1/me/portfolios/\(encodePath(portfolioId))/artifacts/\(encodePath(artifactId))",
            method: "PATCH",
            body: payload,
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(PortfolioArtifact.self, from: data)
    }

    static func deleteArtifact(portfolioId: String, artifactId: String, accessToken: String) async throws {
        let (data, response) = try await client.request(
            path: "/api/v1/me/portfolios/\(encodePath(portfolioId))/artifacts/\(encodePath(artifactId))",
            method: "DELETE",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) || response.statusCode == 204 else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
    }

    static func uploadPortfolioArtifactFile(
        portfolioId: String,
        fileData: Data,
        fileName: String,
        mimeType: String,
        title: String,
        description: String,
        outcomeIds: [String],
        isPublic: Bool,
        accessToken: String,
        onProgress: ((Double) -> Void)? = nil
    ) async throws -> PortfolioArtifact {
        var fields: [String: String] = [
            "title": title,
            "description": description,
            "isPublic": isPublic ? "true" : "false",
        ]
        if !outcomeIds.isEmpty,
           let json = try? JSONEncoder().encode(outcomeIds),
           let raw = String(data: json, encoding: .utf8) {
            fields["outcomeIds"] = raw
        }
        let (data, response) = try await client.uploadMultipart(
            path: "/api/v1/me/portfolios/\(encodePath(portfolioId))/artifacts/upload",
            fieldName: "file",
            fileName: fileName,
            mimeType: mimeType,
            fileData: fileData,
            extraFields: fields,
            accessToken: accessToken,
            onProgress: onProgress
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(PortfolioArtifact.self, from: data)
    }

    static func portfolioArtifactContentPath(portfolioId: String, artifactId: String) -> String {
        "/api/v1/me/portfolios/\(encodePath(portfolioId))/artifacts/\(encodePath(artifactId))/content"
    }
}
