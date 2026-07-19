import SwiftUI

struct SettingsAdminHubRoute: Hashable {}

/// MOB.3 Settings/Admin hub — grouped, searchable menu mirroring web admin nav.
struct SettingsAdminHubView: View {
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme
    @State private var searchText = ""
    @State private var pendingPage: SettingsMenuLogic.ItemId?

    private var canView: Bool {
        SettingsMenuLogic.shouldShowHubEntry(
            features: shell.platformFeatures,
            permissions: shell.permissions
        )
    }

    private var groups: [SettingsMenuLogic.MenuGroup] {
        SettingsMenuLogic.visibleGroups(
            features: shell.platformFeatures,
            permissions: shell.permissions,
            query: searchText
        )
    }

    var body: some View {
        Group {
            if canView { content } else { accessDenied }
        }
        .navigationTitle(L.text("mobile.settings.menu.title"))
        .navigationBarTitleDisplayMode(.inline)
        .searchable(
            text: $searchText,
            prompt: L.text("mobile.settings.menu.search.prompt")
        )
        .navigationDestination(for: SettingsMenuLogic.ItemId.self) { item in
            destination(for: item)
        }
        .navigationDestination(item: $pendingPage) { item in
            destination(for: item)
        }
        .onAppear { openPendingAuditLogIfNeeded() }
    }

    private var accessDenied: some View {
        LMSEmptyState(
            systemImage: "lock.fill",
            title: L.text("mobile.settings.menu.accessDenied.title"),
            message: L.text("mobile.settings.menu.accessDenied.message")
        )
        .padding(16)
    }

    private var content: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()
            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    Text(L.text("mobile.settings.menu.description"))
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                    if groups.isEmpty {
                        LMSEmptyState(
                            systemImage: "slider.horizontal.3",
                            title: L.text("mobile.settings.menu.empty.title"),
                            message: L.text("mobile.settings.menu.empty.message")
                        )
                    } else {
                        ForEach(groups) { group in
                            groupCard(group)
                        }
                    }
                }
                .padding(16)
            }
        }
    }

    private func groupCard(_ group: SettingsMenuLogic.MenuGroup) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(L.text(group.titleKey))
                .font(.footnote.weight(.semibold))
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                .textCase(.uppercase)
                .accessibilityAddTraits(.isHeader)

            LMSCard {
                VStack(spacing: 0) {
                    ForEach(Array(group.items.enumerated()), id: \.element.id) { index, item in
                        if index > 0 {
                            Divider().padding(.leading, 44)
                        }
                        SettingsNavigationRow(
                            route: item.id,
                            systemImage: item.systemImage,
                            title: L.text(item.titleKey),
                            subtitle: L.text(item.subtitleKey)
                        )
                    }
                }
            }
        }
    }

    @ViewBuilder
    private func destination(for item: SettingsMenuLogic.ItemId) -> some View {
        switch item {
        case .platformSettings:
            PlatformSettingsView()
        case .orgStructure:
            OrgStructureAdminView()
        case .orgBranding:
            OrgBrandingAdminView()
        case .rolesPermissions:
            RolesPermissionsAdminView()
        case .people:
            PeopleAdminView()
        case .archivedCourses:
            ArchivedCoursesAdminView()
        case .aiAdmin:
            AiAdminHubView()
        case .transcriptsAdvising:
            TranscriptsAdvisingAdminView()
        case .integrations:
            IntegrationsAdminView()
        case .auditLog:
            AuditLogAdminView()
        case .boardsGovernance:
            BoardsGovernanceAdminView()
        }
    }

    private func openPendingAuditLogIfNeeded() {
        guard shell.consumePendingSettingsAdminPage() == .auditLog else { return }
        guard AuditLogAdminLogic.canView(
            features: shell.platformFeatures,
            permissions: shell.permissions
        ) else { return }
        pendingPage = .auditLog
    }
}
