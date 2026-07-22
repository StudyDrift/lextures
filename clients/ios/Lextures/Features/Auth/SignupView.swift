import AuthenticationServices
import SwiftUI

struct SignupView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme

    var onSignIn: () -> Void
    var onMfaRequired: () -> Void = {}

    @State private var displayName = ""
    @State private var email = ""
    @State private var password = ""
    @State private var registerAsParent = false
    @State private var policy: PasswordPolicy = .fallback
    @State private var isLoading = false
    @State private var appleLoading = false
    @State private var appleRawNonce: String = ""
    @State private var errorMessage: String?
    @State private var oidcStatus: OidcStatusResponse?

    private var timezone: String {
        TimeZone.current.identifier
    }

    private var showNativeApple: Bool {
        oidcStatus?.showsAppleNative == true
    }

    var body: some View {
        ScrollView {
            VStack(spacing: 0) {
                authHeader(
                    title: "Create your account",
                    subtitle: "One account for courses, assignments, and messages. If your school uses SSO, you can sign in that way later."
                )

                AuthCard {
                    VStack(spacing: 20) {
                        if showNativeApple {
                            SignInWithAppleButton(.signUp) { request in
                                let raw = AppleSignInController.randomNonceString()
                                appleRawNonce = raw
                                request.requestedScopes = [.fullName, .email]
                                request.nonce = AppleSignInController.sha256Hex(raw)
                            } onCompletion: { result in
                                Task { await handleAppleAuthorization(result) }
                            }
                            .signInWithAppleButtonStyle(colorScheme == .dark ? .white : .black)
                            .frame(height: 48)
                            .frame(maxWidth: .infinity)
                            .disabled(appleLoading || isLoading)
                            .accessibilityIdentifier("auth.signup.signInWithApple")

                            HStack {
                                Rectangle()
                                    .fill(LexturesTheme.fieldBorder(for: colorScheme))
                                    .frame(height: 1)
                                Text(L.text("auth.login.orDivider"))
                                    .font(.caption)
                                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                                Rectangle()
                                    .fill(LexturesTheme.fieldBorder(for: colorScheme))
                                    .frame(height: 1)
                            }
                        }

                        AuthTextField(
                            title: "Display name (optional)",
                            text: $displayName,
                            placeholder: "Alex",
                            textContentType: .nickname,
                            autocapitalization: .words
                        )

                        AuthTextField(
                            title: "Email",
                            text: $email,
                            placeholder: "you@school.edu",
                            keyboard: .emailAddress,
                            textContentType: .emailAddress
                        )

                        VStack(alignment: .leading, spacing: 6) {
                            AuthTextField(
                                title: "Password",
                                text: $password,
                                placeholder: "At least \(policy.minLength) characters",
                                isSecure: true,
                                textContentType: .newPassword
                            )

                            passwordRequirements
                            passwordStrength
                        }

                        Toggle(isOn: $registerAsParent) {
                            Text("I am registering as a parent or guardian for read-only access when my school links my account to a student.")
                                .font(.subheadline)
                                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                                .fixedSize(horizontal: false, vertical: true)
                        }
                        .toggleStyle(SwitchToggleStyle(tint: LexturesTheme.primary))
                        .padding(12)
                        .background(
                            RoundedRectangle(cornerRadius: 8, style: .continuous)
                                .fill(colorScheme == .dark ? Color.white.opacity(0.04) : Color.black.opacity(0.02))
                        )
                        .overlay(
                            RoundedRectangle(cornerRadius: 8, style: .continuous)
                                .stroke(LexturesTheme.fieldBorder(for: colorScheme), lineWidth: 1)
                        )

                        if let errorMessage {
                            Text(errorMessage)
                                .font(.subheadline)
                                .foregroundStyle(LexturesTheme.error)
                                .frame(maxWidth: .infinity, alignment: .leading)
                        }

                        Button(isLoading ? "Creating account…" : "Create account") {
                            Task { await submit() }
                        }
                        .buttonStyle(AuthPrimaryButtonStyle())
                        .disabled(isLoading || appleLoading || email.isEmpty || password.count < policy.minLength)

                        HStack(spacing: 4) {
                            Text("Already have an account?")
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            Button("Sign in", action: onSignIn)
                                .font(.subheadline.weight(.medium))
                                .foregroundStyle(LexturesTheme.primaryMuted)
                        }
                        .font(.subheadline)
                        .frame(maxWidth: .infinity)
                        .padding(.top, 4)
                    }
                }
            }
            .padding(.horizontal, 20)
            .padding(.vertical, 24)
        }
        .scrollDismissesKeyboard(.automatic)
        .task {
            async let pol = AuthAPI.fetchPasswordPolicy()
            async let oidc = AuthAPI.fetchOidcStatus()
            policy = await pol
            oidcStatus = await oidc
        }
    }

    private var passwordRequirements: some View {
        VStack(alignment: .leading, spacing: 4) {
            Text("At least \(policy.minLength) characters")
            if policy.requireUpper { Text("One uppercase letter") }
            if policy.requireLower { Text("One lowercase letter") }
            if policy.requireDigit { Text("One digit") }
            if policy.requireSpecial { Text("One symbol or punctuation character") }
            if policy.checkHibp {
                Text("Must not appear in known public breach lists (checked securely)")
            }
        }
        .font(.caption)
        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
    }

    private var passwordStrength: some View {
        let strength = PasswordStrength.evaluate(password)
        return HStack(spacing: 8) {
            Text("Strength:")
                .font(.caption.weight(.medium))
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            Text(strength.label)
                .font(.caption.weight(.semibold))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            GeometryReader { geo in
                ZStack(alignment: .leading) {
                    Capsule().fill(Color.gray.opacity(0.25))
                    Capsule()
                        .fill(strength.color)
                        .frame(width: geo.size.width * strength.fraction)
                }
            }
            .frame(height: 6)
        }
        .accessibilityElement(children: .combine)
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
            let response = try await AuthAPI.signup(
                email: email.trimmingCharacters(in: .whitespacesAndNewlines),
                password: password,
                displayName: displayName,
                registerAsParent: registerAsParent,
                timezone: timezone
            )
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

    @MainActor
    private func handleAppleAuthorization(_ result: Result<ASAuthorization, Error>) async {
        appleLoading = true
        errorMessage = nil
        defer { appleLoading = false }

        switch result {
        case .failure(let error):
            if let authError = error as? ASAuthorizationError, authError.code == .canceled {
                return
            }
            errorMessage = L.text("auth.social.appleFailed")
        case .success(let authorization):
            guard let credential = authorization.credential as? ASAuthorizationAppleIDCredential,
                  let tokenData = credential.identityToken,
                  let idToken = String(data: tokenData, encoding: .utf8),
                  !idToken.isEmpty
            else {
                errorMessage = L.text("auth.social.appleFailed")
                return
            }
            var authCode: String?
            if let codeData = credential.authorizationCode {
                authCode = String(data: codeData, encoding: .utf8)
            }
            var fullName: String?
            if let name = credential.fullName {
                let parts = [name.givenName, name.familyName].compactMap { $0 }.filter { !$0.isEmpty }
                if !parts.isEmpty { fullName = parts.joined(separator: " ") }
            }
            do {
                let response = try await AuthAPI.nativeAppleSignIn(
                    idToken: idToken,
                    rawNonce: appleRawNonce,
                    authorizationCode: authCode,
                    fullName: fullName,
                    email: credential.email
                )
                try session.applyTokenResponse(response)
            } catch AuthSession.AuthSessionError.mfaRequired {
                onMfaRequired()
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
}

private enum PasswordStrength {
    case weak, fair, strong

    var label: String {
        switch self {
        case .weak: "Weak"
        case .fair: "Fair"
        case .strong: "Strong"
        }
    }

    var fraction: CGFloat {
        switch self {
        case .weak: 0.33
        case .fair: 0.66
        case .strong: 1
        }
    }

    var color: Color {
        switch self {
        case .weak: LexturesTheme.error
        case .fair: .orange
        case .strong: Color(red: 0.047, green: 0.596, blue: 0.341)
        }
    }

    static func evaluate(_ password: String) -> PasswordStrength {
        if password.count < 8 { return .weak }
        var score = 0
        if password.rangeOfCharacter(from: .uppercaseLetters) != nil { score += 1 }
        if password.rangeOfCharacter(from: .lowercaseLetters) != nil { score += 1 }
        if password.rangeOfCharacter(from: .decimalDigits) != nil { score += 1 }
        if password.rangeOfCharacter(from: CharacterSet.punctuationCharacters.union(.symbols)) != nil { score += 1 }
        if password.count >= 12 { score += 1 }
        switch score {
        case 0 ... 2: return .weak
        case 3 ... 4: return .fair
        default: return .strong
        }
    }
}
