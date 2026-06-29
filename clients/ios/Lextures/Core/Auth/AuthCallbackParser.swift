import Foundation

/// Parsed auth payload from SSO (`lextures://auth/callback`) or magic-link deep links.
struct AuthCallbackPayload: Equatable {
    var accessToken: String?
    var refreshToken: String?
    var expiresIn: Int?
    var requiresMFA: Bool
    var mfaPendingToken: String?
    var mfaSetupRequired: Bool
    var magicLinkToken: String?

    var asTokenResponse: AuthTokenResponse {
        AuthTokenResponse(
            accessToken: accessToken,
            refreshToken: refreshToken,
            expiresIn: expiresIn,
            requiresMFA: requiresMFA ? true : nil,
            mfaPendingToken: mfaPendingToken,
            mfaSetupRequired: mfaSetupRequired ? true : nil,
            user: nil
        )
    }
}

enum AuthCallbackParser {
    /// Returns a payload when `raw` is an auth callback or magic-link URL; otherwise `nil`.
    static func parse(_ raw: String?) -> AuthCallbackPayload? {
        guard let raw, !raw.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else { return nil }
        let trimmed = raw.trimmingCharacters(in: .whitespacesAndNewlines)

        if let magic = parseMagicLink(trimmed) {
            return magic
        }
        if let callback = parseAuthCallback(trimmed) {
            return callback
        }
        return nil
    }

    static func parseMagicLink(_ value: String) -> AuthCallbackPayload? {
        guard let path = extractPath(from: value) else { return nil }
        let segments = path.split(separator: "/").map(String.init)
        guard segments.first?.lowercased() == "login",
              segments.count >= 2,
              segments[1].lowercased() == "magic-link" else {
            return nil
        }
        let token = queryValue(named: "token", in: value)?.trimmingCharacters(in: .whitespacesAndNewlines)
        guard let token, !token.isEmpty else { return nil }
        return AuthCallbackPayload(
            accessToken: nil,
            refreshToken: nil,
            expiresIn: nil,
            requiresMFA: false,
            mfaPendingToken: nil,
            mfaSetupRequired: false,
            magicLinkToken: token
        )
    }

    static func parseAuthCallback(_ value: String) -> AuthCallbackPayload? {
        guard let url = URL(string: value) else { return nil }
        let isCustomScheme = url.scheme?.lowercased() == AuthConstants.callbackScheme
            && url.host?.lowercased() == AuthConstants.callbackHost
            && url.path == AuthConstants.callbackPath
        if !isCustomScheme {
            return nil
        }
        return payload(from: url)
    }

    static func payload(from url: URL) -> AuthCallbackPayload {
        var params = queryItems(from: url)
        if let fragment = url.fragment, !fragment.isEmpty {
            params.merge(queryItems(fromFragment: fragment)) { _, new in new }
        }
        return AuthCallbackPayload(
            accessToken: params["access_token"],
            refreshToken: params["refresh_token"],
            expiresIn: params["expires_in"].flatMap { Int($0) },
            requiresMFA: params["requires_mfa"] == "1" || params["requires_mfa"] == "true",
            mfaPendingToken: params["mfa_pending_token"],
            mfaSetupRequired: params["mfa_setup_required"] == "1" || params["mfa_setup_required"] == "true",
            magicLinkToken: nil
        )
    }

    private static func extractPath(from value: String) -> String? {
        if value.hasPrefix("lextures://") {
            let stripped = String(value.dropFirst("lextures://".count))
            return stripped.hasPrefix("/") ? stripped : "/\(stripped)"
        }
        if value.hasPrefix("/") {
            return value.split(separator: "?", maxSplits: 1).first.map(String.init) ?? value
        }
        if let url = URL(string: value), let host = url.host?.lowercased() {
            if host == "lextures.com" || host.hasSuffix(".lextures.com") || host == "localhost" {
                var path = url.path
                if !path.hasPrefix("/") { path = "/\(path)" }
                return path
            }
        }
        return nil
    }

    private static func queryValue(named name: String, in value: String) -> String? {
        guard let url = URL(string: value) else { return nil }
        return queryItems(from: url)[name]
    }

    private static func queryItems(from url: URL) -> [String: String] {
        var out: [String: String] = [:]
        URLComponents(url: url, resolvingAgainstBaseURL: false)?.queryItems?.forEach { item in
            if let value = item.value {
                out[item.name] = value
            }
        }
        return out
    }

    private static func queryItems(fromFragment fragment: String) -> [String: String] {
        var components = URLComponents()
        components.query = fragment
        var out: [String: String] = [:]
        components.queryItems?.forEach { item in
            if let value = item.value {
                out[item.name] = value
            }
        }
        return out
    }
}
