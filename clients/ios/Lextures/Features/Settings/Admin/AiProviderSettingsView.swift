import SwiftUI

/// AI provider picker and masked BYOK secret field (M14.5).
struct AiProviderSettingsView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme

    @State private var available = false
    @State private var providers: [String] = []
    @State private var modelAliases: [String] = []
    @State private var provider = "openrouter"
    @State private var modelAlias = "claude-3-5-sonnet"
    @State private var fallbackProvider = ""
    @State private var byokKey = ""
    @State private var byokConfigured = false
    @State private var loading = true
    @State private var saveStatus: OrgBrandingAdminLogic.SaveStatus = .idle
    @State private var testing = false
    @State private var testMessage: String?

    var body: some View {
        Group {
            if available {
                LMSCard {
                    VStack(alignment: .leading, spacing: 16) {
                        Text(L.text("mobile.admin.orgBranding.aiProvider.title"))
                            .font(.headline)
                            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))

                        Text(L.text("mobile.admin.orgBranding.aiProvider.intro"))
                            .font(.subheadline)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                        if loading {
                            ProgressView()
                                .frame(maxWidth: .infinity)
                        } else {
                            providerPicker
                            modelAliasPicker
                            fallbackPicker
                            byokField
                            actionButtons
                        }
                    }
                }
            }
        }
        .task { await load() }
    }

    private var providerPicker: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text(L.text("mobile.admin.orgBranding.aiProvider.provider"))
                .font(.subheadline.weight(.semibold))
            Picker("", selection: $provider) {
                ForEach(providers, id: \.self) { value in
                    Text(OrgBrandingAdminLogic.providerLabel(value)).tag(value)
                }
            }
            .pickerStyle(.menu)
        }
    }

    private var modelAliasPicker: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text(L.text("mobile.admin.orgBranding.aiProvider.modelAlias"))
                .font(.subheadline.weight(.semibold))
            Picker("", selection: $modelAlias) {
                ForEach(modelAliases, id: \.self) { value in
                    Text(value).tag(value)
                }
            }
            .pickerStyle(.menu)
        }
    }

    private var fallbackPicker: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text(L.text("mobile.admin.orgBranding.aiProvider.fallback"))
                .font(.subheadline.weight(.semibold))
            Picker("", selection: $fallbackProvider) {
                Text(L.text("mobile.admin.orgBranding.aiProvider.fallbackNone")).tag("")
                ForEach(providers, id: \.self) { value in
                    Text(OrgBrandingAdminLogic.providerLabel(value)).tag(value)
                }
            }
            .pickerStyle(.menu)
        }
    }

    private var byokField: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text(L.text("mobile.admin.orgBranding.aiProvider.byokKey"))
                .font(.subheadline.weight(.semibold))
            SecureField(
                OrgBrandingAdminLogic.platformSecretPlaceholder,
                text: $byokKey
            )
            .textInputAutocapitalization(.never)
            .autocorrectionDisabled()
            .textFieldStyle(.roundedBorder)
            if byokConfigured {
                Text(L.text("mobile.admin.orgBranding.aiProvider.byokConfigured"))
                    .font(.caption)
                    .foregroundStyle(.green)
            }
            Text(L.text("mobile.admin.orgBranding.aiProvider.byokHint"))
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
        }
    }

    private var actionButtons: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack(spacing: 12) {
                Button {
                    Task { await save() }
                } label: {
                    if case .saving = saveStatus {
                        ProgressView()
                    } else {
                        Text(L.text("mobile.admin.orgBranding.aiProvider.save"))
                    }
                }
                .buttonStyle(.borderedProminent)
                .tint(LexturesTheme.brandTeal)
                .disabled(loading || saveStatus == .saving || testing)

                Button {
                    Task { await testConnection() }
                } label: {
                    if testing {
                        ProgressView()
                    } else {
                        Text(L.text("mobile.admin.orgBranding.aiProvider.test"))
                    }
                }
                .buttonStyle(.bordered)
                .disabled(loading || saveStatus == .saving || testing)
            }

            switch saveStatus {
            case .saved:
                Text(L.text("mobile.admin.orgBranding.aiProvider.saved"))
                    .font(.caption)
                    .foregroundStyle(.green)
            case .error(let message):
                Text(message)
                    .font(.caption)
                    .foregroundStyle(.red)
            default:
                EmptyView()
            }

            if let testMessage {
                Text(testMessage)
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
        }
    }

    private func load() async {
        guard let token = session.accessToken else {
            loading = false
            return
        }
        loading = true
        defer { loading = false }
        do {
            let response = try await LMSAPI.fetchAiProviderSettings(accessToken: token)
            available = true
            providers = response.providers ?? Array(OrgBrandingAdminLogic.providerLabels.keys)
            modelAliases = response.modelAliases ?? ["claude-3-5-sonnet", "gpt-4o", "gemini-1.5-pro"]
            provider = response.provider ?? provider
            modelAlias = response.modelAlias ?? modelAlias
            fallbackProvider = response.fallbackProvider ?? ""
            byokConfigured = response.byokConfigured == true
            byokKey = OrgBrandingAdminLogic.byokFieldValue(configured: byokConfigured, draft: byokKey)
            saveStatus = .idle
        } catch let error as APIError {
            if case let .httpStatus(code, _) = error, code == 404 || code == 403 {
                available = false
            } else {
                available = true
                saveStatus = .error(OrgBrandingAdminLogic.userFacingError(
                    error,
                    fallbackKey: "mobile.admin.orgBranding.aiProvider.loadError"
                ))
            }
        } catch {
            available = true
            saveStatus = .error(OrgBrandingAdminLogic.userFacingError(
                error,
                fallbackKey: "mobile.admin.orgBranding.aiProvider.loadError"
            ))
        }
    }

    private func save() async {
        guard let token = session.accessToken else { return }
        saveStatus = .saving
        let request = OrgBrandingAdminLogic.buildAiProviderSaveRequest(
            provider: provider,
            modelAlias: modelAlias,
            fallbackProvider: fallbackProvider,
            byokKey: byokKey
        )
        do {
            let response = try await LMSAPI.putAiProviderSettings(body: request, accessToken: token)
            provider = response.provider ?? provider
            modelAlias = response.modelAlias ?? modelAlias
            fallbackProvider = response.fallbackProvider ?? ""
            byokConfigured = response.byokConfigured == true
            byokKey = OrgBrandingAdminLogic.platformSecretPlaceholder
            saveStatus = .saved
        } catch {
            saveStatus = .error(OrgBrandingAdminLogic.userFacingError(
                error,
                fallbackKey: "mobile.admin.orgBranding.aiProvider.saveError"
            ))
        }
    }

    private func testConnection() async {
        guard let token = session.accessToken else { return }
        testing = true
        defer { testing = false }
        do {
            let response = try await LMSAPI.testAiProviderConnection(accessToken: token)
            let latency = response.latencyMs.map(String.init) ?? "?"
            let preview = response.responsePreview ?? "OK"
            let providerName = response.provider ?? provider
            testMessage = L.format(
                "mobile.admin.orgBranding.aiProvider.testSuccess",
                providerName,
                latency,
                preview
            )
        } catch {
            testMessage = OrgBrandingAdminLogic.userFacingError(
                error,
                fallbackKey: "mobile.admin.orgBranding.aiProvider.testError"
            )
        }
    }
}
