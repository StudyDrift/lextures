import SwiftUI

enum AuthScreen {
    case getStarted
    case login
    case signup
    case mfa
}

struct AuthFlowView: View {
    @Environment(AuthSession.self) private var session
    @State private var screen: AuthScreen = EnvironmentStore.shared.hasSelection ? .login : .getStarted
    @State private var signOutBanner: String?

    var body: some View {
        ZStack {
            PublicAuthBackground()

            Group {
                switch screen {
                case .getStarted:
                    GetStartedView {
                        withAnimation(.easeInOut(duration: 0.2)) {
                            screen = .login
                        }
                    }
                    .transition(.move(edge: .leading).combined(with: .opacity))
                case .login:
                    LoginView(
                        onCreateAccount: {
                            withAnimation(.easeInOut(duration: 0.2)) {
                                screen = .signup
                            }
                        },
                        onMfaRequired: {
                            withAnimation(.easeInOut(duration: 0.2)) {
                                screen = .mfa
                            }
                        },
                        onChangeEnvironment: {
                            EnvironmentStore.shared.clearSelection()
                            withAnimation(.easeInOut(duration: 0.2)) {
                                screen = .getStarted
                            }
                        },
                        bannerMessage: signOutBanner
                    )
                    .transition(.move(edge: .trailing).combined(with: .opacity))
                case .signup:
                    SignupView(
                        onSignIn: {
                            withAnimation(.easeInOut(duration: 0.2)) {
                                screen = .login
                            }
                        },
                        onMfaRequired: {
                            withAnimation(.easeInOut(duration: 0.2)) {
                                screen = .mfa
                            }
                        }
                    )
                    .transition(.move(edge: .leading).combined(with: .opacity))
                case .mfa:
                    MFAChallengeView {
                        withAnimation(.easeInOut(duration: 0.2)) {
                            screen = .login
                        }
                    }
                    .transition(.move(edge: .trailing).combined(with: .opacity))
                }
            }
        }
        .onChange(of: session.mfaRequired) { _, required in
            if required != nil {
                screen = .mfa
            }
        }
        .onAppear {
            signOutBanner = session.consumeSignOutMessage()
            if session.mfaRequired != nil {
                screen = .mfa
            } else if !EnvironmentStore.shared.hasSelection {
                screen = .getStarted
            }
        }
    }
}
