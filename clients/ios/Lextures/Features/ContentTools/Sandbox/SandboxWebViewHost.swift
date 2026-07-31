import SwiftUI
import WebKit

/// CT.M4 sandboxed tool host: native document fetch → opaque-origin WKWebView → bridge.
struct SandboxWebViewHost: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.locale) private var locale
    @Environment(\.layoutDirection) private var layoutDirection
    @Environment(\.openURL) private var openURL

    let toolId: String
    let instanceId: String
    let toolVersion: String
    let title: String
    let config: JSONValue
    let state: JSONValue
    let revision: Int64
    let readOnly: Bool
    let capabilities: [String]
    var save: (JSONValue) -> Void
    var runAction: (String, JSONValue) async throws -> JSONValue?
    var announce: (String, Bool) -> Void

    @State private var height: CGFloat = 160
    @State private var ready = false
    @State private var timedOut = false
    @State private var crashed = false
    @State private var documentHTML: String?
    @State private var fetchFailed = false
    @State private var reloadToken = 0

    var body: some View {
        Group {
            if crashed {
                ToolErrorCardView(
                    toolName: title,
                    message: L.text("mobile.contentTools.sandbox.crashed"),
                    onRetry: { resetAndReload() }
                )
            } else if timedOut && !ready {
                ToolErrorCardView(
                    toolName: title,
                    message: L.text("mobile.contentTools.sandbox.timeout"),
                    onRetry: { resetAndReload() }
                )
            } else if fetchFailed {
                ToolPlaceholderView(
                    reason: .unavailable,
                    message: L.text("mobile.contentTools.sandbox.needsConnection")
                )
            } else {
                VStack(alignment: .leading, spacing: 4) {
                    if !ready {
                        ToolPlaceholderView(reason: .loading)
                    }
                    if let documentHTML {
                        SandboxWKView(
                            html: documentHTML,
                            instanceId: instanceId,
                            toolId: toolId,
                            config: config,
                            state: state,
                            revision: revision,
                            readOnly: readOnly,
                            locale: locale.identifier,
                            dir: layoutDirection == .rightToLeft ? "rtl" : "ltr",
                            height: $height,
                            ready: $ready,
                            crashed: $crashed,
                            capabilities: capabilities,
                            save: save,
                            runAction: runAction,
                            announce: announce,
                            onOpenURL: { openURL($0) }
                        )
                        .frame(height: ready ? height : 0)
                        .clipped()
                        .opacity(ready ? 1 : 0)
                        .accessibilityElement(children: .contain)
                        .accessibilityLabel("\(title), \(L.text("mobile.contentTools.sandbox.badge"))")
                    }
                    Text(L.text("mobile.contentTools.sandbox.badge"))
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                }
            }
        }
        .task(id: "\(toolId)|\(instanceId)|\(reloadToken)") {
            await fetchDocument()
        }
        .onAppear {
            _ = SandboxWebViewPool.shared.retain(instanceId)
        }
        .onDisappear {
            SandboxWebViewPool.shared.release(instanceId)
        }
    }

    private func resetAndReload() {
        crashed = false
        timedOut = false
        ready = false
        fetchFailed = false
        documentHTML = nil
        reloadToken += 1
    }

    private func fetchDocument() async {
        documentHTML = nil
        fetchFailed = false
        ready = false
        timedOut = false
        let path = ContentToolSandboxLogic.documentPath(toolId: toolId, version: toolVersion)
        let url = AppConfiguration.webURL(path: path)
        var request = URLRequest(url: url)
        if let token = session.accessToken, !token.isEmpty {
            request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }
        do {
            let (data, response) = try await URLSession.shared.data(for: request)
            guard let http = response as? HTTPURLResponse, (200 ... 299).contains(http.statusCode),
                  let html = String(data: data, encoding: .utf8), !html.isEmpty else {
                fetchFailed = true
                return
            }
            documentHTML = html
            try? await Task.sleep(nanoseconds: UInt64(ContentToolSandboxLogic.readyTimeoutMs) * 1_000_000)
            if !Task.isCancelled && !ready {
                timedOut = true
            }
        } catch {
            fetchFailed = true
        }
    }
}

// MARK: - WKWebView representable

private struct SandboxWKView: UIViewRepresentable {
    let html: String
    let instanceId: String
    let toolId: String
    let config: JSONValue
    let state: JSONValue
    let revision: Int64
    let readOnly: Bool
    let locale: String
    let dir: String
    @Binding var height: CGFloat
    @Binding var ready: Bool
    @Binding var crashed: Bool
    let capabilities: [String]
    var save: (JSONValue) -> Void
    var runAction: (String, JSONValue) async throws -> JSONValue?
    var announce: (String, Bool) -> Void
    var onOpenURL: (URL) -> Void

    func makeCoordinator() -> Coordinator {
        Coordinator(parent: self)
    }

