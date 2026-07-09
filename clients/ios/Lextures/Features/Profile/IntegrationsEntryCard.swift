import SwiftUI

/// Integrations entry in profile settings (M14.1).
struct IntegrationsEntryCard: View {
    @Environment(AppShellModel.self) private var shell

    var body: some View {
        if AccountIntegrationsLogic.integrationsEnabled(shell.platformFeatures) {
            LMSCard {
                SettingsNavigationRow(
                    route: IntegrationsRoute(),
                    systemImage: "link.circle",
                    title: L.text("mobile.integrations.title"),
                    subtitle: L.text("mobile.integrations.entry.subtitle")
                )
            }
        }
    }
}