import Foundation
import UIKit
import SwiftUI

/// Single entry point for opening URLs from app UI (MB.1 FR-1).
/// Feature code MUST call this instead of `openURL` / `UIApplication.shared.open`.
@MainActor
enum LinkOpener {
    struct Request: Equatable {
        var urlString: String
        var source: String
        /// When set, overrides shell policy (tests / forced system).
        var forceSystemBrowser: Bool = false
        /// Optional access token for first-party WebItem loads.
        var accessToken: String? = nil
        /// When true, Report appears in overflow (user-generated surfaces).
        var allowReport: Bool = false
    }

    /// Classify and act. Native destinations are handed to `shell.openDeepLink`.
    /// In-app browser is presented via `shell.presentInAppBrowser`.
    @discardableResult
    static func open(_ request: Request, shell: AppShellModel) -> MobileLinkPolicy.Classification {
        let state = policyState(from: shell)
        var classification = MobileLinkPolicy.classify(urlString: request.urlString, state: state)
        if request.forceSystemBrowser,
           classification == .inAppBrowser || classification == .systemBrowser {
            classification = .systemBrowser
        }

        switch classification {
        case .native:
            let dest = DeepLinkRouter.resolve(request.urlString)
            shell.openDeepLink(dest)
            emit(source: request.source, classification: "internal", outcome: "native")

        case .inAppBrowser:
            guard let url = resolveURL(request.urlString) else {
                emit(source: request.source, classification: "external", outcome: "error", errorClass: "bad_url")
                return .blocked
            }
            shell.presentInAppBrowser(
                InAppBrowserSession(
                    initialURL: url,
                    accessToken: request.accessToken,
                    source: request.source,
                    allowReport: request.allowReport
                )
            )
            emit(source: request.source, classification: "external", outcome: "in_app")

        case .systemBrowser:
            guard let url = resolveURL(request.urlString) else {
                emit(source: request.source, classification: "external", outcome: "error", errorClass: "bad_url")
                return .blocked
            }
            UIApplication.shared.open(url)
            emit(source: request.source, classification: "external", outcome: "system")

        case .externalApp:
            guard let url = resolveURL(request.urlString) else {
                emit(source: request.source, classification: "external", outcome: "error", errorClass: "bad_url")
                return .blocked
            }
            UIApplication.shared.open(url)
            emit(source: request.source, classification: "external", outcome: "external_app")

        case .authSession:
            // Auth flows own ASWebAuthenticationSession; if we land here from content,
            // fall back to system browser rather than capturing in WKWebView.
            if let url = resolveURL(request.urlString) {
                UIApplication.shared.open(url)
            }
            emit(source: request.source, classification: "external", outcome: "auth_session")

        case .blocked:
            shell.presentLinkBlockedNotice()
            emit(source: request.source, classification: "external", outcome: "blocked")
        }

        return classification
    }

    /// Convenience for simple string URLs.
    @discardableResult
    static func open(_ urlString: String, shell: AppShellModel, source: String) -> MobileLinkPolicy.Classification {
        open(Request(urlString: urlString, source: source), shell: shell)
    }

    /// Convenience for `URL`.
    @discardableResult
    static func open(_ url: URL, shell: AppShellModel, source: String) -> MobileLinkPolicy.Classification {
        open(Request(urlString: url.absoluteString, source: source), shell: shell)
    }

    /// OpenURLAction bridge for SwiftUI `.environment(\.openURL, …)`.
    static func openURLAction(shell: AppShellModel, source: String = "content") -> OpenURLAction {
        OpenURLAction { url in
            _ = open(url, shell: shell, source: source)
            return .handled
        }
    }

    static func policyState(from shell: AppShellModel) -> MobileLinkPolicy.State {
        MobileLinkPolicy.State(
            handling: MobileLinkPolicy.Handling.parse(shell.platformFeatures.mobileLinkHandling),
            // MB.1: in-app browser is always on; org/platform policy uses mobileLinkHandling only.
            inAppBrowserEnabled: true,
            apiHost: AppConfiguration.apiBaseURL.host ?? ""
        )
    }

    static func resolveURL(_ urlString: String) -> URL? {
        let t = urlString.trimmingCharacters(in: .whitespacesAndNewlines)
        if t.hasPrefix("/") {
            return AppConfiguration.webURL(path: t)
        }
        return URL(string: t)
    }

    private static func emit(source: String, classification: String, outcome: String, errorClass: String? = nil) {
        var raw: [String: String] = [
            "source": source,
            "classification": classification,
            "outcome": outcome,
        ]
        if let errorClass { raw["errorClass"] = errorClass }
        let payload = MobileLinkPolicy.sanitizeTelemetry(raw)
        // Content-free counter channel; no third-party URL ever leaves the device (FR-25).
        #if DEBUG
        print("[LinkOpener]", payload)
        #endif
        InAppBrowserTelemetry.record(payload)
    }
}

/// Lightweight in-memory counters (content-free).
enum InAppBrowserTelemetry {
    private static var counts: [String: Int] = [:]

    static func record(_ payload: [String: String]) {
        let key = [
            payload["source"] ?? "unknown",
            payload["classification"] ?? "unknown",
            payload["outcome"] ?? "unknown",
        ].joined(separator: "|")
        counts[key, default: 0] += 1
    }

    static func resetForTests() { counts = [:] }
    static func snapshotForTests() -> [String: Int] { counts }
}

/// Presented in-app browser session (shell-owned).
struct InAppBrowserSession: Identifiable, Equatable {
    let id = UUID()
    var initialURL: URL
    var accessToken: String?
    var source: String
    var allowReport: Bool

    static func == (lhs: InAppBrowserSession, rhs: InAppBrowserSession) -> Bool {
        lhs.id == rhs.id
    }
}