    func makeUIView(context: Context) -> WKWebView {
        let wkConfig = WKWebViewConfiguration()
        // Non-persistent store — nothing the tool writes survives the mount (FR-2 / privacy).
        wkConfig.websiteDataStore = .nonPersistent()
        wkConfig.preferences.javaScriptCanOpenWindowsAutomatically = false
        wkConfig.defaultWebpagePreferences.allowsContentJavaScript = true

        let webView = WKWebView(frame: .zero, configuration: wkConfig)
        webView.navigationDelegate = context.coordinator
        webView.uiDelegate = context.coordinator
        webView.scrollView.isScrollEnabled = false
        webView.isOpaque = false
        webView.backgroundColor = .clear

        let bridge = SandboxBridge(webView: webView, handlers: context.coordinator.makeHandlers())
        bridge.attach(to: webView.configuration.userContentController)
        context.coordinator.bridge = bridge
        context.coordinator.webView = webView

        // Opaque origin: loadHTMLString with nil baseURL (FR-2).
        webView.loadHTMLString(html, baseURL: nil)
        return webView
    }

    func updateUIView(_ webView: WKWebView, context: Context) {
        context.coordinator.parent = self
    }

    static func dismantleUIView(_ uiView: WKWebView, coordinator: Coordinator) {
        coordinator.bridge?.dispose(controller: uiView.configuration.userContentController)
        coordinator.bridge = nil
        uiView.stopLoading()
        uiView.navigationDelegate = nil
        uiView.uiDelegate = nil
    }

    @MainActor
    final class Coordinator: NSObject, WKNavigationDelegate, WKUIDelegate {
        var parent: SandboxWKView
        var bridge: SandboxBridge?
        weak var webView: WKWebView?
        private var didInit = false

        init(parent: SandboxWKView) {
            self.parent = parent
        }

        func makeHandlers() -> SandboxBridge.Handlers {
            SandboxBridge.Handlers(
                onReady: { [weak self] contract in
                    guard let self else { return }
                    if ContentToolSandboxLogic.contractInSupportedRange(contract) {
                        self.parent.ready = true
                    } else {
                        self.parent.crashed = true
                    }
                },
                onSave: { [weak self] state, _ in
                    guard let self, !self.parent.readOnly else { return }
                    self.parent.save(state)
                    self.bridge?.postStateAccepted(revision: self.parent.revision + 1)
                },
                onRunAction: { [weak self] id, action, input in
                    guard let self else { return }
                    Task { @MainActor in
                        do {
                            let result = try await self.parent.runAction(action, input)
                            self.bridge?.postActionResult(id: id, result: result)
                        } catch {
                            self.bridge?.postError(
                                id: id,
                                code: "action_failed",
                                message: error.localizedDescription
                            )
                        }
                    }
                },
                onResize: { [weak self] height in
                    self?.parent.height = CGFloat(height)
                },
                onAnnounce: { [weak self] message, assertive in
                    self?.parent.announce(message, assertive)
                },
                onInvalid: { _ in },
                onMetric: { _, _ in }
            )
        }

        func webView(_ webView: WKWebView, didFinish navigation: WKNavigation!) {
            guard !didInit else { return }
            didInit = true
            bridge?.postInit(
                instanceId: parent.instanceId,
                config: parent.config,
                state: parent.state,
                revision: parent.revision,
                locale: parent.locale,
                dir: parent.dir,
                readOnly: parent.readOnly,
                participantId: ContentToolSandboxLogic.opaqueParticipantId(parent.instanceId)
            )
        }

        func webViewWebContentProcessDidTerminate(_ webView: WKWebView) {
            parent.crashed = true
        }

        func webView(
            _ webView: WKWebView,
            decidePolicyFor navigationAction: WKNavigationAction,
            decisionHandler: @escaping (WKNavigationActionPolicy) -> Void
        ) {
            // Allow the initial about:blank / html load; block everything else (FR-13).
            if navigationAction.navigationType == .other {
                let url = navigationAction.request.url
                if url == nil || url?.absoluteString == "about:blank" || url?.scheme == nil {
                    decisionHandler(.allow)
                    return
                }
            }
            if let url = navigationAction.request.url {
                parent.onOpenURL(url)
            }
            decisionHandler(.cancel)
        }

        func webView(
            _ webView: WKWebView,
            createWebViewWith configuration: WKWebViewConfiguration,
            for navigationAction: WKNavigationAction,
            windowFeatures: WKWindowFeatures
        ) -> WKWebView? {
            if let url = navigationAction.request.url {
                parent.onOpenURL(url)
            }
            return nil
        }

        func webView(
            _ webView: WKWebView,
            requestMediaCapturePermissionFor origin: WKSecurityOrigin,
            initiatedByFrame frame: WKFrameInfo,
            type: WKMediaCaptureType,
            decisionHandler: @escaping (WKPermissionDecision) -> Void
        ) {
            let needed: String
            switch type {
            case .camera: needed = "camera"
            case .microphone: needed = "microphone"
            case .cameraAndMicrophone: needed = "camera"
            @unknown default:
                decisionHandler(.deny)
                return
            }
            decisionHandler(parent.capabilities.contains(needed) ? .prompt : .deny)
        }
    }
}
