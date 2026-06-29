import SwiftUI

struct LoginView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme

    var onCreateAccount: () -> Void

    @State private var email = ""
    @State private var password = ""
    @State private var isLoading = false
    @State private var errorMessage: String?

    var body: some View {
        ScrollView {
            VStack(spacing: 0) {
                authHeader(
                    title: L.text("auth.login.title"),
                    subtitle: L.text("auth.login.subtitle")
                )

                AuthCard {
                    VStack(spacing: 20) {
                        AuthTextField(
                            title: L.text("auth.login.email"),
                            text: $email,
                            placeholder: L.text("auth.login.emailPlaceholder"),
                            keyboard: .emailAddress,
                            textContentType: .username
                        )

                        AuthTextField(
                            title: L.text("auth.login.password"),
                            text: $password,
                            placeholder: "••••••••",
                            isSecure: true,
                            textContentType: .password
                        )

                        if let errorMessage {
                            Text(errorMessage)
                                .font(.subheadline)
                                .foregroundStyle(LexturesTheme.error)
                                .frame(maxWidth: .infinity, alignment: .leading)
                                .accessibilityAddTraits(.isStaticText)
                        }

                        Button(isLoading ? L.text("auth.login.submitting") : L.text("auth.login.submit")) {
                            Task { await submit() }
                        }
                        .buttonStyle(AuthPrimaryButtonStyle())
                        .disabled(isLoading || email.isEmpty || password.isEmpty)
                        .accessibilityLabel(isLoading ? L.text("auth.login.submitting") : L.text("auth.login.submit"))
                        .minimumTapTarget()

                        HStack {
                            Spacer()
                            Text(L.text("auth.login.forgotPassword"))
                                .font(.subheadline)
                                .foregroundStyle(LexturesTheme.primaryMuted)
                                .opacity(0.7)
                        }
                        .accessibilityHint(L.text("auth.login.forgotPasswordHint"))

                        footerLink
                    }
                }
            }
            .padding(.horizontal, 20)
            .padding(.vertical, 24)
        }
        .scrollDismissesKeyboard(.automatic)
    }

    private var footerLink: some View {
        HStack(spacing: 4) {
            Text(L.text("auth.login.newHere"))
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            Button(L.text("auth.login.createAccount"), action: onCreateAccount)
                .font(.subheadline.weight(.medium))
                .foregroundStyle(LexturesTheme.primaryMuted)
        }
        .font(.subheadline)
        .frame(maxWidth: .infinity)
        .padding(.top, 4)
    }

    @ViewBuilder
    private func authHeader(title: String, subtitle: String) -> some View {
        VStack(spacing: 20) {
            BrandLogoView(maxHeight: 56)
                .accessibilityHidden(true)

            VStack(spacing: 8) {
                Text(title)
                    .font(.system(.title, design: .serif).weight(.semibold))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    .multilineTextAlignment(.center)

                Text(subtitle)
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    .multilineTextAlignment(.center)
                    .fixedSize(horizontal: false, vertical: true)
            }
        }
        .padding(.bottom, 32)
    }

    @MainActor
    private func submit() async {
        isLoading = true
        errorMessage = nil
        defer { isLoading = false }

        do {
            let response = try await AuthAPI.login(email: email.trimmingCharacters(in: .whitespacesAndNewlines), password: password)
            try session.applyTokenResponse(response)
        } catch let error as AuthSession.AuthSessionError {
            errorMessage = error.localizedDescription
        } catch let error as APIError {
            if case .transport = error {
                errorMessage = session.serverUnreachableMessage()
            } else {
                errorMessage = error.localizedDescription
            }
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}
