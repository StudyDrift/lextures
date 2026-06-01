import SwiftUI

enum AuthScreen {
    case login
    case signup
}

struct AuthFlowView: View {
    @State private var screen: AuthScreen = .login

    var body: some View {
        ZStack {
            PublicAuthBackground()

            Group {
                switch screen {
                case .login:
                    LoginView {
                        withAnimation(.easeInOut(duration: 0.2)) {
                            screen = .signup
                        }
                    }
                    .transition(.move(edge: .trailing).combined(with: .opacity))
                case .signup:
                    SignupView {
                        withAnimation(.easeInOut(duration: 0.2)) {
                            screen = .login
                        }
                    }
                    .transition(.move(edge: .leading).combined(with: .opacity))
                }
            }
        }
    }
}
