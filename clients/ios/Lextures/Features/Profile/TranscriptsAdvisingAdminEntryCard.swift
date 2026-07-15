import SwiftUI

struct TranscriptsAdvisingAdminEntryCard: View {
    @Environment(AppShellModel.self) private var shell

    var body: some View {
        if TranscriptsAdvisingAdminLogic.shouldShowEntry(
            features: shell.platformFeatures,
            permissions: shell.permissions
        ) {
            LMSCard {
                SettingsNavigationRow(
                    route: TranscriptsAdvisingAdminRoute(),
                    systemImage: "doc.text.magnifyingglass",
                    title: L.text("mobile.admin.transcriptsAdvising.hub.title"),
                    subtitle: L.text("mobile.admin.transcriptsAdvising.hub.entry.subtitle")
                )
            }
        }
    }
}
