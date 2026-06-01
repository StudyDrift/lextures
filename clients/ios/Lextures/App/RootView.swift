import SwiftUI

struct RootView: View {
    @Environment(AuthSession.self) private var session

    var body: some View {
        Group {
            switch session.phase {
            case .splash:
                SplashView()
                    .task {
                        try? await Task.sleep(for: .milliseconds(900))
                        withAnimation(.easeInOut(duration: 0.35)) {
                            session.finishSplash()
                        }
                    }
            case .unauthenticated:
                AuthFlowView()
                    .transition(.opacity)
            case .authenticated:
                PlaceholderHomeView()
                    .transition(.opacity)
            }
        }
        .animation(.easeInOut(duration: 0.35), value: session.phase)
    }
}
