import SwiftUI

struct MFAChallengeView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme

    var onCancel: () -> Void

    @State private var code = ""
    @State private var backup = ""
    @State private var showBackup = false
    @State private var isLoading = false
    @State private var passkeyBusy = false
    @State private var errorMessage: String?
    @State private var totpCredentialId: String?
    @State private var totpSetupUri: String?

    private var mode: MFARequired {
        session.mfaRequired ?? .challenge
    }

    var body: some View {
        ScrollView {
            VStack(spacing: 0) {
                authHeader(
                    title: mode == .setup ? L.text("auth.mfa.setupTitle") : L.text("auth.mfa.title"),
                    subtitle: mode == .setup ? L.text("auth.mfa.setupSubtitle") : L.text("auth.mfa.subtitle")
                )

                AuthCard {
                    VStack(spacing: 20) {
                        if mode == .setup, totpSetupUri == nil {
                            setupChoices
                        }

                        if mode == .setup, totpSetupUri != nil {
                            Text(L.text("auth.mfa.scanInstructions"))
                                .font(.subheadline)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                                .frame(maxWidth: .infinity, alignment: .leading)
                        }

                        if totpSetupUri != nil || mode == .challenge {
                            totpForm
                        }

                        if mode == .challenge {
                            passkeySection
                        }

                        Button(L.text("auth.mfa.cancel"), action: cancel)
                            .font(.subheadline.weight(.medium))
                            .foregroundStyle(LexturesTheme.primaryMuted)
                            .frame(maxWidth: .infinity)
                    }
                }
            }
            .padding(.horizontal, 20)
            .padding(.vertical, 24)
        }
        .scrollDismissesKeyboard(.automatic)
    }

    private var setupChoices: some View {
        VStack(spacing: 12) {
            Button(L.text("auth.mfa.setupTotp")) {
                Task { await startTotpEnrol() }
            }
            .buttonStyle(AuthOutlineButtonStyle())
            .disabled(isLoading)

            Button(passkeyBusy ? L.text("auth.mfa.passkeyWaiting") : L.text("auth.mfa.setupPasskey")) {
                Task { await runPasskeyCeremony(setup: true) }
            }
            .buttonStyle(AuthPrimaryButtonStyle())
            .disabled(passkeyBusy || isLoading)
        }
    }

    private var totpForm: some View {
        VStack(spacing: 16) {
            AuthTextField(
                title: L.text("auth.mfa.code"),
                text: $code,
                placeholder: L.text("auth.mfa.codePlaceholder"),
                keyboard: .numberPad,
                textContentType: .oneTimeCode
            )

            if let errorMessage {
                Text(errorMessage)
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.error)
                    .frame(maxWidth: .infinity, alignment: .leading)
            }

            Button(isLoading ? L.text("auth.mfa.verifying") : submitLabel) {
                Task { await submitTotp() }
            }
            .buttonStyle(AuthPrimaryButtonStyle())
            .disabled(isLoading || code.count != 6)
        }
    }

    private var passkeySection: some View {
        VStack(spacing: 16) {
            Text(L.text("auth.login.orDivider"))
                .font(.caption.weight(.semibold))
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                .textCase(.uppercase)

            Button(passkeyBusy ? L.text("auth.mfa.passkeyWaiting") : L.text("auth.mfa.usePasskey")) {
                Task { await runPasskeyCeremony(setup: false) }
            }
            .buttonStyle(AuthOutlineButtonStyle())
            .disabled(passkeyBusy || isLoading)

            if !showBackup {
                Button(L.text("auth.mfa.useBackup")) {
                    showBackup = true
                }
                .font(.subheadline.weight(.medium))
                .foregroundStyle(LexturesTheme.primaryMuted)
            } else {
                VStack(spacing: 12) {
                    AuthTextField(
                        title: L.text("auth.mfa.backupCode"),
                        text: $backup,
                        placeholder: "ABCD-1234",
                        autocapitalization: .allCharacters
                    )
                    Button(L.text("auth.mfa.verify")) {
                        Task { await submitBackup() }
                    }
                    .buttonStyle(AuthOutlineButtonStyle())
                    .disabled(isLoading || backup.count < 8)
                }
            }
        }
    }

    private var submitLabel: String {
        mode == .setup ? L.text("auth.mfa.confirmEnrolment") : L.text("auth.mfa.verify")
    }

    @MainActor
    private func startTotpEnrol() async {
        guard let token = session.mfaPendingToken else { return }
        isLoading = true
        errorMessage = nil
        defer { isLoading = false }
        do {
            let response = try await AuthAPI.mfaTotpEnrol(mfaPendingToken: token)
            totpCredentialId = response.credentialId
            totpSetupUri = response.otpauthUri
        } catch let error as APIError {
            errorMessage = error.localizedDescription
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    @MainActor
    private func submitTotp() async {
        guard let token = session.mfaPendingToken else { return }
        isLoading = true
        errorMessage = nil
        defer { isLoading = false }
        do {
            if mode == .setup {
                guard let totpCredentialId else { return }
                try await AuthAPI.mfaTotpVerifyEnrol(
                    credentialId: totpCredentialId,
                    code: code,
                    mfaPendingToken: token
                )
                let response = try await AuthAPI.mfaSetupComplete(mfaPendingToken: token)
                try session.applyTokenResponse(response)
            } else {
                let response = try await AuthAPI.mfaTotpChallenge(code: code, mfaPendingToken: token)
                try session.applyTokenResponse(response)
            }
        } catch let error as APIError {
            errorMessage = error.localizedDescription
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    @MainActor
    private func submitBackup() async {
        guard let token = session.mfaPendingToken else { return }
        isLoading = true
        errorMessage = nil
        defer { isLoading = false }
        do {
            let response = try await AuthAPI.mfaBackupChallenge(code: backup, mfaPendingToken: token)
            try session.applyTokenResponse(response)
        } catch let error as APIError {
            errorMessage = error.localizedDescription
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    @MainActor
    private func runPasskeyCeremony(setup: Bool) async {
        guard let token = session.mfaPendingToken else { return }
        passkeyBusy = true
        errorMessage = nil
        defer { passkeyBusy = false }
        do {
            let begin = try await AuthAPI.mfaWebAuthnBegin(setup: setup, mfaPendingToken: token)
            guard let sessionId = begin.sessionId, let options = begin.options else {
                throw MFAChallengeError.passkeyFailed
            }
            let credentialJSON = try await MfaPasskeyController.performCeremony(optionsData: options, setup: setup)
            if setup {
                _ = try await AuthAPI.mfaWebAuthnComplete(
                    setup: true,
                    sessionId: sessionId,
                    credentialJSON: credentialJSON,
                    mfaPendingToken: token
                )
                let response = try await AuthAPI.mfaSetupComplete(mfaPendingToken: token)
                try session.applyTokenResponse(response)
            } else if let response = try await AuthAPI.mfaWebAuthnComplete(
                setup: false,
                sessionId: sessionId,
                credentialJSON: credentialJSON,
                mfaPendingToken: token
            ) {
                try session.applyTokenResponse(response)
            }
        } catch let error as MFAChallengeError {
            errorMessage = error.localizedDescription
        } catch let error as APIError {
            errorMessage = error.localizedDescription
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    private func cancel() {
        session.clearMfaFlow()
        onCancel()
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

struct AuthOutlineButtonStyle: ButtonStyle {
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.isEnabled) private var isEnabled

    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .font(.subheadline.weight(.semibold))
            .frame(maxWidth: .infinity)
            .padding(.vertical, 14)
            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            .background(LexturesTheme.cardBackground(for: colorScheme))
            .clipShape(RoundedRectangle(cornerRadius: 14, style: .continuous))
            .overlay(
                RoundedRectangle(cornerRadius: 14, style: .continuous)
                    .stroke(LexturesTheme.fieldBorder(for: colorScheme), lineWidth: 1)
            )
            .opacity(isEnabled ? 1 : 0.55)
            .scaleEffect(configuration.isPressed ? 0.98 : 1)
    }
}
