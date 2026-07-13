import SwiftUI

/// AI provider card: provider/model picker + write-only BYOK secret (M14.5).
struct AiProviderSettingsView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme

    @Binding var settings: AIProviderSettings?
    @Binding var statusMessage: String?
    @Binding var errorMessage: String?
    let available: Bool

    @State private var provider = OrgBrandingAdminLogic.defaultProvider
    @State private var modelAlias = OrgBrandingAdminLogic.defaultModelAlias
    @State private var fallbackProvider = ""
    @State private var byokKey = ""
    @State private var byokConfigured = false
    @State private var saving = false
    @State private var testing = false

    private var providers: [String] {
        OrgBrandingAdminLogic.providerOptions(from: settings)
    }

    private var modelAliases: [String] {
        OrgBrandingAdminLogic.modelAliasOptions(from: settings)
    }

    var body: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 14) {
                Text(L.text("mobile.admin.orgBranding.provider.title"))
                    .font(.headline)
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                Text(L.text("mobile.admin.orgBranding.provider.intro"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                if !available {
                    Text(L.text("mobile.admin.orgBranding.provider.unavailable"))
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                } else {
                    pickerRow(
                        title: L.text("mobile.admin.orgBranding.provider.provider"),
                        selection: $provider,
                        options: providers
                    ) { OrgBrandingAdminLogic.providerLabel(for: $0) }

                    pickerRow(
                        title: L.text("mobile.admin.orgBranding.provider.modelAlias"),
                        selection: $modelAlias,
                        options: modelAliases
                    ) { $0 }

                    VStack(alignment: .leading, spacing: 6) {
                        Text(L.text("mobile.admin.orgBranding.provider.fallback"))
                            .font(.subheadline.weight(.medium))
                            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                        Picker("", selection: $fallbackProvider) {
                            Text(L.text("mobile.admin.orgBranding.provider.fallbackNone")).tag("")
                            ForEach(providers, id: \.self) { providerOption in
                                Text(OrgBrandingAdminLogic.providerLabel(for: providerOption)).tag(providerOption)
                            }
                        }
                        .pickerStyle(.menu)
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .frame(minHeight: 44)
                    }

                    VStack(alignment: .leading, spacing: 6) {
                        Text(L.text("mobile.admin.orgBranding.provider.byok"))
                            .font(.subheadline.weight(.medium))
                            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                        SecureField(
                            OrgBrandingAdminLogic.secretPlaceholder,
                            text: $byokKey
                        )
                        .textContentType(.password)
                        .autocorrectionDisabled()
                        .textInputAutocapitalization(.never)
                        .padding(12)
                        .background(LexturesTheme.sceneBackground(for: colorScheme), in: RoundedRectangle(cornerRadius: 10))
                        .accessibilityLabel(L.text("mobile.admin.orgBranding.provider.byok"))
                        Text(
                            byokConfigured
                                ? L.text("mobile.admin.orgBranding.provider.byokConfigured")
                                : L.text("mobile.admin.orgBranding.provider.byokHint")
                        )
                        .font(.caption2)
                        .foregroundStyle(
                            byokConfigured
                                ? LexturesTheme.brandTeal
                                : LexturesTheme.textSecondary(for: colorScheme)
                        )
                    }

                    HStack(spacing: 12) {
                        Button {
                            Task { await save() }
                        } label: {
                            if saving {
                                ProgressView().frame(maxWidth: .infinity)
                            } else {
                                Text(L.text("mobile.admin.orgBranding.provider.save"))
                                    .frame(maxWidth: .infinity)
                            }
                        }
                        .buttonStyle(.borderedProminent)
                        .tint(LexturesTheme.brandTeal)
                        .disabled(saving || testing)
                        .frame(minHeight: 44)

                        Button {
                            Task { await testConnection() }
                        } label: {
                            if testing {
                                ProgressView().frame(maxWidth: .infinity)
                            } else {
                                Text(L.text("mobile.admin.orgBranding.provider.test"))
                                    .frame(maxWidth: .infinity)
                            }
                        }
                        .buttonStyle(.bordered)
                        .disabled(saving || testing)
                        .frame(minHeight: 44)
                    }
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
        .onAppear { apply(settings) }
        .onChange(of: settings) { _, next in apply(next) }
    }

    private func pickerRow(
        title: String,
        selection: Binding<String>,
        options: [String],
        label: @escaping (String) -> String
    ) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            Text(title)
                .font(.subheadline.weight(.medium))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            Picker(title, selection: selection) {
                ForEach(options, id: \.self) { option in
                    Text(label(option)).tag(option)
                }
            }
            .pickerStyle(.menu)
            .frame(maxWidth: .infinity, alignment: .leading)
            .frame(minHeight: 44)
        }
    }

    private func apply(_ data: AIProviderSettings?) {
        provider = data?.provider ?? OrgBrandingAdminLogic.defaultProvider
        modelAlias = data?.modelAlias ?? OrgBrandingAdminLogic.defaultModelAlias
        fallbackProvider = data?.fallbackProvider ?? ""
        byokConfigured = data?.byokConfigured ?? false
        byokKey = OrgBrandingAdminLogic.displaySecretField(byokConfigured: byokConfigured)
        if !providers.contains(provider), let first = providers.first {
            provider = first
        }
        if !modelAliases.contains(modelAlias), let first = modelAliases.first {
            modelAlias = first
        }
    }

    @MainActor
    private func save() async {
        guard let token = session.accessToken else { return }
        saving = true
        errorMessage = nil
        statusMessage = nil
        defer { saving = false }
        do {
            let body = OrgBrandingAdminLogic.aiProviderPutBody(
                provider: provider,
                modelAlias: modelAlias,
                fallbackProvider: fallbackProvider,
                byokKey: byokKey
            )
            let saved = try await LMSAPI.putAIProviderSettings(body: body, accessToken: token)
            settings = saved
            apply(saved)
            statusMessage = L.text("mobile.admin.orgBranding.provider.saved")
        } catch {
            errorMessage = OrgBrandingAdminLogic.userFacingError(error)
        }
    }

    @MainActor
    private func testConnection() async {
        guard let token = session.accessToken else { return }
        testing = true
        errorMessage = nil
        statusMessage = nil
        defer { testing = false }
        do {
            let result = try await LMSAPI.testAIProviderSettings(accessToken: token)
            let name = result.provider ?? provider
            let ms = result.latencyMs.map { String(format: "%.0f" , $0) } ?? "?"
            let preview = result.responsePreview ?? "OK"
            statusMessage = L.format(
                "mobile.admin.orgBranding.provider.testSuccess",
                name,
                ms,
                preview
            )
        } catch {
            errorMessage = OrgBrandingAdminLogic.userFacingError(error)
        }
    }
}
