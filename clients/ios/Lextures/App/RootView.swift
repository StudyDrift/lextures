import SwiftUI

struct RootView: View {
    @Environment(AuthSession.self) private var session
    @Environment(BiometricGate.self) private var biometricGate
    @Environment(\.scenePhase) private var scenePhase
    @Bindable private var networkMonitor = NetworkMonitor.shared

    var body: some View {
        Group {
            switch session.phase {
            case .splash:
                SplashView()
                    .task {
                        // Show splash first, then refresh. Avoids overlapping Swift concurrency +
                        // URLSession work at process start (problematic on iOS 26/27 betas).
                        try? await Task.sleep(for: .milliseconds(900))
                        await session.refreshIfNeeded()
                        withAnimation(.easeInOut(duration: 0.35)) {
                            session.finishSplash()
                        }
                    }
            case .unauthenticated:
                AuthFlowView()
                    .transition(.opacity)
            case .authenticated:
                AuthenticatedRootView()
                    .transition(.opacity)
            }
        }
        .animation(.easeInOut(duration: 0.35), value: session.phase)
        .onOpenURL { url in
            Task { await handleIncomingURL(url) }
        }
        .onChange(of: scenePhase) { _, newPhase in
            switch newPhase {
            case .background:
                biometricGate.recordBackground()
            case .active:
                biometricGate.evaluateOnForeground()
                if session.phase == .authenticated {
                    Task {
                        await session.refreshIfNeeded()
                        await OfflineService.shared.syncNow(accessToken: session.accessToken)
                    }
                }
            case .inactive:
                break
            @unknown default:
                break
            }
        }
        .onChange(of: networkMonitor.isOnline) { _, online in
            if online, session.phase == .authenticated {
                Task { await OfflineService.shared.syncNow(accessToken: session.accessToken) }
            }
        }
    }

    @MainActor
    private func handleIncomingURL(_ url: URL) async {
        if AuthCallbackParser.parse(url.absoluteString) != nil {
            do {
                try await session.handleAuthCallback(AuthCallbackParser.parse(url.absoluteString)!)
            } catch AuthSession.AuthSessionError.mfaRequired {
                // MFA state stored; AuthFlowView shows the challenge screen.
            } catch {
                // Ignore invalid/expired deep links at the root; login UI handles user-initiated flows.
            }
            return
        }
        if session.phase == .authenticated {
            // Navigation deep links are handled inside MainTabView as well.
        }
    }
}
