import SwiftUI

struct PlatformSettingsAdminEntryCard: View {
    @Environment(AppShellModel.self) private var shell

    var body: some View {
        if PlatformSettingsAdminLogic.shouldShowEntry(
            features: shell.platformFeatures,
            permissions: shell.permissions
        ) {
            LMSCard {
                SettingsNavigationRow(
                    route: PlatformSettingsAdminRoute(),
                    systemImage: "switch.2",
                    title: L.text("mobile.admin.platform.title"),
                    subtitle: L.text("mobile.admin.platform.entry.subtitle")
                )
            }
        }
    }
}

