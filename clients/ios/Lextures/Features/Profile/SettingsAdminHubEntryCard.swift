import SwiftUI

struct SettingsAdminHubEntryCard: View {
    @Environment(AppShellModel.self) private var shell

    var body: some View {
        if SettingsMenuLogic.shouldShowHubEntry(
            features: shell.platformFeatures,
            permissions: shell.permissions
        ) {
            LMSCard {
                SettingsNavigationRow(
                    route: SettingsAdminHubRoute(),
                    systemImage: "slider.horizontal.3",
                    title: L.text("mobile.settings.menu.title"),
                    subtitle: L.text("mobile.settings.menu.entry.subtitle")
                )
            }
        }
    }
}
