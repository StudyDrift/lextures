import SwiftUI

struct AiAdminEntryCard: View {
    @Environment(AppShellModel.self) private var shell

    var body: some View {
        if AiModelsAdminLogic.shouldShowEntry(
            features: shell.platformFeatures,
            permissions: shell.permissions
        ) {
            LMSCard {
                SettingsNavigationRow(
                    route: AiAdminHubRoute(),
                    systemImage: "sparkles",
                    title: L.text("mobile.admin.ai.hub.title"),
                    subtitle: L.text("mobile.admin.ai.hub.entry.subtitle")
                )
            }
        }
    }
}
