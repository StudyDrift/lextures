import SwiftUI

struct RootView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.scenePhase) private var scenePhase

    var body: some View {
        Group {
            switch session.phase {
            case .splash:
                SplashView()
                    .task {
                        // Refresh in parallel with the splash animation so a stale
                        // 15-minute access token is replaced before the app shows.
                        async let minimumSplash: Void? = try? await Task.sleep(for: .milliseconds(900))
                        await session.refreshIfNeeded()
                        _ = await minimumSplash
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
                    .task {
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
                Task { await session.refreshIfNeeded() }
            }
        }
    }
}
