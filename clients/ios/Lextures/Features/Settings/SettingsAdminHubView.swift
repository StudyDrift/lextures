import SwiftUI

struct SettingsAdminHubRoute: Hashable {}

struct SettingsAdminHubView: View {
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme
    @State private var searchText = ""
    @State private var pendingPage: SettingsMenuLogic.ItemId?

    private var canView: Bool {
        SettingsMenuLogic.shouldShowHubEntry(features: shell.platformFeatures, permissions: shell.permissions)
    }

    private var groups: [SettingsMenuLogic.MenuGroup] {
        SettingsMenuLogic.visibleGroups(features: shell.platformFeatures, permissions: shell.permissions, query: searchText)
    }

    var body: some View {
        Group { if canView { content } else { accessDenied } }
        .navigationTitle(L.text("mobile.settings.menu.title"))
        .navigationBarTitleDisplayMode(.inline)
        .searchable(text: $searchText, prompt: L.text("mobile.settings.menu.search.prompt"))
        .navigationDestination(item: $pendingPage) { destination(for: $0) }
        .onAppear { openPendingAuditLogIfNeeded() }
    }

    private var accessDenied: some View {
        LMSEmptyState(systemImage: "lock.fill", title: L.text("mobile.settings.menu.accessDenied.title"), message: L.text("mobile.settings.menu.accessDenied.message")).padding(16)
    }

    private var content: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()
            ScrollView {
                VStack(alignment: .leading, spacing: 20) {
                    Text(L.text("mobile.settings.menu.description")).font(.subheadline).foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    if groups.isEmpty {
                        LMSEmptyState(systemImage: "slider.horizontal.3", title: L.text("mobile.settings.menu.empty.title"), message: L.text("mobile.settings.menu.empty.message"))
                    } else {
                        ForEach(groups) { groupSection($0) }
                    }
                }
                .padding(16)
            }
        }
    }

    private func groupSection(_ group: SettingsMenuLogic.MenuGroup) -> some View {
        VStack(alignment: .leading, spacing: 12) {
            LMSSectionHeader(title: L.text(group.titleKey), systemImage: group.id.systemImage)
            ForEach(group.items) { item in
                LMSCard {
                    SettingsDestinationRow(systemImage: item.systemImage, title: L.text(item.titleKey), subtitle: L.text(item.subtitleKey)) {
                        destination(for: item.id)
                    }
                }
            }
        }
    }

    @ViewBuilder
    private func destination(for item: SettingsMenuLogic.ItemId) -> some View {
        switch item {
        case .platformSettings: PlatformSettingsView()
        case .orgStructure: OrgStructureAdminView()
        case .orgBranding: OrgBrandingAdminView()
        case .rolesPermissions: RolesPermissionsAdminView()
        case .people: PeopleAdminView()
        case .courses: CoursesAdminView()
        case .archivedCourses: ArchivedCoursesAdminView()
        case .aiAdmin: AiAdminHubView()
        case .transcriptsAdvising: TranscriptsAdvisingAdminView()
        case .integrations: IntegrationsAdminView()
        case .auditLog: AuditLogAdminView()
        case .boardsGovernance: BoardsGovernanceAdminView()
        }
    }

    private func openPendingAuditLogIfNeeded() {
        guard shell.consumePendingSettingsAdminPage() == .auditLog else { return }
        guard AuditLogAdminLogic.canView(features: shell.platformFeatures, permissions: shell.permissions) else { return }
        pendingPage = .auditLog
    }
}
