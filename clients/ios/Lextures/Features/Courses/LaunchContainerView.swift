import SwiftUI
import WebKit

/// Secured web container for H5P, SCORM, LTI, and vibe activities (M3.3).
struct LaunchContainerView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.openURL) private var openURL

    let course: CourseSummary
    let item: CourseStructureItem
    var onProgressChanged: (() async -> Void)?

    @State private var target: InteractiveLaunchTarget?
    @State private var loading = true
    @State private var errorMessage: String?
    @State private var loadGeneration = 0

    var body: some View {
        Group {
            if !NetworkMonitor.shared.isOnline {
                offlineState
            } else if let target {
                interactiveContent(target)
            } else if loading {
                ProgressView()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else {
                errorState
            }
        }
        .navigationTitle(target?.title ?? item.title)
        .navigationBarTitleDisplayMode(.inline)
        .task(id: loadGeneration) { await load() }
        .onDisappear {
            Task { await onProgressChanged?() }
        }
    }

    @ViewBuilder
    private func interactiveContent(_ target: InteractiveLaunchTarget) -> some View {
        VStack(spacing: 0) {
            if target.hasResume {
                Text(L.text("mobile.modules.interactive.resume"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .padding(.horizontal, 16)
                    .padding(.top, 8)
            }
            AuthenticatedLaunchWebView(
                target: target,
                accessToken: session.accessToken,
                onWebError: { errorMessage = L.text("mobile.modules.webLoadError") },
                onH5PXAPI: { statement in
                    await forwardH5PXAPI(packageId: target.packageId, statement: statement)
                },
                onActivityEvent: {
                    await onProgressChanged?()
                }
            )
            launchActions(externalURL: externalURL(for: target))
        }
    }

    private var offlineState: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()
            LMSEmptyState(
                systemImage: "wifi.slash",
                title: item.title,
                message: L.text("mobile.modules.interactive.offline")
            )
            .padding(24)
        }
    }

    private var errorState: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()
            VStack(spacing: 16) {
                LMSEmptyState(
                    systemImage: ItemKind.icon(for: item.kind),
                    title: item.title,
                    message: errorMessage ?? L.text("mobile.modules.loadError")
                )
                HStack(spacing: 12) {
                    Button(L.text("mobile.modules.interactive.retry")) {
                        errorMessage = nil
                        loading = true
                        loadGeneration += 1
                    }
                    .buttonStyle(AuthPrimaryButtonStyle())
                    if let url = target.flatMap({ externalURL(for: $0) }) {
                        Button(L.text("mobile.modules.openExternal")) {
                            openURL(url)
                        }
                        .font(.subheadline.weight(.semibold))
                    }
                }
            }
            .padding(24)
        }
    }

    @ViewBuilder
    private func launchActions(externalURL: URL?) -> some View {
        HStack {
            Spacer()
            if let externalURL {
                Button(L.text("mobile.modules.openExternal")) {
                    openURL(externalURL)
                }
                .font(.subheadline.weight(.semibold))
                .padding(12)
            }
        }
        .background(LexturesTheme.sceneBackground(for: colorScheme))
    }

    private func externalURL(for target: InteractiveLaunchTarget) -> URL? {
        switch target.content {
        case .webURL(let urlString):
            return URL(string: urlString)
        case .html:
            return nil
        }
    }

    private func load() async {
        guard NetworkMonitor.shared.isOnline else {
            loading = false
            return
        }
        guard let token = session.accessToken else { return }
        loading = true
        errorMessage = nil
        defer { loading = false }
        do {
            target = try await LaunchContainerLoader.resolveLaunchTarget(
                courseCode: course.courseCode,
                item: item,
                accessToken: token
            )
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.modules.loadError")
            target = nil
        }
    }

    private func forwardH5PXAPI(packageId: String?, statement: [String: Any]) async {
        guard let packageId, let token = session.accessToken else { return }
        try? await LMSAPI.postXAPIStatement(
            courseCode: course.courseCode,
            packageId: packageId,
            statement: statement,
            accessToken: token
        )
        await onProgressChanged?()
    }
}

