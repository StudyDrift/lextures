import SwiftUI
import WebKit

/// WKWebView host for the in-app browser. No user scripts, no message handlers (FR-24).
struct BrowserWebView: UIViewRepresentable {
    let url: URL
    let accessToken: String?
    @Binding var currentURL: URL
    @Binding var pageTitle: String
    @Binding var progress: Double
    @Binding var canGoBack: Bool
    @Binding var loadState: InAppBrowserView.LoadState

    func makeCoordinator() -> Coordinator {
        Coordinator(self)
    }

    func makeUIView(context: Context) -> WKWebView {
        let config = WKWebViewConfiguration()
        // App-scoped data store, separate from ephemeral API session cookies.
        config.websiteDataStore = InAppBrowserDataStore.shared.store
        config.allowsInlineMediaPlayback = true
        config.mediaTypesRequiringUserActionForPlayback = []
        // Explicitly no userContentController scripts / message handlers (FR-24).
        config.userContentController = WKUserContentController()

        let webView = WKWebView(frame: .zero, configuration: config)
        webView.navigationDelegate = context.coordinator
        webView.uiDelegate = context.coordinator
        webView.allowsBackForwardNavigationGestures = true
        webView.allowsLinkPreview = true

        context.coordinator.observe(webView)
        context.coordinator.load(url, into: webView, accessToken: accessToken)
        return webView
    }

    func updateUIView(_ webView: WKWebView, context: Context) {
        // Session is fixed per presentation; no URL prop thrashing.
    }

    final class Coordinator: NSObject, WKNavigationDelegate, WKUIDelegate {
        var parent: BrowserWebView
        private var progressObs: NSKeyValueObservation?
        private var titleObs: NSKeyValueObservation?
        private var urlObs: NSKeyValueObservation?
        private var canGoBackObs: NSKeyValueObservation?
        private weak var webView: WKWebView?

        init(_ parent: BrowserWebView) {
            self.parent = parent
        }

        func observe(_ webView: WKWebView) {
            self.webView = webView
            progressObs = webView.observe(\.estimatedProgress, options: .new) { [weak self] wv, _ in
                DispatchQueue.main.async { self?.parent.progress = wv.estimatedProgress }
            }
            titleObs = webView.observe(\.title, options: .new) { [weak self] wv, _ in
                DispatchQueue.main.async { self?.parent.pageTitle = wv.title ?? "" }
            }
            urlObs = webView.observe(\.url, options: .new) { [weak self] wv, _ in
                DispatchQueue.main.async {
                    if let u = wv.url { self?.parent.currentURL = u }
                }
            }
            canGoBackObs = webView.observe(\.canGoBack, options: .new) { [weak self] wv, _ in
                DispatchQueue.main.async { self?.parent.canGoBack = wv.canGoBack }
            }

            NotificationCenter.default.addObserver(
                self, selector: #selector(goBack), name: .inAppBrowserGoBack, object: nil
            )
            NotificationCenter.default.addObserver(
                self, selector: #selector(reload), name: .inAppBrowserReload, object: nil
            )
        }

        deinit {
            NotificationCenter.default.removeObserver(self)
            progressObs = nil
            titleObs = nil
            urlObs = nil
            canGoBackObs = nil
        }

        @objc func goBack() {
            webView?.goBack()
        }

        @objc func reload() {
            guard let webView else { return }
            parent.loadState = .loading
            if webView.url != nil {
                webView.reload()
            } else {
                load(parent.url, into: webView, accessToken: parent.accessToken)
            }
        }

        func load(_ url: URL, into webView: WKWebView, accessToken: String?) {
            var request = URLRequest(url: url)
            if let accessToken,
               MobileLinkPolicy.shouldAttachBearer(requestURL: url, apiBaseURL: AppConfiguration.apiBaseURL) {
                request.setValue("Bearer \(accessToken)", forHTTPHeaderField: "Authorization")
            }
            parent.loadState = .loading
            webView.load(request)
        }

        func webView(_ webView: WKWebView, didFinish navigation: WKNavigation!) {
            parent.loadState = .loaded
            parent.progress = 1
            DispatchQueue.main.asyncAfter(deadline: .now() + 0.25) {
                self.parent.progress = 0
            }
        }

        func webView(_ webView: WKWebView, didFail navigation: WKNavigation!, withError error: Error) {
            handleFailure(error)
        }

        func webView(_ webView: WKWebView, didFailProvisionalNavigation navigation: WKNavigation!, withError error: Error) {
            handleFailure(error)
        }

        private func handleFailure(_ error: Error) {
            let ns = error as NSError
            if ns.domain == NSURLErrorDomain && ns.code == NSURLErrorNotConnectedToInternet {
                parent.loadState = .offline
            } else {
                parent.loadState = .error(L.text("mobile.browser.errorBody"))
            }
            parent.progress = 0
        }

        // Re-evaluate bearer on every navigation (FR-21). Do not attach across cross-origin redirects.
        func webView(
            _ webView: WKWebView,
            decidePolicyFor navigationAction: WKNavigationAction,
            decisionHandler: @escaping (WKNavigationActionPolicy) -> Void
        ) {
            guard let url = navigationAction.request.url else {
                decisionHandler(.allow)
                return
            }
            let scheme = (url.scheme ?? "").lowercased()
            if scheme == "http" || scheme == "https" {
                // If the request already has Authorization and the host is not first-party, cancel & reload without it.
                if navigationAction.request.value(forHTTPHeaderField: "Authorization") != nil,
                   !MobileLinkPolicy.shouldAttachBearer(requestURL: url, apiBaseURL: AppConfiguration.apiBaseURL) {
                    decisionHandler(.cancel)
                    var clean = URLRequest(url: url)
                    webView.load(clean)
                    return
                }
                decisionHandler(.allow)
                return
            }
            // Non-web schemes: hand to LinkOpener-equivalent OS open, stay in browser.
            if ["mailto", "tel", "sms", "itms-apps", "market"].contains(scheme) {
                UIApplication.shared.open(url)
                decisionHandler(.cancel)
                return
            }
            decisionHandler(.cancel)
        }

        // window.open / target=_blank → same browser (FR-16 / AC-14).
        func webView(
            _ webView: WKWebView,
            createWebViewWith configuration: WKWebViewConfiguration,
            for navigationAction: WKNavigationAction,
            windowFeatures: WKWindowFeatures
        ) -> WKWebView? {
            if navigationAction.targetFrame == nil, let url = navigationAction.request.url {
                webView.load(URLRequest(url: url))
            }
            return nil
        }
    }
}

/// Session-scoped website data store; purged on sign-out and Settings clear (FR-22/23).
final class InAppBrowserDataStore {
    static let shared = InAppBrowserDataStore()

    /// App-scoped store separate from ephemeral API session cookies (FR-22).
    let store = WKWebsiteDataStore.default()

    private init() {}

    func purgeAll() async {
        let types = WKWebsiteDataStore.allWebsiteDataTypes()
        let records = await store.dataRecords(ofTypes: types)
        await store.removeData(ofTypes: types, for: records)
    }
}
