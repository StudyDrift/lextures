import SwiftUI

enum AuthScreen {
    case login
    case signup
    case mfa
}

struct AuthFlowView: View {
    @Environment(AuthSession.self) private var session
    @State private var screen: AuthScreen = .login
    @State private var signOutBanner: String?

    var body: some View {
        ZStack {
            PublicAuthBackground()

            Group {
                switch screen {
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
                        bannerMessage: signOutBanner
                    )
                    .transition(.move(edge: .trailing).combined(with: .opacity))
                case .signup:
                    SignupView {
                        withAnimation(.easeInOut(duration: 0.2)) {
                            screen = .login
                        }
                    }
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
            }
        }
    }
}
