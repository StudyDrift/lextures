import SwiftUI
import WebKit

/// In-app browser for external links and textbook resources with auth injection (M3.1).
struct WebItemView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.openURL) private var openURL

    let title: String
    let urlString: String
    var provider: String?

    @State private var loadError: String?

    var body: some View {
        VStack(spacing: 0) {
            if let loadError {
                LMSErrorBanner(message: loadError)
                    .padding(16)
            }
            AuthenticatedWebView(
                urlString: urlString,
                accessToken: session.accessToken,
                onError: { loadError = L.text("mobile.modules.webLoadError") }
            )
            HStack {
                Spacer()
                Button(L.text("mobile.modules.openExternal")) {
                    if let url = resolvedURL { openURL(url) }
                }
                .font(.subheadline.weight(.semibold))
                .padding(12)
            }
            .background(LexturesTheme.sceneBackground(for: colorScheme))
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

struct AuthenticatedWebView: UIViewRepresentable {
    let urlString: String
    let accessToken: String?
    var onError: () -> Void

    func makeCoordinator() -> Coordinator { Coordinator(onError: onError) }

    func makeUIView(context: Context) -> WKWebView {
        let config = WKWebViewConfiguration()
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
        if let accessToken, url.absoluteString.hasPrefix(AppConfiguration.apiBaseURL.absoluteString) {
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
    }
}
