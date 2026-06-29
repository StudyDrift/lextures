import SwiftUI

struct RootView: View {
    @Environment(AuthSession.self) private var session
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
                MainTabView()
                    .transition(.opacity)
                    .environment(OfflineService.shared)
                    .task {
                        OfflineService.shared.configure(accessToken: session.accessToken)
                        await OfflineService.shared.syncNow(accessToken: session.accessToken)
                        // Keep the access token fresh while the app stays open.
                        while !Task.isCancelled {
                            try? await Task.sleep(for: .seconds(10 * 60))
                            await session.refreshIfNeeded()
                        }
                    }
            }
        }
        .animation(.easeInOut(duration: 0.35), value: session.phase)
        .onChange(of: scenePhase) { _, newPhase in
            // Returning from background: the token has likely expired in the meantime.
            if newPhase == .active, session.phase == .authenticated {
                Task {
                    await session.refreshIfNeeded()
                    await OfflineService.shared.syncNow(accessToken: session.accessToken)
                }
            }
        }
        .onChange(of: networkMonitor.isOnline) { _, online in
            if online, session.phase == .authenticated {
                Task { await OfflineService.shared.syncNow(accessToken: session.accessToken) }
            }
        }
    }
}
