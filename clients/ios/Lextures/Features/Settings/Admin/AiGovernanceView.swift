import SwiftUI

/// AI governance policy toggles and model limits (M14.5).
struct AiGovernanceView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme

    @State private var available = false
    @State private var enabled: [String: Bool] = [:]
    @State private var allowedModelsText = ""
    @State private var loading = true
    @State private var saveStatus: OrgBrandingAdminLogic.SaveStatus = .idle

    var body: some View {
        Group {
            if available {
                LMSCard {
                    VStack(alignment: .leading, spacing: 16) {
                        Text(L.text("mobile.admin.orgBranding.aiGovernance.title"))
                            .font(.headline)
                            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))

                        Text(L.text("mobile.admin.orgBranding.aiGovernance.intro"))
                            .font(.subheadline)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                        if loading {
                            ProgressView()
                                .frame(maxWidth: .infinity)
                        } else {
                            featureToggles
                            allowedModelsField
                            saveSection
                        }
                    }
                }
            }
        }
        .task { await load() }
    }

    private var featureToggles: some View {
        VStack(alignment: .leading, spacing: 10) {
            ForEach(OrgBrandingAdminLogic.aiFeatureKeys, id: \.key) { feature in
                Toggle(isOn: Binding(
                    get: { enabled[feature.key] != false },
                    set: { enabled[feature.key] = $0 }
                )) {
                    Text(L.dynamicText(feature.labelKey))
                        .font(.subheadline)
                }
            }
        }
    }

    private var allowedModelsField: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text(L.text("mobile.admin.orgBranding.aiGovernance.allowedModels"))
                .font(.subheadline.weight(.semibold))
            TextEditor(text: $allowedModelsText)
                .font(.system(.caption, design: .monospaced))
                .frame(minHeight: 88)
                .overlay(
                    RoundedRectangle(cornerRadius: 8)
                        .stroke(LexturesTheme.textSecondary(for: colorScheme).opacity(0.3))
                )
            Text(L.text("mobile.admin.orgBranding.aiGovernance.allowedModelsHint"))
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
        }
    }

    private var saveSection: some View {
        VStack(alignment: .leading, spacing: 8) {
            Button {
                Task { await save() }
            } label: {
                if case .saving = saveStatus {
                    ProgressView()
                        .frame(maxWidth: .infinity)
                } else {
                    Text(L.text("mobile.admin.orgBranding.aiGovernance.save"))
                        .frame(maxWidth: .infinity)
                }
            }
            .buttonStyle(.borderedProminent)
            .tint(LexturesTheme.brandTeal)
            .disabled(loading || saveStatus == .saving)

            switch saveStatus {
            case .saved:
                Text(L.text("mobile.admin.orgBranding.aiGovernance.saved"))
                    .font(.caption)
                    .foregroundStyle(.green)
            case .error(let message):
                Text(message)
                    .font(.caption)
                    .foregroundStyle(.red)
            default:
                EmptyView()
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
            let response = try await LMSAPI.fetchAiConfig(accessToken: token)
            available = true
            enabled = response.featuresEnabled ?? [:]
            allowedModelsText = OrgBrandingAdminLogic.allowedModelsText(response.allowedModels)
            saveStatus = .idle
        } catch let error as APIError {
            if case let .httpStatus(code, _) = error, code == 404 || code == 403 {
                available = false
            } else {
                available = true
                saveStatus = .error(OrgBrandingAdminLogic.userFacingError(
                    error,
                    fallbackKey: "mobile.admin.orgBranding.aiGovernance.loadError"
                ))
            }
        } catch {
            available = true
            saveStatus = .error(OrgBrandingAdminLogic.userFacingError(
                error,
                fallbackKey: "mobile.admin.orgBranding.aiGovernance.loadError"
            ))
        }
    }

    private func save() async {
        guard let token = session.accessToken else { return }
        saveStatus = .saving
        let request = OrgBrandingAdminLogic.buildAiConfigSaveRequest(
            enabled: enabled,
            allowedModelsText: allowedModelsText
        )
        do {
            let response = try await LMSAPI.putAiConfig(body: request, accessToken: token)
            enabled = response.featuresEnabled ?? enabled
            allowedModelsText = OrgBrandingAdminLogic.allowedModelsText(response.allowedModels)
            saveStatus = .saved
        } catch {
            saveStatus = .error(OrgBrandingAdminLogic.userFacingError(
                error,
                fallbackKey: "mobile.admin.orgBranding.aiGovernance.saveError"
            ))
        }
    }
}
