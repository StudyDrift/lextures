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
