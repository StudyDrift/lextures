import SwiftUI

/// AI governance card: feature toggles and allowed models (M14.5).
struct AiGovernanceView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme

    @Binding var config: AIGovernanceConfig?
    @Binding var statusMessage: String?
    @Binding var errorMessage: String?
    let available: Bool

    @State private var enabled: [String: Bool] = [:]
    @State private var allowedModelsText = ""
    @State private var saving = false

    var body: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 14) {
                Text(L.text("mobile.admin.orgBranding.ai.title"))
                    .font(.headline)
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                Text(L.text("mobile.admin.orgBranding.ai.intro"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                if !available {
                    Text(L.text("mobile.admin.orgBranding.ai.unavailable"))
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                } else {
                    ForEach(OrgBrandingAdminLogic.featureKeys, id: \.key) { item in
                        Toggle(isOn: binding(for: item.key)) {
                            Text(L.dynamicText(item.labelKey))
                                .font(.subheadline)
                                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                        }
                        .tint(LexturesTheme.brandTeal)
                        .frame(minHeight: 44)
                    }

                    VStack(alignment: .leading, spacing: 6) {
                        Text(L.text("mobile.admin.orgBranding.ai.allowedModels"))
                            .font(.subheadline.weight(.medium))
                            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                        TextEditor(text: $allowedModelsText)
                            .font(.caption.monospaced())
                            .frame(minHeight: 88)
                            .padding(8)
                            .background(LexturesTheme.sceneBackground(for: colorScheme), in: RoundedRectangle(cornerRadius: 10))
                            .accessibilityLabel(L.text("mobile.admin.orgBranding.ai.allowedModels"))
                        Text(L.text("mobile.admin.orgBranding.ai.allowedModelsHint"))
                            .font(.caption2)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }

                    Button {
                        Task { await save() }
                    } label: {
                        if saving {
                            ProgressView().frame(maxWidth: .infinity)
                        } else {
                            Text(L.text("mobile.admin.orgBranding.ai.save"))
                                .frame(maxWidth: .infinity)
                        }
                    }
                    .buttonStyle(.borderedProminent)
                    .tint(LexturesTheme.brandTeal)
                    .disabled(saving)
                    .frame(minHeight: 44)
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
        .onAppear { apply(config) }
        .onChange(of: config) { _, next in apply(next) }
    }

    private func binding(for key: String) -> Binding<Bool> {
        Binding(
            get: { OrgBrandingAdminLogic.isFeatureEnabled(enabled, key: key) },
            set: { enabled[key] = $0 }
        )
    }

    private func apply(_ data: AIGovernanceConfig?) {
        enabled = data?.featuresEnabled ?? [:]
        allowedModelsText = OrgBrandingAdminLogic.allowedModelsText(data?.allowedModels)
    }

    @MainActor
    private func save() async {
        guard let token = session.accessToken else { return }
        saving = true
        errorMessage = nil
        statusMessage = nil
        defer { saving = false }
        do {
            let body = OrgBrandingAdminLogic.aiConfigPutBody(
                enabled: enabled,
                allowedModelsText: allowedModelsText
            )
            let saved = try await LMSAPI.putAIGovernanceConfig(body: body, accessToken: token)
            config = saved
            apply(saved)
            statusMessage = L.text("mobile.admin.orgBranding.ai.saved")
        } catch {
            errorMessage = OrgBrandingAdminLogic.userFacingError(error)
        }
    }
}
