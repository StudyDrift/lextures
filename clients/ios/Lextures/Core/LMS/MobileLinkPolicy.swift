import Foundation

/// Pure MB.1 link classification — mirrored on Android. No networking / UI.
enum MobileLinkPolicy {
    enum Handling: String, Equatable {
        case inApp = "in_app"
        case system
        case blocked

        static func parse(_ raw: String?) -> Handling {
            switch (raw ?? "").trimmingCharacters(in: .whitespacesAndNewlines).lowercased() {
            case "system": return .system
            case "blocked": return .blocked
            default: return .inApp
            }
        }
    }

    enum Classification: String, Equatable {
        case native
        case inAppBrowser = "in_app_browser"
        case systemBrowser = "system_browser"
        case externalApp = "external_app"
        case authSession = "auth_session"
        case blocked
    }

    struct State: Equatable {
        var handling: Handling = .inApp
        /// When false, external http(s) fall through to the system browser.
        /// Production always leaves this true; tests may force false.
        var inAppBrowserEnabled: Bool = true
        /// Host of the configured API base (may include port).
        var apiHost: String = ""
        var firstPartyHosts: [String] = ["lextures.com", "localhost", "127.0.0.1"]
    }

    private static let externalSchemes: Set<String> = [
        "mailto", "tel", "sms", "geo", "maps", "itms-apps", "market",
    ]

    private static let webSchemes: Set<String> = ["http", "https"]

    private static let nativeAppHostSuffixes: [String] = [
        "zoom.us", "zoom.com", "meet.google.com", "teams.microsoft.com", "teams.live.com",
    ]

    private static let checkoutHostSuffixes: [String] = [
        "checkout.stripe.com", "billing.stripe.com", "pay.stripe.com",
    ]

    private static let ssoPathMarkers: [String] = [
        "/oauth", "/oidc", "/saml", "/sso", "/auth/callback", "/login/oauth",
    ]

    // MARK: - Classify

    /// Pure classification of a URL string (absolute, relative `/…`, or `lextures://…`).
    static func classify(urlString: String, state: State) -> Classification {
        let trimmed = urlString.trimmingCharacters(in: .whitespacesAndNewlines)
        if trimmed.isEmpty { return .blocked }

        // Relative in-app path.
        if trimmed.hasPrefix("/") {
            return .native
        }

        guard let components = URLComponents(string: trimmed) else {
            return .blocked
        }
        let scheme = (components.scheme ?? "").lowercased()

        if scheme == "lextures" {
            return .native
        }

        if externalSchemes.contains(scheme) {
            return .externalApp
        }

        if !webSchemes.contains(scheme) {
            // Unknown / dangerous schemes (javascript:, file:, data:, …).
            return .blocked
        }

        let host = (components.host ?? "").lowercased()
        if host.isEmpty { return .blocked }

        if isFirstParty(host: host, state: state) {
            return .native
        }

        if isNativeAppHost(host) {
            return .externalApp
        }

        if isCheckoutHost(host) {
            return .systemBrowser
        }

        let path = components.path.lowercased()
        if isSSOPath(path) {
            return .authSession
        }

        // External http(s) — honour admin policy + feature flag.
        switch state.handling {
        case .blocked:
            return .blocked
        case .system:
            return .systemBrowser
        case .inApp:
            return state.inAppBrowserEnabled ? .inAppBrowser : .systemBrowser
        }
    }

    // MARK: - Bearer attachment (FR-20 / FR-21)

    /// True when an Authorization bearer header may be attached for this request host.
    /// Uses parsed-host equality / subdomain — never string-prefix on full URLs.
    static func shouldAttachBearer(requestHost: String?, apiHost: String) -> Bool {
        let req = stripPort(requestHost ?? "").lowercased()
        let api = stripPort(apiHost).lowercased()
        if req.isEmpty || api.isEmpty { return false }
        if req == api { return true }
        // Subdomain of api host (e.g. cdn.api.lextures.com under api.lextures.com).
        if req.hasSuffix("." + api) { return true }
        // Dev: 127.0.0.1 / localhost treated as first-party API when api is loopback.
        if isLoopback(req) && isLoopback(api) { return true }
        return false
    }

    static func shouldAttachBearer(requestURL: URL, apiBaseURL: URL) -> Bool {
        shouldAttachBearer(requestHost: requestURL.host, apiHost: apiBaseURL.host ?? "")
    }

    // MARK: - Display helpers

    /// Registrable-ish host for chrome (drops leading www.).
    static func displayHost(for url: URL) -> String {
        let host = (url.host ?? "").lowercased()
        if host.hasPrefix("www.") { return String(host.dropFirst(4)) }
        return host
    }

    static func isSecure(_ url: URL) -> Bool {
        (url.scheme ?? "").lowercased() == "https"
    }

    // MARK: - Telemetry shape (FR-25 / AC-20)

    /// Returns a content-free telemetry payload, dropping any forbidden keys.
    static func sanitizeTelemetry(_ raw: [String: String]) -> [String: String] {
        let allowed: Set<String> = ["source", "classification", "outcome", "errorClass"]
        var out: [String: String] = [:]
        for (k, v) in raw where allowed.contains(k) {
            out[k] = v
        }
        return out
    }

    // MARK: - Internals

    private static func isFirstParty(host: String, state: State) -> Bool {
        let h = host.lowercased()
        for candidate in state.firstPartyHosts {
            let c = candidate.lowercased()
            if h == c || h.hasSuffix("." + c) { return true }
        }
        // API host itself is first-party for deep links that point at the API origin.
        let api = stripPort(state.apiHost).lowercased()
        if !api.isEmpty && (h == api || h.hasSuffix("." + api)) { return true }
        return false
    }

    private static func isNativeAppHost(_ host: String) -> Bool {
        let h = host.lowercased()
        for suffix in nativeAppHostSuffixes {
            if h == suffix || h.hasSuffix("." + suffix) { return true }
        }
        return false
    }

    private static func isCheckoutHost(_ host: String) -> Bool {
        let h = host.lowercased()
        for suffix in checkoutHostSuffixes {
            if h == suffix || h.hasSuffix("." + suffix) { return true }
        }
        return false
    }

    private static func isSSOPath(_ path: String) -> Bool {
        for marker in ssoPathMarkers {
            if path.contains(marker) { return true }
        }
        return false
    }

    private static func stripPort(_ host: String) -> String {
        if let idx = host.firstIndex(of: ":") {
            return String(host[..<idx])
        }
        return host
    }

    private static func isLoopback(_ host: String) -> Bool {
        let h = stripPort(host).lowercased()
        return h == "localhost" || h == "127.0.0.1" || h == "::1"
    }
}
