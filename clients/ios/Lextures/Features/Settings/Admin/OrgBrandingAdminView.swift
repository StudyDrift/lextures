import SwiftUI

struct OrgBrandingAdminRoute: Hashable {}

/// Host for org branding, AI governance, and AI provider admin (M14.5).
struct OrgBrandingAdminView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme

    @State private var orgId = ""
    @State private var loading = true
    @State private var loadError: String?

    private var features: MobilePlatformFeatures { shell.platformFeatures }
    private var permissions: [String] { shell.permissions }

    var body: some View {
        Group {
            if !OrgBrandingAdminLogic.canView(features: features, permissions: permissions) {
                accessDenied
            } else {
                content
            }
        }
        .navigationTitle(L.text("mobile.admin.orgBranding.title"))
        .navigationBarTitleDisplayMode(.inline)
        .task { await bootstrap() }
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

                    if let loadError {
                        LMSErrorBanner(message: loadError)
                    }

                    if loading {
                        LMSSkeletonList(count: 3)
                    } else if orgId.isEmpty {
                        LMSEmptyState(
                            systemImage: "building.2",
                            title: L.text("mobile.admin.orgBranding.noOrgTitle"),
                            message: L.text("mobile.admin.orgBranding.noOrgMessage")
                        )
                    } else {
                        OrgBrandingView(orgId: orgId)
                        AiGovernanceView()
                        AiProviderSettingsView()
                    }
                }
                .padding(16)
            }
        }
    }

    private func bootstrap() async {
        loading = true
        defer { loading = false }
        orgId = OrgBrandingAdminLogic.resolveOrgId(
            accessToken: session.accessToken,
            courses: []
        ) ?? ""
        loadError = nil
    }
}
