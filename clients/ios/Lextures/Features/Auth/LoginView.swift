import SwiftUI

struct LoginView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme

    var onCreateAccount: () -> Void
    var onMfaRequired: () -> Void
    var bannerMessage: String?

    @State private var email = ""
    @State private var password = ""
    @State private var isLoading = false
    @State private var ssoLoading = false
    @State private var magicLinkStatus: MagicLinkStatus = .idle
    @State private var errorMessage: String?
    @State private var samlStatus: SamlStatusResponse?
    @State private var oidcStatus: OidcStatusResponse?

    private enum MagicLinkStatus {
        case idle
        case sending
        case sent
        case error
    }

    private var forceSaml: Bool {
        samlStatus?.idp?.forceSaml == true
    }

    var body: some View {
        ScrollView {
            VStack(spacing: 0) {
                authHeader(
                    title: L.text("auth.login.title"),
                    subtitle: L.text("auth.login.subtitle")
                )

                AuthCard {
                    VStack(spacing: 20) {
                        if let bannerMessage {
                            Text(bannerMessage)
                                .font(.subheadline)
                                .foregroundStyle(LexturesTheme.error)
                                .frame(maxWidth: .infinity, alignment: .leading)
                                .accessibilityLabel(bannerMessage)
                        }

                        if let providers = ssoProviders, !providers.isEmpty {
                            ssoButtons(providers)
                        }

                        if forceSaml {
                            Text(L.text("auth.login.ssoRequired"))
                                .font(.subheadline)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                                .frame(maxWidth: .infinity, alignment: .leading)
                        }

                        if !forceSaml {
                            passwordForm
                            magicLinkSection
                            footerLink
                        }
                    }
                }
            }
            .padding(.horizontal, 20)
            .padding(.vertical, 24)
        }
        .scrollDismissesKeyboard(.automatic)
        .task {
            async let saml = AuthAPI.fetchSamlStatus()
            async let oidc = AuthAPI.fetchOidcStatus()
            samlStatus = await saml
            oidcStatus = await oidc
        }
    }

    @ViewBuilder
    private var passwordForm: some View {
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
        }

        Button(isLoading ? L.text("auth.login.submitting") : L.text("auth.login.submit")) {
            Task { await submitPassword() }
        }
        .buttonStyle(AuthPrimaryButtonStyle())
        .disabled(isLoading || ssoLoading || email.isEmpty || password.isEmpty)

        HStack {
            Spacer()
            Text(L.text("auth.login.forgotPassword"))
                .font(.subheadline)
                .foregroundStyle(LexturesTheme.primaryMuted)
                .opacity(0.7)
        }
        .accessibilityHint(L.text("auth.login.forgotPasswordHint"))
    }

    @ViewBuilder
    private var magicLinkSection: some View {
        VStack(alignment: .leading, spacing: 12) {
            Divider()
                .padding(.vertical, 4)

            Text(L.text("auth.login.magicLinkTitle"))
                .font(.subheadline.weight(.semibold))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))

            Text(L.text("auth.login.magicLinkHint"))
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

            if magicLinkStatus == .sent {
                Text(L.text("auth.login.magicLinkSent"))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            } else {
                Button(magicLinkLabel) {
                    Task { await sendMagicLink() }
                }
                .buttonStyle(AuthOutlineButtonStyle())
                .disabled(magicLinkStatus == .sending || email.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
            }
        }
    }

    private var magicLinkLabel: String {
        switch magicLinkStatus {
        case .sending:
            return L.text("auth.login.magicLinkSending")
        default:
            return L.text("auth.login.sendMagicLink")
        }
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

    private var ssoProviders: [SSOProvider]? {
        var items: [SSOProvider] = []
        if let idp = samlStatus?.enabled == true ? samlStatus?.idp : nil {
            items.append(.saml(idpId: idp.id))
        }
        if let oidc = oidcStatus {
            if oidc.showsClever {
                items.append(.oidc(path: "/auth/clever/login", label: "Clever"))
            }
            if oidc.showsClassLink {
                items.append(.oidc(path: "/auth/oidc/classlink/login", label: "ClassLink"))
            }
            if oidc.enabled == true {
                if oidc.google == true {
                    items.append(.oidc(path: "/auth/oidc/google/login", label: "Google"))
                }
                if oidc.microsoft == true {
                    items.append(.oidc(path: "/auth/oidc/microsoft/login", label: "Microsoft"))
                }
                if oidc.apple == true {
                    items.append(.oidc(path: "/auth/oidc/apple/login", label: "Apple"))
                }
                if let custom = oidc.custom {
                    for provider in custom {
                        items.append(.oidc(
                            path: "/auth/oidc/custom/login?configId=\(provider.id)",
                            label: provider.displayName
                        ))
                    }
                }
            }
        }
        return items.isEmpty ? nil : items
    }

    @ViewBuilder
    private func ssoButtons(_ providers: [SSOProvider]) -> some View {
        VStack(spacing: 10) {
            ForEach(Array(providers.enumerated()), id: \.offset) { _, provider in
                Button(ssoLabel(for: provider)) {
                    Task { await startSSO(provider) }
                }
                .buttonStyle(AuthOutlineButtonStyle())
                .disabled(ssoLoading || isLoading)
            }
        }
    }

    private func ssoLabel(for provider: SSOProvider) -> String {
        switch provider {
        case let .saml(_):
            let label = samlStatus?.idp?.label ?? "SSO"
            return L.format("auth.login.ssoButton", label)
        case let .oidc(_, label):
            return L.format("auth.login.ssoButton", label)
        }
    }

    @MainActor
    private func submitPassword() async {
        isLoading = true
        errorMessage = nil
        defer { isLoading = false }

        do {
            let response = try await AuthAPI.login(
                email: email.trimmingCharacters(in: .whitespacesAndNewlines),
                password: password
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

    @MainActor
    private func startSSO(_ provider: SSOProvider) async {
        ssoLoading = true
        errorMessage = nil
        defer { ssoLoading = false }

        do {
            let payload = try await SSOAuthController.start(provider: provider)
            try session.applyTokenResponse(payload.asTokenResponse)
        } catch AuthSession.AuthSessionError.mfaRequired {
            onMfaRequired()
        } catch let error as SSOAuthError {
            if case .cancelled = error {
                errorMessage = error.localizedDescription
            } else {
                errorMessage = error.localizedDescription
            }
        } catch let error as APIError {
            errorMessage = error.localizedDescription
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    @MainActor
    private func sendMagicLink() async {
        magicLinkStatus = .sending
        errorMessage = nil
        do {
            _ = try await AuthAPI.requestMagicLink(
                email: email.trimmingCharacters(in: .whitespacesAndNewlines)
            )
            magicLinkStatus = .sent
        } catch let error as APIError {
            magicLinkStatus = .error
            errorMessage = error.localizedDescription
        } catch {
            magicLinkStatus = .error
            errorMessage = error.localizedDescription
        }
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
}
