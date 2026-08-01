import SwiftUI
import WebKit

/// Course web / textbook item — shared MB.1 browser chrome with first-party auth injection (FR-20/28).
struct WebItemView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell

    let title: String
    let urlString: String
    var provider: String?

    var body: some View {
        Group {
            if let url = resolvedURL {
                // When the in-app browser feature is on and policy allows, use full shared chrome.
                // Otherwise fall back to system open for external, or still show shared browser for first-party.
                let state = LinkOpener.policyState(from: shell)
                let classification = MobileLinkPolicy.classify(urlString: urlString, state: state)
                if classification == .inAppBrowser || classification == .native || classification == .systemBrowser {
                    InAppBrowserView(
                        session: InAppBrowserSession(
                            initialURL: url,
                            accessToken: session.accessToken,
                            source: "web_item",
                            allowReport: false
                        ),
                        onDismiss: {
                            // Pushed as a nav destination — close is a no-op; user uses system back.
                        }
                    )
                    .navigationBarHidden(true)
                } else {
                    ContentUnavailableView(
                        L.text("mobile.browser.blockedByPolicy"),
                        systemImage: "hand.raised",
                        description: Text(L.text("mobile.browser.blockedByPolicy"))
                    )
                }
            } else {
                ContentUnavailableView(
                    L.text("mobile.browser.errorTitle"),
                    systemImage: "exclamationmark.triangle",
                    description: Text(L.text("mobile.browser.errorBody"))
                )
            }
        }
        .navigationTitle(title)
        .navigationBarTitleDisplayMode(.inline)
    }

    private var resolvedURL: URL? {
        if urlString.hasPrefix("/") {
            return AppConfiguration.apiURL(path: urlString)
        }
        return URL(string: urlString)
    }
}

// MARK: - AuthenticatedWebView (collab docs & embedded first-party pages)

/// Lightweight WKWebView with first-party bearer injection (parsed-host check, FR-20).
/// Used by surfaces that need an embedded web editor (e.g. collab docs) rather than the full-screen MB.1 chrome.
struct AuthenticatedWebView: UIViewRepresentable {
    let urlString: String
    let accessToken: String?
    var onError: () -> Void

    func makeCoordinator() -> Coordinator { Coordinator(onError: onError) }

    func makeUIView(context: Context) -> WKWebView {
        let config = WKWebViewConfiguration()
        config.websiteDataStore = InAppBrowserDataStore.shared.store
        let webView = WKWebView(frame: .zero, configuration: config)
        webView.navigationDelegate = context.coordinator
        load(into: webView)
        return webView
    }

    func updateUIView(_ webView: WKWebView, context: Context) {
        if context.coordinator.lastURL != urlString {
            context.coordinator.lastURL = urlString
            load(into: webView)
        }
    }

    private func load(into webView: WKWebView) {
        let url: URL?
        if urlString.hasPrefix("/") {
            url = AppConfiguration.apiURL(path: urlString)
        } else {
            url = URL(string: urlString)
        }
        guard let url else {
            onError()
            return
        }
        var request = URLRequest(url: url)
        if let accessToken,
           MobileLinkPolicy.shouldAttachBearer(requestURL: url, apiBaseURL: AppConfiguration.apiBaseURL) {
            request.setValue("Bearer \(accessToken)", forHTTPHeaderField: "Authorization")
        }
        webView.load(request)
    }

    final class Coordinator: NSObject, WKNavigationDelegate {
        var lastURL: String?
        let onError: () -> Void

        init(onError: @escaping () -> Void) {
            self.onError = onError
        }

        func webView(_ webView: WKWebView, didFail navigation: WKNavigation!, withError error: Error) {
            onError()
        }

        func webView(_ webView: WKWebView, didFailProvisionalNavigation navigation: WKNavigation!, withError error: Error) {
            onError()
        }

        /// Do not re-attach Authorization across cross-origin redirects (FR-21).
        func webView(
            _ webView: WKWebView,
            decidePolicyFor navigationAction: WKNavigationAction,
            decisionHandler: @escaping (WKNavigationActionPolicy) -> Void
        ) {
            guard let url = navigationAction.request.url else {
                decisionHandler(.allow)
                return
            }
            if navigationAction.request.value(forHTTPHeaderField: "Authorization") != nil,
               !MobileLinkPolicy.shouldAttachBearer(requestURL: url, apiBaseURL: AppConfiguration.apiBaseURL) {
                decisionHandler(.cancel)
                webView.load(URLRequest(url: url))
                return
            }
            decisionHandler(.allow)
        }
    }
}
