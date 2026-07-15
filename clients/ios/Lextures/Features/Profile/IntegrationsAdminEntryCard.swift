import SwiftUI

struct IntegrationsAdminEntryCard: View {
    @Environment(AppShellModel.self) private var shell

    var body: some View {
        if IntegrationsAdminLogic.shouldShowEntry(
            features: shell.platformFeatures,
            permissions: shell.permissions
        ) {
            LMSCard {
                SettingsNavigationRow(
                    route: IntegrationsAdminRoute(),
                    systemImage: "link.badge.plus",
                    title: L.text("mobile.admin.integrations.hub.title"),
                    subtitle: L.text("mobile.admin.integrations.hub.entry.subtitle")
                )
            }
        }
    }
}
