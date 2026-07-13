import SwiftUI

struct OrgStructureAdminRoute: Hashable {}

enum OrgStructureAdminSection: String, CaseIterable, Identifiable {
    case organizations
    case orgUnits
    case terms

    var id: String { rawValue }

    var title: String {
        switch self {
        case .organizations: return L.text("mobile.admin.orgStructure.tab.organizations")
        case .orgUnits: return L.text("mobile.admin.orgStructure.tab.orgUnits")
        case .terms: return L.text("mobile.admin.orgStructure.tab.terms")
        }
    }
}

/// Host for organizations, org units, and academic terms admin (M14.4).
struct OrgStructureAdminView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme

    @State private var selectedSection: OrgStructureAdminSection = .orgUnits
    @State private var selectedOrgId = ""
    @State private var organizations: [AdminOrgRow] = []

    private var features: MobilePlatformFeatures { shell.platformFeatures }
    private var permissions: [String] { shell.permissions }

    private var availableSections: [OrgStructureAdminSection] {
        var sections: [OrgStructureAdminSection] = []
        if OrgStructureAdminLogic.canManageOrganizations(permissions: permissions) {
            sections.append(.organizations)
        }
        if OrgStructureAdminLogic.canManageOrgUnitsAndTerms(permissions: permissions) {
            sections.append(.orgUnits)
            sections.append(.terms)
        }
        return sections
    }

    var body: some View {
        Group {
            if !OrgStructureAdminLogic.canView(features: features, permissions: permissions) {
                accessDenied
            } else {
                content
            }
        }
        .navigationTitle(L.text("mobile.admin.orgStructure.title"))
        .navigationBarTitleDisplayMode(.inline)
        .task { await bootstrap() }
        .onChange(of: availableSections) { _, sections in
            if !sections.contains(selectedSection), let first = sections.first {
                selectedSection = first
            }
        }
    }

    private var accessDenied: some View {
        LMSEmptyState(
            systemImage: "lock.fill",
            title: L.text("mobile.admin.orgStructure.accessDeniedTitle"),
            message: L.text("mobile.admin.orgStructure.accessDeniedMessage")
        )
        .padding(16)
    }

    private var content: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            VStack(alignment: .leading, spacing: 0) {
                if availableSections.count > 1 {
                    Picker("", selection: $selectedSection) {
                        ForEach(availableSections) { section in
                            Text(section.title).tag(section)
                        }
                    }
                    .pickerStyle(.segmented)
                    .padding(.horizontal, 16)
                    .padding(.top, 8)
                }

                Group {
                    switch selectedSection {
                    case .organizations:
                        OrganizationsAdminPanel(organizations: organizations)
                    case .orgUnits:
                        OrgStructureView(
                            orgId: selectedOrgId,
                            organizations: organizations,
                            canPickOrg: OrgStructureAdminLogic.canManageOrganizations(permissions: permissions),
                            selectedOrgId: $selectedOrgId
                        )
                    case .terms:
                        TermsAdminView(
                            orgId: selectedOrgId,
                            organizations: organizations,
                            canPickOrg: OrgStructureAdminLogic.canManageOrganizations(permissions: permissions),
                            selectedOrgId: $selectedOrgId
                        )
                    }
                }
            }
        }
    }

    @MainActor
    private func bootstrap() async {
        if let first = availableSections.first {
            selectedSection = first
        }
        if selectedOrgId.isEmpty {
            selectedOrgId = OrgStructureAdminLogic.resolveOrgId(
                accessToken: session.accessToken,
                courses: []
            ) ?? ""
        }
        guard OrgStructureAdminLogic.canManageOrganizations(permissions: permissions),
              let token = session.accessToken else { return }
        do {
            organizations = try await LMSAPI.fetchAdminOrganizations(accessToken: token)
            if selectedOrgId.isEmpty, let first = organizations.first?.id {
                selectedOrgId = first
            }
        } catch {
            organizations = []
        }
    }
}

private struct OrganizationsAdminPanel: View {
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.openURL) private var openURL

    let organizations: [AdminOrgRow]

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                Text(L.text("mobile.admin.orgStructure.description"))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                webLinkCard

                if organizations.isEmpty {
                    LMSEmptyState(
                        systemImage: "building.2",
                        title: L.text("mobile.admin.orgStructure.organizations.emptyTitle"),
                        message: L.text("mobile.admin.orgStructure.organizations.emptyMessage")
                    )
                } else {
                    ForEach(organizations) { org in
                        LMSCard {
                            VStack(alignment: .leading, spacing: 6) {
                                Text(org.name)
                                    .font(.headline)
                                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                                Text(org.slug)
                                    .font(.caption)
                                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                                HStack(spacing: 12) {
                                    Text(org.status.capitalized)
                                    if let users = org.userCount {
                                        Text(L.format("mobile.admin.orgStructure.organizations.users", Int(users)))
                                    }
                                    if let courses = org.courseCount {
                                        Text(L.format("mobile.admin.orgStructure.organizations.courses", Int(courses)))
                                    }
                                }
                                .font(.caption)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            }
                            .frame(maxWidth: .infinity, alignment: .leading)
                        }
                    }
                }
            }
            .padding(16)
        }
    }

    private var webLinkCard: some View {
        Button {
            openURL(AppConfiguration.webURL(path: OrgStructureAdminLogic.webOrganizationsPath()))
        } label: {
            LMSCard {
                HStack(spacing: 10) {
                    Image(systemName: "safari")
                        .foregroundStyle(LexturesTheme.brandTeal)
                    VStack(alignment: .leading, spacing: 2) {
                        Text(L.text("mobile.admin.orgStructure.webTitle"))
                            .font(.subheadline.weight(.semibold))
                            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                        Text(L.text("mobile.admin.orgStructure.webHint"))
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                    Spacer(minLength: 0)
                    Image(systemName: "arrow.up.right")
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
            }
        }
        .buttonStyle(.plain)
    }
}
