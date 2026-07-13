import SwiftUI

struct OrgBrandingAdminRoute: Hashable {}

/// Host for org branding, AI governance, and AI provider settings (M14.5).
struct OrgBrandingAdminView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme

    @State private var orgId = ""
    @State private var branding = OrgBrandingResponse()
    @State private var governance: AIGovernanceConfig?
    @State private var providerSettings: AIProviderSettings?
    @State private var governanceAvailable = true
    @State private var providerAvailable = true
    @State private var loading = true
    @State private var errorMessage: String?
    @State private var statusMessage: String?

    private var features: MobilePlatformFeatures { shell.platformFeatures }

    var body: some View {
        Group {
            if !OrgBrandingAdminLogic.canView(features: features, permissions: shell.permissions) {
                accessDenied
            } else {
                content
            }
        }
        .navigationTitle(L.text("mobile.admin.orgBranding.title"))
        .navigationBarTitleDisplayMode(.inline)
        .refreshable { await load() }
        .task { await load() }
    }

    private var accessDenied: some View {
        LMSEmptyState(
            systemImage: "lock.fill",
            title: L.text("mobile.admin.orgBranding.accessDeniedTitle"),
            message: L.text("mobile.admin.orgBranding.accessDeniedMessage")
        )
        .padding(16)
    }

    private var content: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    Text(L.text("mobile.admin.orgBranding.description"))
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                    if let errorMessage {
                        LMSErrorBanner(message: errorMessage)
                    }
                    if let statusMessage {
                        Text(statusMessage)
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.brandTeal)
                            .accessibilityLabel(statusMessage)
                    }

                    if loading && orgId.isEmpty {
                        LMSSkeletonList(count: 3)
                    } else if orgId.isEmpty {
                        LMSEmptyState(
                            systemImage: "building.2",
                            title: L.text("mobile.admin.orgBranding.noOrgTitle"),
                            message: L.text("mobile.admin.orgBranding.noOrgMessage")
                        )
                    } else {
                        OrgBrandingView(
                            orgId: orgId,
                            branding: $branding,
                            statusMessage: $statusMessage,
                            errorMessage: $errorMessage
                        )
                        AiGovernanceView(
                            config: $governance,
                            statusMessage: $statusMessage,
                            errorMessage: $errorMessage,
                            available: governanceAvailable
                        )
                        AiProviderSettingsView(
                            settings: $providerSettings,
                            statusMessage: $statusMessage,
                            errorMessage: $errorMessage,
                            available: providerAvailable
                        )
                    }
                }
                .padding(16)
            }
        }
    }

    @MainActor
    private func load() async {
        guard let token = session.accessToken else { return }
        loading = true
        errorMessage = nil
        defer { loading = false }

        if orgId.isEmpty {
            orgId = OrgBrandingAdminLogic.resolveOrgId(
                accessToken: token,
                courses: []
            ) ?? ""
        }
        guard !orgId.isEmpty else { return }

        do {
            branding = try await LMSAPI.fetchOrgBranding(orgId: orgId, accessToken: token)
        } catch {
            errorMessage = OrgBrandingAdminLogic.userFacingError(error)
        }

        do {
            governance = try await LMSAPI.fetchAIGovernanceConfig(accessToken: token)
            governanceAvailable = true
        } catch {
            if let api = error as? APIError, case let .httpStatus(code, _) = api, code == 403 || code == 404 {
                governanceAvailable = false
                governance = nil
            } else {
                governanceAvailable = false
                governance = nil
            }
        }

        do {
            providerSettings = try await LMSAPI.fetchAIProviderSettings(accessToken: token)
            providerAvailable = true
        } catch {
            if let api = error as? APIError, case let .httpStatus(code, _) = api, code == 403 || code == 404 {
                providerAvailable = false
                providerSettings = nil
            } else {
                providerAvailable = false
                providerSettings = nil
            }
        }
    }
}