enum LaunchContainerLoader {
    static func resolveLaunchTarget(
        courseCode: String,
        item: CourseStructureItem,
        accessToken: String
    ) async throws -> InteractiveLaunchTarget {
        guard let kind = InteractiveLaunchLogic.kind(for: item.kind) else {
            throw APIError.httpStatus(404, message: L.text("mobile.modules.loadError"))
        }
        switch kind {
        case .h5p:
            let payload = try await LMSAPI.fetchModuleH5P(
                courseCode: courseCode,
                itemId: item.id,
                accessToken: accessToken
            )
            guard payload.extractStatus == "ready", !payload.packageId.isEmpty else {
                throw APIError.httpStatus(503, message: L.text("mobile.modules.interactive.preparing"))
            }
            let renderPath = InteractiveLaunchLogic.h5pRenderPath(
                courseCode: courseCode,
                packageId: payload.packageId
            )
            return InteractiveLaunchTarget(
                title: payload.title.isEmpty ? item.title : payload.title,
                kind: .h5p,
                content: .webURL(InteractiveLaunchLogic.resolveURL(renderPath)),
                packageId: payload.packageId,
                hasResume: false
            )
        case .scorm:
            let payload = try await LMSAPI.fetchModuleScorm(
                courseCode: courseCode,
                itemId: item.id,
                accessToken: accessToken
            )
            guard payload.extractStatus == "ready" else {
                throw APIError.httpStatus(503, message: L.text("mobile.modules.interactive.preparing"))
            }
            guard let scoId = payload.scos.first?.id, !scoId.isEmpty else {
                throw APIError.httpStatus(404, message: L.text("mobile.modules.loadError"))
            }
            let launch = try await LMSAPI.launchScorm(
                courseCode: courseCode,
                scoId: scoId,
                accessToken: accessToken
            )
            guard !launch.renderUrl.isEmpty else {
                throw APIError.httpStatus(500, message: L.text("mobile.modules.loadError"))
            }
            return InteractiveLaunchTarget(
                title: payload.title.isEmpty ? item.title : payload.title,
                kind: .scorm,
                content: .webURL(InteractiveLaunchLogic.resolveURL(launch.renderUrl)),
                packageId: nil,
                hasResume: InteractiveLaunchLogic.scormHasResume(initialCmi: launch.initialCmi ?? [:])
            )
        case .ltiLink:
            let meta = try await LMSAPI.fetchModuleLtiLink(
                courseCode: courseCode,
                itemId: item.id,
                accessToken: accessToken
            )
            let ticket = try await LMSAPI.postLtiEmbedTicket(
                courseCode: courseCode,
                itemId: item.id,
                accessToken: accessToken
            )
            guard !ticket.ticket.isEmpty else {
                throw APIError.httpStatus(500, message: L.text("mobile.modules.interactive.ltiError"))
            }
            let framePath = InteractiveLaunchLogic.ltiFramePath(ticket: ticket.ticket)
            return InteractiveLaunchTarget(
                title: meta.title.isEmpty ? item.title : meta.title,
                kind: .ltiLink,
                content: .webURL(InteractiveLaunchLogic.resolveURL(framePath)),
                packageId: nil,
                hasResume: false
            )
        }
    }
}

struct AuthenticatedLaunchWebView: UIViewRepresentable {
    let target: InteractiveLaunchTarget
    let accessToken: String?
    var onWebError: () -> Void
    var onH5PXAPI: ([String: Any]) async -> Void = { _ in }
    var onActivityEvent: () async -> Void = {}

    func makeCoordinator() -> Coordinator {
        Coordinator(
            onWebError: onWebError,
            onH5PXAPI: onH5PXAPI,
            onActivityEvent: onActivityEvent
        )
    }

    func makeUIView(context: Context) -> WKWebView {
        let config = WKWebViewConfiguration()
        config.defaultWebpagePreferences.allowsContentJavaScript = true
        config.allowsInlineMediaPlayback = true
        config.mediaTypesRequiringUserActionForPlayback = []
        config.userContentController.add(context.coordinator, name: "lexturesInteractive")

        let webView = WKWebView(frame: .zero, configuration: config)
        webView.navigationDelegate = context.coordinator
        webView.isOpaque = false
        webView.backgroundColor = .clear
        load(into: webView, coordinator: context.coordinator)
        return webView
    }

    func updateUIView(_ webView: WKWebView, context: Context) {
        if context.coordinator.lastTarget != target {
            context.coordinator.lastTarget = target
            load(into: webView, coordinator: context.coordinator)
        }
    }

    private func load(into webView: WKWebView, coordinator: Coordinator) {
        guard let accessToken else {
            onWebError()
            return
        }
        let script = WKUserScript(
            source: InteractiveLaunchLogic.authInjectionScript(
                accessToken: accessToken,
                apiBase: AppConfiguration.apiBaseURL.absoluteString
            ),
            injectionTime: .atDocumentStart,
            forMainFrameOnly: false
        )
        webView.configuration.userContentController.removeAllUserScripts()
        webView.configuration.userContentController.addUserScript(script)

        switch target.content {
        case .webURL(let urlString):
            guard let url = URL(string: urlString) else {
                onWebError()
                return
            }
            var request = URLRequest(url: url)
            if url.absoluteString.hasPrefix(AppConfiguration.apiBaseURL.absoluteString) {
                request.setValue("Bearer \(accessToken)", forHTTPHeaderField: "Authorization")
            }
            webView.load(request)
        case .html(let html):
            webView.loadHTMLString(html, baseURL: AppConfiguration.apiBaseURL)
        }
    }

    final class Coordinator: NSObject, WKNavigationDelegate, WKScriptMessageHandler {
        var lastTarget: InteractiveLaunchTarget?
        let onWebError: () -> Void
        let onH5PXAPI: ([String: Any]) async -> Void
        let onActivityEvent: () async -> Void

        init(
            onWebError: @escaping () -> Void,
            onH5PXAPI: @escaping ([String: Any]) async -> Void,
            onActivityEvent: @escaping () async -> Void
        ) {
            self.onWebError = onWebError
            self.onH5PXAPI = onH5PXAPI
            self.onActivityEvent = onActivityEvent
        }

        func userContentController(
            _ userContentController: WKUserContentController,
            didReceive message: WKScriptMessage
        ) {
            guard message.name == "lexturesInteractive",
                  let body = message.body as? [String: Any],
                  body["type"] as? String == "h5p-xapi",
                  let statement = body["statement"] as? [String: Any] else { return }
            Task {
                await onH5PXAPI(statement)
                await onActivityEvent()
            }
        }

        func webView(_ webView: WKWebView, didFail navigation: WKNavigation!, withError error: Error) {
            onWebError()
        }

        func webView(_ webView: WKWebView, didFailProvisionalNavigation navigation: WKNavigation!, withError error: Error) {
            onWebError()
        }
    }
}
