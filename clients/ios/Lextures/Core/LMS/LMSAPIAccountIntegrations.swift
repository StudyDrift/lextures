import Foundation

/// Account integrations API (M14.1) — access keys, calendar token, MCP, service tokens.
extension LMSAPI {
    static func fetchAccessKeyScopes(accessToken: String) async throws -> [AccessKeyScopeDef] {
        let (data, response) = try await client.request(
            path: "/api/v1/me/access-keys/scopes",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(AccessKeyScopesResponse.self, from: data).scopes
    }

    static func fetchAccessKeys(accessToken: String) async throws -> [AccessKeySummary] {
        let (data, response) = try await client.request(
            path: "/api/v1/me/access-keys",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(AccessKeysListResponse.self, from: data).tokens
    }

    static func createAccessKey(
        label: String,
        scopes: [String],
        accessToken: String
    ) async throws -> CreateAccessKeyResponse {
        let (data, response) = try await client.request(
            path: "/api/v1/me/access-keys",
            method: "POST",
            body: CreateAccessKeyRequest(label: label, scopes: scopes),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(CreateAccessKeyResponse.self, from: data)
    }

    static func revokeAccessKey(id: String, accessToken: String) async throws {
        let (data, response) = try await client.request(
            path: "/api/v1/me/access-keys/\(encodePath(id))",
            method: "DELETE",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
    }

    static func rotateAccessKey(id: String, accessToken: String) async throws -> RotateAccessKeyResponse {
        let (data, response) = try await client.request(
            path: "/api/v1/me/access-keys/\(encodePath(id))/rotate",
            method: "POST",
            body: RotateAccessKeyRequest(),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(RotateAccessKeyResponse.self, from: data)
    }

    static func fetchMCPConfig(accessToken: String) async throws -> MCPConfigResponse {
        let (data, response) = try await client.request(
            path: "/api/v1/me/integrations/mcp",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(MCPConfigResponse.self, from: data)
    }

    /// Returns nil when the caller lacks admin permission (HTTP 403).
    static func fetchServiceTokens(accessToken: String) async throws -> [AccessKeySummary]? {
        let (data, response) = try await client.request(
            path: "/api/v1/admin/tokens",
            authorized: true,
            accessToken: accessToken
        )
        if response.statusCode == 403 { return nil }
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        let all = try decode(AccessKeysListResponse.self, from: data).tokens
        return AccountIntegrationsLogic.activeServiceTokens(all)
    }

    static func createServiceToken(
        serviceAccountName: String,
        label: String,
        scopes: [String],
        accessToken: String
    ) async throws -> CreateServiceTokenResponse {
        let (data, response) = try await client.request(
            path: "/api/v1/admin/tokens",
            method: "POST",
            body: CreateServiceTokenRequest(
                serviceAccountName: serviceAccountName,
                label: label,
                scopes: scopes
            ),
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
        return try decode(CreateServiceTokenResponse.self, from: data)
    }

    static func revokeServiceToken(id: String, accessToken: String) async throws {
        let (data, response) = try await client.request(
            path: "/api/v1/admin/tokens/\(encodePath(id))",
            method: "DELETE",
            authorized: true,
            accessToken: accessToken
        )
        guard (200 ... 299).contains(response.statusCode) else {
            throw APIError.httpStatus(response.statusCode, message: parseAPIErrorMessage(from: data))
        }
    }
}