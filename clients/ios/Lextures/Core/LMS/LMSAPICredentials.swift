import Foundation

/// Completion credentials: list, share, LinkedIn, Open Badge export (M9.3).
extension LMSAPI {
    static func fetchMyCredentials(accessToken: String) async throws -> [IssuedCredentialSummary] {
        let (data, response) = try await client.request(
            path: "/api/v1/me/credentials",
            authorized: true,
            accessToken: accessToken
        )
        if response.statusCode == 404 {
            throw APIError.httpStatus(404, message: L.text("mobile.credentials.unavailable"))
        }
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(CredentialsListResponse.self, from: data).credentials ?? []
    }

    static func fetchCredentialLinkedInParams(
        credentialId: String,
        accessToken: String
    ) async throws -> CredentialLinkedInParams {
        let (data, response) = try await client.request(
            path: "/api/v1/credentials/\(encodePath(credentialId))/linkedin-params",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(CredentialLinkedInParams.self, from: data)
    }

    static func fetchCredentialBadgeExportUrl(
        credentialId: String,
        accessToken: String
    ) async throws -> CredentialBadgeExportResponse {
        let (data, response) = try await client.request(
            path: "/api/v1/credentials/\(encodePath(credentialId))/badge-export",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(CredentialBadgeExportResponse.self, from: data)
    }

    static func recordCredentialShare(
        credentialId: String,
        channel: String,
        accessToken: String
    ) async throws {
        let body = CredentialShareRequest(channel: channel)
        let (data, response) = try await client.request(
            path: "/api/v1/credentials/\(encodePath(credentialId))/share",
            method: "POST",
            body: body,
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) || response.statusCode == 204 else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
    }

    static func credentialPdfURL(credentialId: String) -> URL {
        AppConfiguration.apiURL(path: "/api/v1/credentials/\(encodePath(credentialId))/download")
    }
}