import Foundation

extension LMSAPI {
    /// GET `/api/v1/admin-console/audit-log` — org-scoped admin audit events (MOB.3).
    static func fetchAdminAuditLog(
        orgId: String? = nil,
        action: String? = nil,
        accessToken: String
    ) async throws -> [AdminAuditEvent] {
        var query: [URLQueryItem] = []
        if let orgId, !orgId.isEmpty {
            query.append(URLQueryItem(name: "orgId", value: orgId))
        }
        if let action = AuditLogAdminLogic.normalizedActionFilter(action ?? "") {
            query.append(URLQueryItem(name: "action", value: action))
        }
        var path = "/api/v1/admin-console/audit-log"
        if !query.isEmpty {
            var components = URLComponents()
            components.queryItems = query
            if let qs = components.percentEncodedQuery, !qs.isEmpty {
                path += "?\(qs)"
            }
        }
        let (data, _) = try await client.request(
            path: path,
            authorized: true,
            accessToken: accessToken
        )
        return try decode(AdminAuditLogResponse.self, from: data).events
    }
}
