import SwiftUI

/// Org structure & terms admin entry in profile settings (M14.4).
struct OrgStructureAdminEntryCard: View {
    @Environment(AppShellModel.self) private var shell

    var body: some View {
        if OrgStructureAdminLogic.shouldShowEntry(
            features: shell.platformFeatures,
            permissions: shell.permissions
        ) {
            LMSCard {
                SettingsNavigationRow(
                    route: OrgStructureAdminRoute(),
                    systemImage: "building.2.crop.circle",
                    title: L.text("mobile.admin.orgStructure.title"),
                    subtitle: L.text("mobile.admin.orgStructure.entry.subtitle")
                )
            }
        }
    }
}
