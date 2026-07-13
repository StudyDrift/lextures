import SwiftUI

/// Roles & permissions admin entry in profile settings (M14.2).
struct RolesPermissionsAdminEntryCard: View {
    @Environment(AppShellModel.self) private var shell

    var body: some View {
        if RolesPermissionsAdminLogic.shouldShowEntry(
            features: shell.platformFeatures,
            permissions: shell.permissions
        ) {
            LMSCard {
                SettingsNavigationRow(
                    route: RolesPermissionsAdminRoute(),
                    systemImage: "person.2.badge.key",
                    title: L.text("mobile.admin.roles.title"),
                    subtitle: L.text("mobile.admin.roles.entry.subtitle")
                )
            }
        }
    }
}
