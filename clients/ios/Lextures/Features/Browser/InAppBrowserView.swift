import SwiftUI
import WebKit
import UniformTypeIdentifiers

/// Full-screen in-app browser (MB.1) — X/Instagram shape over the current screen.
struct InAppBrowserView: View {
    @Environment(\.lxReduceMotion) private var reduceMotion
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.accessibilityReduceMotion) private var a11yReduceMotion

    let session: InAppBrowserSession
    var onDismiss: () -> Void

    @State private var currentURL: URL
    @State private var pageTitle: String = ""
    @State private var progress: Double = 0
    @State private var canGoBack = false
    @State private var loadState: LoadState = .loading
    @State private var showCopiedToast = false
    @State private var showOverflow = false
    @State private var dragOffset: CGFloat = 0

    enum LoadState: Equatable {
        case loading
        case loaded
        case error(String)
        case offline
        case blocked
    }

    init(session: InAppBrowserSession, onDismiss: @escaping () -> Void) {
        self.session = session
        self.onDismiss = onDismiss
        _currentURL = State(initialValue: session.initialURL)
    }

    var body: some View {
        VStack(spacing: 0) {
            BrowserChromeHeader(
                host: MobileLinkPolicy.displayHost(for: currentURL),
                title: pageTitle,
                isSecure: MobileLinkPolicy.isSecure(currentURL),
                progress: progress,
                canGoBack: canGoBack,
                onClose: { dismiss(method: "close") },
                onBack: { NotificationCenter.default.post(name: .inAppBrowserGoBack, object: nil) },
                onCopy: copyCurrentURL,
                onOverflow: { showOverflow = true }
            )
            .gesture(
                DragGesture(minimumDistance: 12)
                    .onChanged { value in
                        if value.translation.height > 0 {
                            dragOffset = value.translation.height
                        }
                    }
                    .onEnded { value in
                        if value.translation.height > 120 {
                            dismiss(method: "drag")
                        } else {
                            withAnimation(LexturesMotion.bubble) { dragOffset = 0 }
                        }
                    }
            )

            ZStack {
                BrowserWebView(
                    url: session.initialURL,
                    accessToken: session.accessToken,
                    currentURL: $currentURL,
                    pageTitle: $pageTitle,
                    progress: $progress,
                    canGoBack: $canGoBack,
                    loadState: $loadState
                )
                .opacity(loadState == .loaded || loadState == .loading ? 1 : 0)

                if case .error(let message) = loadState {
                    BrowserErrorState(
                        kind: .error,
                        message: message,
                        onRetry: { NotificationCenter.default.post(name: .inAppBrowserReload, object: nil) },
                        onOpenExternal: openInSystemBrowser
                    )
                } else if loadState == .offline {
                    BrowserErrorState(
                        kind: .offline,
                        message: L.text("mobile.browser.offlineBody"),
                        onRetry: { NotificationCenter.default.post(name: .inAppBrowserReload, object: nil) },
                        onOpenExternal: openInSystemBrowser
                    )
                } else if loadState == .blocked {
                    BrowserErrorState(
                        kind: .blocked,
                        message: L.text("mobile.browser.blockedByPolicy"),
                        onRetry: nil,
                        onOpenExternal: nil
                    )
                }
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity)
        }
        .background(LexturesTheme.sceneBackground(for: colorScheme))
        .offset(y: max(0, dragOffset))
        .accessibilityAddTraits(.isModal)
        .accessibilityElement(children: .contain)
        .confirmationDialog(L.text("mobile.browser.overflowTitle"), isPresented: $showOverflow, titleVisibility: .hidden) {
            Button(L.text("mobile.browser.copyLink")) { copyCurrentURL() }
            Button(L.text("mobile.browser.share")) { shareCurrentURL() }
            Button(L.text("mobile.browser.openInBrowser")) { openInSystemBrowser() }
            Button(L.text("mobile.browser.reload")) {
                NotificationCenter.default.post(name: .inAppBrowserReload, object: nil)
            }
            if session.allowReport {
                Button(L.text("mobile.browser.report")) {
                    // Report surface is host-owned; emit content-free counter only.
                    InAppBrowserTelemetry.record(
                        MobileLinkPolicy.sanitizeTelemetry([
                            "source": session.source,
                            "classification": "external",
                            "outcome": "report",
                        ])
                    )
                }
            }
            Button(L.text("mobile.browser.close"), role: .cancel) {}
        }
        .overlay(alignment: .bottom) {
            if showCopiedToast {
                Text(L.text("mobile.browser.linkCopied"))
                    .font(.subheadline.weight(.semibold))
                    .padding(.horizontal, 16)
                    .padding(.vertical, 10)
                    .background(.ultraThinMaterial, in: Capsule())
                    .padding(.bottom, 32)
                    .transition(.opacity)
                    .accessibilityLabel(L.text("mobile.browser.linkCopied"))
            }
        }
        .onAppear {
            UIAccessibility.post(
                notification: .screenChanged,
                argument: L.text("mobile.browser.copyLink") + ", " + MobileLinkPolicy.displayHost(for: currentURL)
            )
        }
    }

    private func copyCurrentURL() {
        UIPasteboard.general.string = currentURL.absoluteString
        let generator = UINotificationFeedbackGenerator()
        generator.notificationOccurred(.success)
        withAnimation { showCopiedToast = true }
        DispatchQueue.main.asyncAfter(deadline: .now() + 1.6) {
            withAnimation { showCopiedToast = false }
        }
        InAppBrowserTelemetry.record(
            MobileLinkPolicy.sanitizeTelemetry([
                "source": session.source,
                "classification": "external",
                "outcome": "copy",
            ])
        )
    }

    private func shareCurrentURL() {
        let items: [Any] = [currentURL]
        let av = UIActivityViewController(activityItems: items, applicationActivities: nil)
        guard let scene = UIApplication.shared.connectedScenes.first as? UIWindowScene,
              let root = scene.keyWindow?.rootViewController else { return }
        root.present(av, animated: true)
        InAppBrowserTelemetry.record(
            MobileLinkPolicy.sanitizeTelemetry([
                "source": session.source,
                "classification": "external",
                "outcome": "share",
            ])
        )
    }

    private func openInSystemBrowser() {
        UIApplication.shared.open(currentURL)
        InAppBrowserTelemetry.record(
            MobileLinkPolicy.sanitizeTelemetry([
                "source": session.source,
                "classification": "external",
                "outcome": "escape_system",
            ])
        )
    }

    private func dismiss(method: String) {
        InAppBrowserTelemetry.record(
            MobileLinkPolicy.sanitizeTelemetry([
                "source": session.source,
                "classification": "external",
                "outcome": "dismiss_\(method)",
            ])
        )
        onDismiss()
    }
}

extension Notification.Name {
    static let inAppBrowserGoBack = Notification.Name("lextures.inAppBrowser.goBack")
    static let inAppBrowserReload = Notification.Name("lextures.inAppBrowser.reload")
}

private extension UIWindowScene {
    var keyWindow: UIWindow? { windows.first { $0.isKeyWindow } }
}
