import Foundation

/// Account integrations helpers (M14.1) — gating, secret handling, and feed URL resolution.
enum AccountIntegrationsLogic {
    static let rbacManagePermission = "global:app:rbac:manage"
    static let defaultCreateScopes = ["mcp:connect", "courses:read"]

    static func integrationsEnabled(_ features: MobilePlatformFeatures) -> Bool {
        features.ffMobileSettingsIntegrations
    }

    static func accessKeysEnabled(_ features: MobilePlatformFeatures) -> Bool {
        features.ffApiTokens
    }

    static func calendarSubscriptionsEnabled(_ features: MobilePlatformFeatures) -> Bool {
        features.ffCalendarFeeds
    }

    static func canManageServiceTokens(permissions: [String]) -> Bool {
        permissions.contains(rbacManagePermission)
    }

    /// Whether the service-tokens section may be shown (permission gate; org admins resolved via API probe).
    static func shouldShowServiceTokensSection(
        permissions: [String],
        adminApiForbidden: Bool
    ) -> Bool {
        !adminApiForbidden && canManageServiceTokens(permissions: permissions)
    }

    static func isRevoked(_ revokedAt: String?) -> Bool {
        guard let revokedAt else { return false }
        return !revokedAt.isEmpty
    }

    static func activeAccessKeys(_ tokens: [AccessKeySummary]) -> [AccessKeySummary] {
        tokens.filter { !isRevoked($0.revokedAt) && $0.isServiceToken != true }
    }

    static func revokedAccessKeys(_ tokens: [AccessKeySummary]) -> [AccessKeySummary] {
        tokens.filter { isRevoked($0.revokedAt) && $0.isServiceToken != true }
    }

    static func activeServiceTokens(_ tokens: [AccessKeySummary]) -> [AccessKeySummary] {
        tokens.filter { $0.isServiceToken == true && !isRevoked($0.revokedAt) }
    }

    static func resolveCalendarFeedURL(template: String, token: String) -> String {
        let encoded = token.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? token
        return template.replacingOccurrences(of: "<token>", with: encoded)
    }

    static func resolvedPersonalFeedURL(
        info: CalendarTokenInfo?,
        created: CalendarTokenCreated?
    ) -> String? {
        if let feedUrl = created?.feedUrl, !feedUrl.isEmpty { return feedUrl }
        if let token = created?.token, let template = info?.personalFeedUrl, !template.isEmpty {
            return resolveCalendarFeedURL(template: template, token: token)
        }
        return nil
    }

    static func resolvedCourseFeedURL(template: String, token: String?) -> String? {
        guard let token, !token.isEmpty else { return nil }
        return resolveCalendarFeedURL(template: template, token: token)
    }

    /// Injects an access key into MCP JSON config for copy-to-clipboard.
    static func mcpConfigJSON(base: [String: JSONValue], token: String) -> String? {
        var draft = base
        guard case .object(var servers)? = draft["mcpServers"],
              case .object(var lextures)? = servers["lextures"],
              case .object(var env)? = lextures["env"] else {
            return prettyJSONString(base)
        }
        let trimmed = token.trimmingCharacters(in: .whitespacesAndNewlines)
        if !trimmed.isEmpty {
            env["LEXTURES_API_TOKEN"] = .string(trimmed)
        }
        lextures["env"] = .object(env)
        servers["lextures"] = .object(lextures)
        draft["mcpServers"] = .object(servers)
        return prettyJSONString(draft)
    }

    private static func prettyJSONString(_ value: [String: JSONValue]) -> String? {
        let encoder = JSONEncoder()
        encoder.outputFormatting = [.prettyPrinted, .sortedKeys]
        guard let data = try? encoder.encode(value) else { return nil }
        return String(data: data, encoding: .utf8)
    }
}